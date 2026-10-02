package dev

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/twind-dev/twind/internal/dev/konst"
)

func TestWatchWakesBeforeThePoll(t *testing.T) {
	root := t.TempDir()
	files := []string{filepath.Join(root, "a.go"), filepath.Join(root, "sub", "b.go")}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("package a"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := Watch(ctx, files)
	time.Sleep(konst.Poll)
	var lags []time.Duration
	for i := range 6 {
		file := files[i%len(files)]
		if err := os.WriteFile(filepath.Join(root, "unwatched.txt"), []byte(strconv.Itoa(i)), 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(konst.Poll / 3)
		saved := time.Now()
		if err := os.WriteFile(file, []byte("package a\n"+strconv.Itoa(i)), 0o600); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-changes:
			lags = append(lags, time.Since(saved))
			if !slices.Equal(got, []string{file}) {
				t.Errorf("change %d: got %v, want %s", i, got, file)
			}
		case <-time.After(2 * konst.Poll):
			t.Fatalf("change %d: nothing after %v", i, 2*konst.Poll)
		}
	}
	slices.Sort(lags)
	if lags[len(lags)/2] > konst.Poll/4 {
		t.Errorf("median lag %v of %v, want under a quarter of the %v poll", lags[len(lags)/2], lags, konst.Poll)
	}
}
