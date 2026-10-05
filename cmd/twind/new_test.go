package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
