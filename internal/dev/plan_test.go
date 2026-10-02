package dev

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type planModule struct {
	t    *testing.T
	root string
	exes int
}

func (m *planModule) write(name, body string) {
	m.t.Helper()
	path := filepath.Join(m.root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		m.t.Fatal(err)
	}
}

func (m *planModule) exe() string {
	m.exes++
	name := filepath.Join(m.root, "out", "app"+strings.Repeat("x", m.exes))
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func (m *planModule) run(exe string) string {
	m.t.Helper()
	out, err := exec.Command(exe).CombinedOutput()
	if err != nil {
		m.t.Fatalf("%s: %v\n%s", exe, err, out)
	}
	return string(out)
}

func (m *planModule) goBuild() string {
	m.t.Helper()
	exe := m.exe()
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = m.root
	if out, err := cmd.CombinedOutput(); err != nil {
		m.t.Fatalf("go build: %v\n%s", err, out)
	}
	return exe
}

func newPlanModule(t *testing.T) *planModule {
	m := &planModule{t: t, root: t.TempDir()}
	m.write("go.mod", "module example.com/m\n\ngo 1.26\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"one\" }\n")
	m.write("mid/mid.go", "package mid\n\nimport \"example.com/m/leaf\"\n\nfunc Say() string { return \"mid \" + leaf.Value() }\n")
	m.write("other/other.go", "package other\n\nconst Name = \"other\"\n")
	m.write("data/data.go", "package data\n\nimport _ \"embed\"\n\n//go:embed data.txt\nvar Text string\n")
	m.write("data/data.txt", "first")
	m.write("main.go", "package main\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/m/data\"\n\t\"example.com/m/mid\"\n\t\"example.com/m/other\"\n)\n\nfunc main() { fmt.Println(mid.Say(), other.Name, data.Text) }\n")
	return m
}

func TestPlanRebuildsWhatDependsOnTheEdit(t *testing.T) {
	m := newPlanModule(t)
	m.goBuild()
	p, err := Capture(context.Background(), m.root, ".", filepath.Join(m.root, "plan"))
	if err != nil {
		t.Fatal(err)
	}
	step := func(why string, changed []string, want []string, output string) {
		t.Helper()
		exe := m.exe()
		steps, err := p.Build(context.Background(), changed, exe)
		if err != nil {
			t.Fatalf("%s: %v", why, err)
		}
		if len(steps) == 0 || steps[len(steps)-1].Package != "link" {
			t.Fatalf("%s: steps %v end without a link", why, steps)
		}
		var compiled []string
		for _, s := range steps[:len(steps)-1] {
			compiled = append(compiled, s.Package)
		}
		if !slices.Equal(slices.Sorted(slices.Values(compiled)), slices.Sorted(slices.Values(want))) {
			t.Errorf("%s: compiled %v, want %v", why, compiled, want)
		}
		if leaf, mid := slices.Index(compiled, "example.com/m/leaf"), slices.Index(compiled, "example.com/m/mid"); mid >= 0 && leaf > mid || slices.Contains(compiled, "example.com/m") && compiled[len(compiled)-1] != "example.com/m" {
			t.Errorf("%s: compiled out of dependency order: %v", why, compiled)
		}
		if got := m.run(exe); got != output {
			t.Errorf("%s: fast binary says %q, want %q", why, got, output)
		}
		if got := m.run(m.goBuild()); got != output {
			t.Errorf("%s: go build binary says %q, want %q", why, got, output)
		}
	}
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"two\" }\n")
	step("a body edit in leaf", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid", "example.com/m"}, "mid two other first\n")
	m.write("other/other.go", "// TWIR version=1 hash=abc\n\npackage other\n\nconst Name = \"other\"\n")
	step("a header comment changed, as a regenerated Style IR does", []string{filepath.Join(m.root, "other", "other.go")}, []string{"example.com/m/other", "example.com/m"}, "mid two other first\n")
	m.write("data/data.txt", "second")
	step("an embedded file, the package's export data unchanged", []string{filepath.Join(m.root, "data", "data.txt")}, []string{"example.com/m/data"}, "mid two other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string {\n\tdefer func() {}()\n\treturn \"rec\"\n}\n")
	step("leaf given a defer, so its body leaves the export data", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid", "example.com/m"}, "mid rec other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string {\n\tdefer func() {}()\n\treturn \"body\"\n}\n")
	step("a body edit no importer can see", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf"}, "mid body other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return undefinedName }\n")
	if _, err := p.Build(context.Background(), []string{filepath.Join(m.root, "leaf", "leaf.go")}, m.exe()); err == nil || !strings.Contains(err.Error(), "undefinedName") {
		t.Errorf("a compile error: got %v, want the compiler's message", err)
	}
	m.write("other/other.go", "package other\n\nconst Name = \"OTHER\"\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"three\" }\n")
	step("fixed, with the broken save still pending", []string{filepath.Join(m.root, "leaf", "leaf.go"), filepath.Join(m.root, "other", "other.go"), filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid", "example.com/m/other", "example.com/m"}, "mid three OTHER second\n")

	fast, slow := m.exe(), ""
	if _, err := p.Build(context.Background(), []string{filepath.Join(m.root, "main.go")}, fast); err != nil {
		t.Fatal(err)
	}
	slow = m.goBuild()
	a, errA := os.ReadFile(fast)
	b, errB := os.ReadFile(slow)
	if errA != nil || errB != nil {
		t.Fatal(errA, errB)
	}
	if len(a) != len(b) {
		t.Errorf("fast binary %d bytes, go build %d", len(a), len(b))
	}
	t.Logf("bytes differing from go build: %d of %d", differing(a, b), len(b))
	if n := differing(withoutBuildID(t, fast, a), withoutBuildID(t, slow, b)); n != 0 {
		t.Errorf("%d bytes differ from go build outside the build ID", n)
	}
}

func withoutBuildID(t *testing.T, exe string, body []byte) []byte {
	t.Helper()
	id, err := exec.Command("go", "tool", "buildid", exe).Output()
	if err != nil {
		t.Fatal(err)
	}
	full := strings.TrimSpace(string(id))
	out := body
	for _, part := range append(strings.Split(full, "/"), full) {
		out = bytes.ReplaceAll(out, []byte(part), bytes.Repeat([]byte{0}, len(part)))
	}
	return out
}

func TestPlanFallsBack(t *testing.T) {
	for _, tc := range []struct {
		why  string
		edit func(m *planModule) string
	}{
		{"an import added", func(m *planModule) string {
			m.write("leaf/leaf.go", "package leaf\n\nimport \"strings\"\n\nfunc Value() string { return strings.ToUpper(\"one\") }\n")
			return "leaf/leaf.go"
		}},
		{"a build constraint added", func(m *planModule) string {
			m.write("other/other.go", "//go:build !plan9\n\npackage other\n\nconst Name = \"other\"\n")
			return "other/other.go"
		}},
		{"a file added to a package", func(m *planModule) string {
			m.write("mid/more.go", "package mid\n\nconst More = 1\n")
			return "mid/more.go"
		}},
		{"a file removed", func(m *planModule) string {
			if err := os.Remove(filepath.Join(m.root, "other", "other.go")); err != nil {
				m.t.Fatal(err)
			}
			return "other/other.go"
		}},
		{"go.mod", func(m *planModule) string {
			m.write("go.mod", "module example.com/m\n\ngo 1.25\n")
			return "go.mod"
		}},
		{"a file in no package", func(m *planModule) string {
			m.write("notes.txt", "x")
			return "notes.txt"
		}},
	} {
		m := newPlanModule(t)
		m.goBuild()
		p, err := Capture(context.Background(), m.root, ".", filepath.Join(m.root, "plan"))
		if err != nil {
			t.Fatal(err)
		}
		changed := filepath.Join(m.root, tc.edit(m))
		var stale StaleError
		if _, err := p.Build(context.Background(), []string{changed}, m.exe()); !errors.As(err, &stale) {
			t.Errorf("%s: got %v, want a stale plan", tc.why, err)
		}
	}
}

func differing(a, b []byte) int {
	n := 0
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			n++
		}
	}
	return n + max(len(a), len(b)) - min(len(a), len(b))
}
