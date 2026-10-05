package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNewRefuses(t *testing.T) {
	hidden := t.TempDir()
	if err := os.Mkdir(filepath.Join(hidden, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(t.TempDir(), "fresh")
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{hidden}, "is not empty"},
		{[]string{file}, "is a file"},
		{[]string{fresh, "-module", "Hello World"}, "invalid"},
		{[]string{filepath.Join(t.TempDir(), "has space")}, "invalid"},
		{[]string{}, "want one directory"},
		{[]string{fresh, fresh}, "want one directory"},
	} {
		if err := newApp(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("new %q: want an error naming %q, got %v", c.args, c.want, err)
		}
	}
	if entries, _ := os.ReadDir(hidden); len(entries) != 1 {
		t.Errorf("a refused new wrote into %s: %v", hidden, entries)
	}
	if _, err := os.Stat(fresh); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused new created %s: %v", fresh, err)
	}
}

func TestNewWritesOnlyTheTemplate(t *testing.T) {
	for version, checkout := range map[string]bool{"v0.0.0": true, "v0.5.0": false} {
		dir := t.TempDir()
		if err := writeApp(dir, map[string]string{"Module": "example.com/hello", "Name": "hello", "Version": version}); err != nil {
			t.Fatal(err)
		}
		var files []string
		err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
			if err == nil && !entry.IsDir() {
				files = append(files, filepath.ToSlash(strings.TrimPrefix(name, dir+string(filepath.Separator))))
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{".gitignore", "README.md", "go.mod", "main.go", "main_test.go", "testdata/demo.twd", "twir_gen.go"}; !slices.Equal(files, want) {
			t.Errorf("%s: wrote %q, want %q", version, files, want)
		}
		readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(readme), "no published version") {
			t.Errorf("%s: the README says Twind has no published version:\n%s", version, readme)
		}
		if got := strings.Contains(string(readme), "-replace github.com/pehcastro/twind="); got != checkout {
			t.Errorf("%s: the README points at a checkout: %t, want %t:\n%s", version, got, checkout, readme)
		}
		if mod, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.Contains(string(mod), "github.com/pehcastro/twind "+version) {
			t.Errorf("%s: go.mod does not require it:\n%s", version, mod)
		}
	}
}

func TestNewAppOutsideTheModule(t *testing.T) {
	twind, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "hello-app")
	if err := newApp([]string{dir, "-module", "example.com/hello"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"mod", "edit", "-replace", "github.com/pehcastro/twind=" + twind},
		{"mod", "tidy"},
		{"vet", "./..."},
		{"test", "-count=1", "./..."},
		{"tool", "twind", "check", "."},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=mod")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	src, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(src), "module example.com/hello\n") {
		t.Errorf("go.mod does not start with the module line:\n%s", src)
	}
	if err := newApp([]string{dir}, io.Discard); err == nil || !strings.Contains(err.Error(), "is not empty") {
		t.Errorf("a second new into %s: want it refused, got %v", dir, err)
	}
}
