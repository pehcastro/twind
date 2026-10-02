package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestDevWorkDirUnderTheProject(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, ".twind", "cache", "dev")
	dead, live := filepath.Join(base, "999999991"), filepath.Join(base, strconv.Itoa(os.Getppid()))
	for _, d := range []string{dead, live} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	work, err := devWorkDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(work) != base {
		t.Errorf("work dir %s, want it in %s", work, base)
	}
	if info, err := os.Stat(work); err != nil || !info.IsDir() {
		t.Errorf("work dir %s not made: %v", work, err)
	}
	if _, err := os.Stat(dead); err == nil {
		t.Errorf("%s of a session that is gone was kept", dead)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("%s of a running process was removed: %v", live, err)
	}
}

func TestPruneKeepsTheNewestBinaries(t *testing.T) {
	work := t.TempDir()
	var names []string
	at := time.Now().Add(-time.Hour)
	for i := range 6 {
		name := filepath.Join(work, executable("app-"+strconv.Itoa(i+1)))
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(name, at, at.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := os.WriteFile(filepath.Join(work, "state.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	prune(work, 3, names[0])
	var left []string
	entries, _ := os.ReadDir(work)
	for _, e := range entries {
		left = append(left, e.Name())
	}
	want := []string{filepath.Base(names[0]), filepath.Base(names[3]), filepath.Base(names[4]), filepath.Base(names[5]), "state.json"}
	slices.Sort(want)
	if !slices.Equal(left, want) {
		t.Errorf("left %v, want %v: the three newest, the spared running one, and no other file touched", left, want)
	}
}
