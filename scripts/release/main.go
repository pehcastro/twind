package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	out := flag.String("out", filepath.Join(os.TempDir(), "twind-release"), "directory for the archives, checksums.txt and the try scripts")
	flag.Parse()
	if err := release(*out); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func release(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	var sums strings.Builder
	for _, t := range [][2]string{{"windows", "amd64"}, {"windows", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}} {
		name, archive, err := build(t[0], t[1])
		if err != nil {
			return fmt.Errorf("%s/%s: %w", t[0], t[1], err)
		}
		if err := os.WriteFile(filepath.Join(out, name), archive, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(archive), name)
		fmt.Println(name, len(archive))
	}
	if err := os.WriteFile(filepath.Join(out, "checksums.txt"), []byte(sums.String()), 0o644); err != nil {
		return err
	}
	for ext, call := range map[string][2]string{".ps1": {"} ", "@args\n"}, ".sh": {"main ", "\"$@\"\n"}} {
		try, err := os.ReadFile(filepath.Join("scripts", "try"+ext))
		if err != nil {
			return err
		}
		body, found := strings.CutSuffix(string(try), call[0]+call[1])
		if !found {
			return fmt.Errorf("scripts/try%s does not end in %q", ext, call[0]+call[1])
		}
		for _, d := range []string{"docs", "landing", "portfolio", "playground", "gallery"} {
			if err := os.WriteFile(filepath.Join(out, d+ext), []byte(body+call[0]+d+" "+call[1]), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func build(goos, goarch string) (string, []byte, error) {
	dir, err := os.MkdirTemp("", "twind-release-")
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	exe := "twind"
	if goos == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", filepath.Join(dir, exe), "./cmd/twind")
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", nil, err
	}
	binary, err := os.ReadFile(filepath.Join(dir, exe))
	if err != nil {
		return "", nil, err
	}
	var archive bytes.Buffer
	name := "twind_" + goos + "_" + goarch
	if goos == "windows" {
		w := zip.NewWriter(&archive)
		f, err := w.CreateHeader(&zip.FileHeader{Name: exe, Method: zip.Deflate, Modified: time.Unix(0, 0)})
		if err == nil {
			_, err = f.Write(binary)
		}
		if err == nil {
			err = w.Close()
		}
		return name + ".zip", archive.Bytes(), err
	}
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	err = tw.WriteHeader(&tar.Header{Name: exe, Mode: 0o755, Size: int64(len(binary)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg})
	if err == nil {
		_, err = tw.Write(binary)
	}
	if err == nil {
		err = tw.Close()
	}
	if err == nil {
		err = gz.Close()
	}
	return name + ".tar.gz", archive.Bytes(), err
}
