package fix

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/fix"
	"github.com/pehcastro/twind/twi/text"
)

type Plan struct {
	Terminal string
	Arch     string
	Backups  []string
	Package  string
	SHA256   string
}

func (p Plan) String() string {
	dir := filepath.Dir(p.Terminal)
	var b strings.Builder
	fmt.Fprintf(&b, "Put Microsoft's ConPTY %s (%s) beside %s:\n  %s\n  %s\n", konst.ConPTYVersion, p.Arch, p.Terminal, filepath.Join(dir, konst.DLL), filepath.Join(dir, konst.Host))
	for _, name := range p.Backups {
		fmt.Fprintf(&b, "The existing %s is renamed to %s first.\n", name, name+konst.Backup)
	}
	fmt.Fprintf(&b, "From %s, sha256 %s.\nUndo removes these files and restores the folder.\n", p.Package, p.SHA256)
	return text.Sanitize(b.String(), text.ShowBidi)
}

type Refused string

func (r Refused) Error() string { return string(r) }

func host() (system, error) {
	chain, build, err := process()
	if err != nil {
		return system{}, err
	}
	cache, err := os.UserCacheDir()
	return system{chain, build, filepath.Join(cache, konst.CacheDir), konst.ConPTYSHA256, download, elevate}, err
}

func ConPTY(ctx context.Context, ask func(Plan) bool) (string, error) {
	s, err := host()
	if err != nil {
		return "", err
	}
	return s.conpty(ctx, ask)
}

func Undo(dir string) (string, error) {
	s, err := host()
	if err != nil {
		return "", err
	}
	return s.undo(dir)
}

func Ask(r io.Reader, w io.Writer) func(Plan) bool {
	return func(p Plan) bool {
		_, _ = fmt.Fprintf(w, "%sType y or yes to go ahead: ", p)
		answer, _ := bufio.NewReader(r).ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		return answer == "y" || answer == "yes"
	}
}

type verb int

const (
	copyItem verb = iota
	moveItem
	removeItem
)

type op struct {
	verb     verb
	from, to string
}

type file struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type manifest struct {
	Version string   `json:"version"`
	Files   []file   `json:"files"`
	Backups []string `json:"backups"`
}

type system struct {
	chain   []string
	build   uint32
	cache   string
	sum     string
	fetch   func(context.Context, string) (io.ReadCloser, error)
	elevate func(string) error
}

func (s system) terminal() (string, error) {
	for _, exe := range s.chain {
		if name := strings.ToLower(filepath.Base(exe)); filepath.IsAbs(exe) && (name == konst.Alacritty || name == konst.Rio) {
			return exe, nil
		}
	}
	return "", Refused("not running in " + konst.Alacritty + " or " + konst.Rio + ": Windows Terminal, WezTerm, VS Code and Zed ship their own ConPTY, and mintty does not load one from its folder")
}

func (s system) conpty(ctx context.Context, ask func(Plan) bool) (string, error) {
	if s.build >= konst.ModernBuild {
		return "", Refused(fmt.Sprintf("not needed: Windows build %d has a modern inbox ConPTY", s.build))
	}
	exe, err := s.terminal()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if exists(filepath.Join(dir, konst.Manifest)) {
		return "", Refused("already fixed in " + dir + ": undo first")
	}
	f, err := pe.Open(exe)
	if err != nil {
		return "", err
	}
	machine := f.Machine
	_ = f.Close()
	arch := map[uint16]string{pe.IMAGE_FILE_MACHINE_AMD64: "x64", pe.IMAGE_FILE_MACHINE_ARM64: "arm64", pe.IMAGE_FILE_MACHINE_I386: "x86"}[machine]
	if arch == "" {
		return "", Refused(fmt.Sprintf("%s is built for machine %#x, which the ConPTY package does not cover", exe, machine))
	}
	plan := Plan{Terminal: exe, Arch: arch, Package: konst.ConPTYPackage, SHA256: s.sum}
	ours := []string{konst.DLL, konst.Host}
	for _, name := range ours {
		if !exists(filepath.Join(dir, name)) {
			continue
		}
		if exists(filepath.Join(dir, name+konst.Backup)) {
			return "", Refused(fmt.Sprintf("%s and %s both exist in %s: move one away first", name, name+konst.Backup, dir))
		}
		plan.Backups = append(plan.Backups, name)
	}
	if !ask(plan) {
		return "", Refused("declined: nothing was written")
	}
	staged, err := s.stage(ctx, arch)
	if err != nil {
		return "", err
	}
	m := manifest{Version: konst.ConPTYVersion, Backups: plan.Backups}
	ops := []op{{copyItem, filepath.Join(staged, konst.Manifest), filepath.Join(dir, konst.Manifest)}}
	for _, name := range plan.Backups {
		ops = append(ops, op{moveItem, filepath.Join(dir, name), filepath.Join(dir, name+konst.Backup)})
	}
	for _, name := range ours {
		m.Files = append(m.Files, file{name, sumOf(filepath.Join(staged, name))})
		ops = append(ops, op{copyItem, filepath.Join(staged, name), filepath.Join(dir, name)})
	}
	record, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staged, konst.Manifest), record, konst.FilePerm); err != nil {
		return "", err
	}
	if err := s.apply(dir, ops); err != nil {
		return "", err
	}
	return fmt.Sprintf("Installed ConPTY %s beside %s.\nRestart the terminal: it then gets mouse clicks and scrolling, colour replies, and in Rio images.\n", konst.ConPTYVersion, text.Sanitize(exe, text.ShowBidi)), nil
}

