package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scratch(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("testdata", prefix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestCheckFailsWhenNothingIsChecked(t *testing.T) {
	for _, pattern := range []string{".", "./nothing/..."} {
		if err := check([]string{pattern}, io.Discard); err == nil {
			t.Errorf("check %s passed with no Style IR under it", pattern)
		}
	}
}

func TestCheckScratchCopy(t *testing.T) {
	dir := scratch(t, "scratch")
	app, err := os.ReadFile(filepath.Join("testdata", "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join("testdata", "app", "twir_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	crlf := strings.ReplaceAll(strings.ReplaceAll(string(generated), "\r\n", "\n"), "\n", "\r\n")
	if err := os.WriteFile(filepath.Join(dir, "twir_gen_test.go"), []byte(crlf), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), app, 0o600); err != nil {
		t.Fatal(err)
	}
	pattern := "./" + filepath.ToSlash(dir)
	var out strings.Builder
	if err := check([]string{pattern}, &out); err != nil || !strings.HasPrefix(out.String(), "fresh ") {
		t.Fatalf("a fresh CRLF IR named twir_gen_test.go: %v\n%s", err, out.String())
	}
	touched := strings.Replace(string(app), "text-red-500", "text-blue-500", 1)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(touched), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := check([]string{pattern}, &out); err == nil || !strings.Contains(out.String(), "stale ") || !strings.Contains(out.String(), "twir_gen_test.go") {
		t.Fatalf("a class edited in main.go: want the IR named stale and an error, got %v\n%s", err, out.String())
	}
}

func TestDriveMainPackage(t *testing.T) {
	out := scratch(t, "frames")
	if err := drive([]string{filepath.Join("testdata", "hello.twd"), "--out", out, "--", "./testdata/app"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"start": " a package-level drive\n\n", "short": " a package-level drive\n"} {
		text, err := os.ReadFile(filepath.Join(out, name+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(text) != want {
			t.Errorf("frame %s is %q, want %q", name, text, want)
		}
		if _, err := os.Stat(filepath.Join(out, name+".ansi")); err != nil {
			t.Error(err)
		}
	}
}

func TestDriveFailures(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.twd")
	if err := os.WriteFile(bad, []byte("size 30x2\njump 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{filepath.Join("testdata", "hello.twd"), "."}, "App"},
		{[]string{bad, "./testdata/app"}, "line 2"},
		{[]string{filepath.Join("testdata", "missing.twd"), "./testdata/app"}, "missing.twd"},
		{[]string{}, "want a script"},
		{[]string{"--frames", "x"}, "not defined: -frames"},
	} {
		if err := drive(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("drive %q: want an error naming %q, got %v", c.args, c.want, err)
		}
	}
}

func TestDoctorRefusesAPipe(t *testing.T) {
	var out strings.Builder
	if err := doctor(nil, &out); err == nil || out.Len() > 0 {
		t.Errorf("doctor on a piped stdout: want an error and nothing printed, got %v and %q", err, out.String())
	}
}
