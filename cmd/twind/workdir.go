package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/twind-dev/twind/internal/dev"
)

func devWorkDir(root string) (string, error) {
	base := filepath.Join(root, ".twind", "cache", "dev")
	entries, _ := os.ReadDir(base)
	for _, e := range entries {
		if pid, err := strconv.Atoi(e.Name()); err == nil && pid != os.Getpid() && !dev.Alive(pid) {
			_ = os.RemoveAll(filepath.Join(base, e.Name()))
		}
	}
	work := filepath.Join(base, strconv.Itoa(os.Getpid()))
	return work, os.MkdirAll(work, 0o755)
}

func prune(work string, keep int, spare ...string) {
	type built struct {
		path string
		at   time.Time
	}
	var all []built
	entries, _ := os.ReadDir(work)
	for _, e := range entries {
		path := filepath.Join(work, e.Name())
		if info, err := e.Info(); err == nil && strings.HasPrefix(e.Name(), "app-") && !slices.Contains(spare, path) {
			all = append(all, built{path, info.ModTime()})
		}
	}
	slices.SortFunc(all, func(a, b built) int { return b.at.Compare(a.at) })
	for _, b := range all[min(keep, len(all)):] {
		_ = os.Remove(b.path)
	}
}
