package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	devkonst "github.com/pehcastro/twind/internal/dev/konst"
	style "github.com/pehcastro/twind/internal/konst/style"
)

//go:embed tailwind-sha256sums.txt
var pinnedSums string

const twirgenPath = "github.com/pehcastro/twind/internal/twirgen"

type styler struct {
	tailwind, twirgen func() (string, error)
}

func newStyler(work string, say io.Writer) *styler {
	return &styler{
		tailwind: sync.OnceValues(func() (string, error) {
			dir, err := os.Getwd()
			cache, cacheErr := os.UserCacheDir()
			if err = errors.Join(err, cacheErr); err != nil {
				return "", err
			}
			return locateTailwind(dir, filepath.Join(cache, "twind"), devkonst.TailwindRelease, pinnedSums, say)
		}),
		twirgen: sync.OnceValues(func() (string, error) {
			exe := executable(filepath.Join(work, "twirgen"))
			if out, err := exec.Command("go", "build", "-o", exe, twirgenPath).CombinedOutput(); err != nil {
				return "", fmt.Errorf("go build twirgen: %w\n%s", err, out)
			}
			return exe, nil
		}),
	}
}

type twirgenLine struct {
	pkg  string
	args []string
}

func twirgenLines(dir string) ([]twirgenLine, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var lines []twirgenLine
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".go" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		for line := range strings.Lines(string(src)) {
			if _, after, ok := strings.Cut(line, "//go:generate go run "+twirgenPath); ok {
				clause, err := parser.ParseFile(token.NewFileSet(), "", src, parser.PackageClauseOnly)
				if err != nil {
					return nil, err
				}
				lines = append(lines, twirgenLine{clause.Name.Name, strings.Fields(after)})
			}
		}
	}
	return lines, nil
}

func (st *styler) regenerate(dir string) error {
	lines, err := twirgenLines(dir)
	if err != nil || len(lines) == 0 {
		return err
	}
	bin, err := st.tailwind()
	if err != nil {
		return err
	}
	twirgen, err := st.twirgen()
	if err != nil {
		return err
	}
	for _, l := range lines {
		run := exec.Command(twirgen, append([]string{"-tailwind", bin}, l.args...)...)
		run.Dir, run.Env = dir, append(os.Environ(), "GOPACKAGE="+l.pkg)
		out, err := run.CombinedOutput()
		if err == nil {
			continue
		}
		var kept strings.Builder
		for line := range strings.Lines(string(out)) {
			if !strings.Contains(line, ": warning: ") {
				kept.WriteString(line)
			}
		}
		return fmt.Errorf("%stwirgen: %w", kept.String(), err)
	}
	return nil
}

func executable(path string) string {
	if runtime.GOOS == "windows" {
		return path + ".exe"
	}
	return path
}

func tailwindAsset(goos, goarch string) string {
	system := map[string]string{"windows": "windows", "linux": "linux", "darwin": "macos"}[goos]
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[goarch]
	if system == "" || arch == "" {
		return ""
	}
	name := "tailwindcss-" + system + "-" + arch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func locateTailwind(dir, cache, release, sums string, say io.Writer) (string, error) {
	asset := tailwindAsset(runtime.GOOS, runtime.GOARCH)
	want := ""
	for line := range strings.Lines(sums) {
		if sum, name, _ := strings.Cut(strings.TrimRight(line, "\r\n"), "  ./"); name == asset && asset != "" {
			want = sum
		}
	}
	if want == "" {
		return "", fmt.Errorf("no pinned Tailwind %s for %s/%s %s", style.TailwindVersion, runtime.GOOS, runtime.GOARCH, asset)
	}
	for d := dir; ; d = filepath.Dir(d) {
		own := filepath.Join(d, ".twind", "bin", asset)
		if _, err := os.Stat(own); err == nil {
			return own, nil
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	home := filepath.Join(cache, "tailwind", style.TailwindVersion)
	path := filepath.Join(home, asset)
	if f, err := os.Open(path); err == nil {
		h := sha256.New()
		_, err = io.Copy(h, f)
		_ = f.Close()
		if err == nil && hex.EncodeToString(h.Sum(nil)) == want {
			return path, nil
		}
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", err
	}
	parts, _ := filepath.Glob(filepath.Join(home, asset+".part*"))
	for _, part := range parts {
		_ = os.Remove(part)
	}
	url := release + style.TailwindVersion + "/" + asset
	_, _ = fmt.Fprintf(say, "fetching Tailwind %s from %s into %s\n", style.TailwindVersion, url, home)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	tmp, err := os.CreateTemp(home, asset+".part")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(tmp, h), resp.Body)
	if err = errors.Join(err, tmp.Close()); err != nil {
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return "", fmt.Errorf("fetch %s: sha256 %s, pinned %s", url, got, want)
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(home, "sha256sums.txt"), []byte(want+"  ./"+asset+"\n"), 0o644); err != nil {
		return "", err
	}
	return path, os.Rename(tmp.Name(), path)
}
