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
	step := func(why string, changed []string, want []string, output string) (string, []Step) {
		t.Helper()
		exe, steps, err := p.Build(context.Background(), changed, m.exe())
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
		return exe, steps
	}
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"two\" }\n")
	step("a body edit in leaf", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid", "example.com/m"}, "mid two other first\n")
	m.write("other/other.go", "// TWIR version=1 hash=abc\n\npackage other\n\nconst Name = \"other\"\n")
	step("a header comment changed, as a regenerated Style IR does", []string{filepath.Join(m.root, "other", "other.go")}, []string{"example.com/m/other", "example.com/m"}, "mid two other first\n")
	m.write("data/data.txt", "second")
	step("an embedded file, the package's export data unchanged", []string{filepath.Join(m.root, "data", "data.txt")}, []string{"example.com/m/data"}, "mid two other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"body\" }\n")
	step("a second body edit, the first left no inline body in the export data", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf"}, "mid body other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return Wrap(\"generic\") }\n\nfunc Wrap[T any](v T) T { return v }\n")
	step("a generic added", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid"}, "mid generic other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return Wrap(\"generic\") }\n\nfunc Wrap[T any](v T) T { return v + v }\n")
	if _, _, err := p.Build(context.Background(), []string{filepath.Join(m.root, "leaf", "leaf.go")}, m.exe()); err == nil {
		t.Error("v + v on any compiled")
	}
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return Wrap(\"generic\") }\n\nfunc Wrap[T ~string](v T) T { return v + v }\n")
	step("a generic's body, exported with or without -l", []string{filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid"}, "mid genericgeneric other second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return undefinedName }\n")
	if _, _, err := p.Build(context.Background(), []string{filepath.Join(m.root, "leaf", "leaf.go")}, m.exe()); err == nil || !strings.Contains(err.Error(), "undefinedName") {
		t.Errorf("a compile error: got %v, want the compiler's message", err)
	}
	m.write("other/other.go", "package other\n\nconst Name = \"OTHER\"\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"three\" }\n")
	three, _ := step("fixed, with the broken save still pending", []string{filepath.Join(m.root, "leaf", "leaf.go"), filepath.Join(m.root, "other", "other.go"), filepath.Join(m.root, "leaf", "leaf.go")}, []string{"example.com/m/leaf", "example.com/m/mid", "example.com/m/other", "example.com/m"}, "mid three OTHER second\n")

	leaf := filepath.Join(m.root, "leaf", "leaf.go")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"four\" }\n")
	step("four", []string{leaf}, []string{"example.com/m/leaf"}, "mid four OTHER second\n")
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"three\" }\n")
	exe, steps := step("an undo", []string{leaf}, []string{"example.com/m/leaf"}, "mid three OTHER second\n")
	if exe != three || !steps[0].Reused || !steps[1].Reused {
		t.Errorf("an undo: binary %s, steps %+v; want the archive and %s reused", exe, steps, three)
	}
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"three\" } /"+"* a note *"+"/\n")
	exe, steps = step("a note at a line's end", []string{leaf}, []string{"example.com/m/leaf"}, "mid three OTHER second\n")
	if exe != three || steps[0].Reused || !steps[1].Reused {
		t.Errorf("a note: binary %s, steps %+v; want leaf compiled and %s reused", exe, steps, three)
	}

	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string {\n\tpanic(\"boom\")\n}\n")
	exe, _, err = p.Build(context.Background(), []string{leaf}, m.exe())
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(exe).CombinedOutput(); err == nil || !bytes.Contains(out, []byte("leaf/leaf.go:4")) {
		t.Errorf("a panic in the fast binary: %v, want file and line in\n%s", err, out)
	}
}

func TestPlanWarmLeavesOnlyTheEditedPackage(t *testing.T) {
	m := newPlanModule(t)
	m.goBuild()
	p, err := Capture(context.Background(), m.root, ".", filepath.Join(m.root, "plan"))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.write("leaf/leaf.go", "package leaf\n\nfunc Value() string { return \"two\" }\n")
	exe, steps, err := p.Build(context.Background(), []string{filepath.Join(m.root, "leaf", "leaf.go")}, m.exe())
	if err != nil {
		t.Fatal(err)
	}
	var compiled []string
	for _, s := range steps {
		if !s.Reused {
			compiled = append(compiled, s.Package)
		}
	}
	if !slices.Equal(compiled, []string{"example.com/m/leaf", "link"}) {
		t.Errorf("first edit after a warm: compiled %v, want leaf and the link only; steps %+v", compiled, steps)
	}
	if got := m.run(exe); got != "mid two other first\n" {
		t.Errorf("fast binary says %q", got)
	}
}

func TestPlanNeverRewritesAnArchive(t *testing.T) {
	m := newPlanModule(t)
	m.goBuild()
	p, err := Capture(context.Background(), m.root, ".", filepath.Join(m.root, "plan"))
	if err != nil {
		t.Fatal(err)
	}
	pk := p.order[0]
	work := filepath.Join(m.root, "plan", "same")
	first, errA := p.compile(context.Background(), pk, work, rewrite(pk.importcfg, p.have))
	second, errB := p.compile(context.Background(), pk, work, rewrite(pk.importcfg, p.have))
	if errA != nil || errB != nil || first == second {
		t.Errorf("two compiles of %s wrote %s and %s (%v, %v); a remembered archive must keep its bytes", pk.path, first, second, errA, errB)
	}
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
		if _, _, err := p.Build(context.Background(), []string{changed}, m.exe()); !errors.As(err, &stale) {
			t.Errorf("%s: got %v, want a stale plan", tc.why, err)
		}
	}
}
