package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	style "github.com/twind-dev/twind/internal/konst/style"
)

func TestTailwindAsset(t *testing.T) {
	for _, c := range []struct{ goos, goarch, want string }{
		{"windows", "amd64", "tailwindcss-windows-x64.exe"},
		{"darwin", "arm64", "tailwindcss-macos-arm64"},
		{"linux", "amd64", "tailwindcss-linux-x64"},
		{"plan9", "386", ""},
		{"linux", "riscv64", ""},
	} {
		if got := tailwindAsset(c.goos, c.goarch); got != c.want {
			t.Errorf("%s/%s: %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestLocateTailwind(t *testing.T) {
	asset := tailwindAsset(runtime.GOOS, runtime.GOARCH)
	good := []byte("the pinned tailwind")
	sum := sha256.Sum256(good)
	sums := "aaaa  ./tailwindcss-other\n" + hex.EncodeToString(sum[:]) + "  ./" + asset + "\n"
	var requests atomic.Int32
	body := good
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v"+style.TailwindVersion+"/"+asset {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	release := server.URL + "/v"
	cache := t.TempDir()
	cached := filepath.Join(cache, "tailwind", style.TailwindVersion, asset)
	locate := func() (string, error) {
		return locateTailwind(t.TempDir(), cache, release, sums, io.Discard)
	}
	left := func(when string) {
		entries, _ := os.ReadDir(filepath.Dir(cached))
		for _, e := range entries {
			if e.Name() != "sha256sums.txt" {
				t.Errorf("%s: %s left in the cache", when, e.Name())
			}
		}
	}

	body = []byte("not the pinned tailwind")
	if _, err := locate(); err == nil || !strings.Contains(err.Error(), hex.EncodeToString(sum[:])) {
		t.Errorf("a wrong download: want an error naming the pinned sum, got %v", err)
	}
	left("a wrong download")

	body, status = good, http.StatusBadGateway
	if _, err := locate(); err == nil || !strings.Contains(err.Error(), "502") {
		t.Errorf("a 502: want an error naming it, got %v", err)
	}
	left("a 502")

	status = http.StatusOK
	requests.Store(0)
	stray := filepath.Join(filepath.Dir(cached), asset+".part123")
	if err := os.WriteFile(stray, []byte("a fetch killed half way"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := locate()
	if _, statErr := os.Stat(stray); statErr == nil {
		t.Errorf("%s left from a killed fetch is still there", stray)
	}
	if err != nil || path != cached {
		t.Fatalf("first fetch: %q %v, want %q", path, err, cached)
	}
	if got, _ := os.ReadFile(path); string(got) != string(good) {
		t.Errorf("fetched %q", got)
	}
	if listed, _ := os.ReadFile(filepath.Join(filepath.Dir(cached), "sha256sums.txt")); !strings.Contains(string(listed), hex.EncodeToString(sum[:])+"  ./"+asset) {
		t.Errorf("no sha256sums.txt line beside the binary for twirgen: %q", listed)
	}
	if _, err := locate(); err != nil || requests.Load() != 1 {
		t.Errorf("second run: %v, %d requests, want 1 in all", err, requests.Load())
	}

	if err := os.WriteFile(cached, good[:4], 0o755); err != nil {
		t.Fatal(err)
	}
	if path, err := locate(); err != nil || requests.Load() != 2 {
		t.Errorf("a cut cached file: %v, %d requests, want it fetched again", err, requests.Load())
	} else if got, _ := os.ReadFile(path); string(got) != string(good) {
		t.Errorf("a cut cached file was kept: %q", got)
	}

	project := t.TempDir()
	own := filepath.Join(project, ".twind", "bin", asset)
	if err := os.MkdirAll(filepath.Dir(own), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(own, []byte("the project's own"), 0o755); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(project, "cmd", "app")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if path, err := locateTailwind(inner, cache, release, sums, io.Discard); err != nil || path != own || requests.Load() != 2 {
		t.Errorf("project bin: %q %v after %d requests, want %q and no request", path, err, requests.Load(), own)
	}

	if _, err := locateTailwind(t.TempDir(), t.TempDir(), release, "aaaa  ./tailwindcss-other\n", io.Discard); err == nil || !strings.Contains(err.Error(), asset) || requests.Load() != 2 {
		t.Errorf("no pinned sum: %v after %d requests, want an error naming %s and no request", err, requests.Load(), asset)
	}
}