func (s system) stage(ctx context.Context, arch string) (string, error) {
	home := filepath.Join(s.cache, konst.ConPTYVersion)
	pkg := filepath.Join(home, konst.PackageFile)
	if sumOf(pkg) != s.sum {
		if err := os.MkdirAll(home, konst.DirPerm); err != nil {
			return "", err
		}
		body, err := s.fetch(ctx, konst.ConPTYPackage)
		if err != nil {
			return "", err
		}
		tmp, err := os.CreateTemp(home, konst.PackageFile)
		if err != nil {
			_ = body.Close()
			return "", err
		}
		defer func() { _ = os.Remove(tmp.Name()) }()
		h := sha256.New()
		_, err = io.Copy(io.MultiWriter(tmp, h), body)
		if err = errors.Join(err, body.Close(), tmp.Close()); err != nil {
			return "", err
		}
		if got := hex.EncodeToString(h.Sum(nil)); got != s.sum {
			return "", fmt.Errorf("%s: sha256 %s, pinned %s; nothing was written", konst.ConPTYPackage, got, s.sum)
		}
		if err := os.Rename(tmp.Name(), pkg); err != nil {
			return "", err
		}
	}
	z, err := zip.OpenReader(pkg)
	if err != nil {
		return "", err
	}
	defer func() { _ = z.Close() }()
	staged := filepath.Join(home, arch)
	if err := os.MkdirAll(staged, konst.DirPerm); err != nil {
		return "", err
	}
	for name, entry := range map[string]string{konst.DLL: konst.DLLEntry, konst.Host: konst.HostEntry} {
		b, err := fs.ReadFile(z, fmt.Sprintf(entry, arch))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(staged, name), b, konst.FilePerm); err != nil {
			return "", err
		}
	}
	return staged, nil
}

func (s system) undo(dir string) (string, error) {
	if dir == "" {
		exe, err := s.terminal()
		if err != nil {
			return "", err
		}
		dir = filepath.Dir(exe)
	}
	record, err := os.ReadFile(filepath.Join(dir, konst.Manifest))
	if errors.Is(err, fs.ErrNotExist) {
		return "", Refused("nothing to undo in " + dir + ": no " + konst.Manifest)
	}
	if err != nil {
		return "", err
	}
	var m manifest
	if err := json.Unmarshal(record, &m); err != nil {
		return "", fmt.Errorf("%s: %w", konst.Manifest, err)
	}
	ours := []string{konst.DLL, konst.Host}
	var ops []op
	var kept []string
	for _, f := range m.Files {
		if !slices.Contains(ours, f.Name) {
			return "", fmt.Errorf("%s names %q, not a file this fix writes", konst.Manifest, f.Name)
		}
		path := filepath.Join(dir, f.Name)
		switch {
		case sumOf(path) == f.SHA256:
			ops = append(ops, op{removeItem, path, ""})
		case exists(path):
			kept = append(kept, f.Name)
		}
	}
	for _, name := range m.Backups {
		if !slices.Contains(ours, name) {
			return "", fmt.Errorf("%s names %q, not a file this fix writes", konst.Manifest, name)
		}
		if !slices.Contains(kept, name) && exists(filepath.Join(dir, name+konst.Backup)) {
			ops = append(ops, op{moveItem, filepath.Join(dir, name+konst.Backup), filepath.Join(dir, name)})
		}
	}
	ops = append(ops, op{removeItem, filepath.Join(dir, konst.Manifest), ""})
	if err := s.apply(dir, ops); err != nil {
		return "", fmt.Errorf("%w\nThe terminal keeps these files open while it runs: close every window of it, then undo from another terminal, naming %s", err, dir)
	}
	said := "Restored " + dir + ". Restart the terminal.\n"
	for _, name := range kept {
		said += "Left " + name + ": it changed after the fix.\n"
	}
	return text.Sanitize(said, text.ShowBidi), nil
}

func (s system) apply(dir string, ops []op) error {
	probe, err := os.CreateTemp(dir, konst.Manifest)
	if errors.Is(err, fs.ErrPermission) {
		return s.elevate(script(ops))
	}
	if err != nil {
		return err
	}
	if err := errors.Join(probe.Close(), os.Remove(probe.Name())); err != nil {
		return err
	}
	for _, o := range ops {
		if err := run(o); err != nil {
			return err
		}
	}
	return nil
}

func run(o op) error {
	switch o.verb {
	case copyItem:
		b, err := os.ReadFile(o.from)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(o.to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, konst.FilePerm)
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		return errors.Join(err, f.Close())
	case moveItem:
		if exists(o.to) {
			return fmt.Errorf("%s already exists", o.to)
		}
		return os.Rename(o.from, o.to)
	case removeItem:
		return os.Remove(o.from)
	}
	panic(fmt.Sprintf("fix: unknown verb %d", o.verb))
}

func script(ops []op) string {
	quote := func(path string) string {
		var b strings.Builder
		b.WriteByte('\'')
		for _, r := range path {
			if strings.ContainsRune("'\u2018\u2019\u201a\u201b", r) {
				b.WriteRune(r)
			}
			b.WriteRune(r)
		}
		b.WriteByte('\'')
		return b.String()
	}
	lines := []string{"$ErrorActionPreference = 'Stop'"}
	for _, o := range ops {
		switch o.verb {
		case copyItem:
			lines = append(lines, "Copy-Item -LiteralPath "+quote(o.from)+" -Destination "+quote(o.to))
		case moveItem:
			lines = append(lines, "Move-Item -LiteralPath "+quote(o.from)+" -Destination "+quote(o.to))
		case removeItem:
			lines = append(lines, "Remove-Item -LiteralPath "+quote(o.from))
		default:
			panic(fmt.Sprintf("fix: unknown verb %d", o.verb))
		}
	}
	return strings.Join(lines, "\n")
}

func download(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return resp.Body, nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func sumOf(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
