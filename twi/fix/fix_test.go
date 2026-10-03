package fix

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/fix"
)

func fakeExe(machine uint16) []byte {
	const header = 0x80
	b := make([]byte, header+len("PE\x00\x00")+binary.Size(pe.FileHeader{}))
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], header)
	copy(b[header:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(b[header+len("PE\x00\x00"):], machine)
	return b
}

func fakePackage(t *testing.T) ([]byte, string) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, arch := range []string{"x86", "x64", "arm64"} {
		for _, entry := range []string{konst.DLLEntry, konst.HostEntry} {
			f, err := w.Create(fmt.Sprintf(entry, arch))
			if err != nil {
				t.Fatal(err)
			}
			_, _ = f.Write([]byte(arch + " " + filepath.Base(entry)))
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

func snapshot(t *testing.T, dir string) map[string]string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[e.Name()] = string(b)
	}
	return files
}

type rig struct {
	sys     system
	dir     string
	fetched int
}

func newRig(t *testing.T, exe string, machine uint16, files map[string]string) *rig {
	pkg, sum := fakePackage(t)
	r := &rig{dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(r.dir, exe), fakeExe(machine), konst.FilePerm); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(r.dir, name), []byte(body), konst.FilePerm); err != nil {
			t.Fatal(err)
		}
	}
	r.sys = system{
		chain: []string{`C:\Program Files\PowerShell\7\pwsh.exe`, filepath.Join(r.dir, exe), `C:\Windows\explorer.exe`},
		build: 19045,
		cache: t.TempDir(),
		sum:   sum,
		fetch: func(context.Context, string) (io.ReadCloser, error) {
			r.fetched++
			return io.NopCloser(bytes.NewReader(pkg)), nil
		},
		elevate: func(script string) error {
			t.Fatalf("elevated in a writable folder: %s", script)
			return nil
		},
	}
	return r
}

func yes(Plan) bool { return true }

func TestConPTYRefusesWithoutYes(t *testing.T) {
	r := newRig(t, konst.Alacritty, pe.IMAGE_FILE_MACHINE_AMD64, map[string]string{"alacritty.yml": "font"})
	before := snapshot(t, r.dir)
	var asked Plan
	_, err := r.sys.conpty(context.Background(), func(p Plan) bool {
		asked = p
		if r.fetched != 0 {
			t.Error("downloaded before the answer")
		}
		return false
	})
	if !errors.As(err, new(Refused)) {
		t.Errorf("no: got %v, want a refusal", err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, before) {
		t.Errorf("no: folder changed to %v", got)
	}
	if r.fetched != 0 {
		t.Errorf("no: fetched %d times", r.fetched)
	}
	said := asked.String()
	for _, want := range []string{filepath.Join(r.dir, konst.DLL), filepath.Join(r.dir, konst.Host), konst.ConPTYVersion, konst.ConPTYPackage, r.sys.sum, "x64"} {
		if !strings.Contains(said, want) {
			t.Errorf("the plan does not say %q:\n%s", want, said)
		}
	}
}

func TestConPTYRefusesTerminalsThatDoNotNeedIt(t *testing.T) {
	cases := []struct {
		name  string
		chain []string
		build uint32
	}{
		{"windows terminal", []string{`C:\pwsh.exe`, `C:\Program Files\WindowsApps\Microsoft.WindowsTerminal\WindowsTerminal.exe`}, 19045},
		{"vs code", []string{`C:\pwsh.exe`, `C:\Code\Code.exe`}, 19045},
		{"wezterm", []string{`C:\pwsh.exe`, `C:\WezTerm\wezterm-gui.exe`}, 19045},
		{"mintty", []string{`C:\Git\usr\bin\bash.exe`, `C:\Git\usr\bin\mintty.exe`}, 19045},
		{"conhost", []string{`C:\Windows\explorer.exe`}, 19045},
		{"no parent", []string{}, 19045},
		{"modern windows", nil, konst.ModernBuild},
	}
	for _, tc := range cases {
		r := newRig(t, konst.Alacritty, pe.IMAGE_FILE_MACHINE_AMD64, nil)
		if tc.chain != nil {
			r.sys.chain = tc.chain
		}
		r.sys.build = tc.build
		before := snapshot(t, r.dir)
		_, err := r.sys.conpty(context.Background(), func(Plan) bool {
			t.Errorf("%s: asked", tc.name)
			return true
		})
		if !errors.As(err, new(Refused)) {
			t.Errorf("%s: got %v, want a refusal", tc.name, err)
		}
		if got := snapshot(t, r.dir); !maps.Equal(got, before) || r.fetched != 0 {
			t.Errorf("%s: folder %v, fetched %d", tc.name, got, r.fetched)
		}
	}
}

func TestConPTYBacksUpAndUndoRestores(t *testing.T) {
	r := newRig(t, konst.Rio, pe.IMAGE_FILE_MACHINE_ARM64, map[string]string{konst.DLL: "old dll", "rio.toml": "theme"})
	before := snapshot(t, r.dir)
	said, err := r.sys.conpty(context.Background(), func(p Plan) bool {
		if !strings.Contains(p.String(), konst.DLL+konst.Backup) {
			t.Errorf("the plan does not name the backup:\n%s", p)
		}
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(said, "Restart") {
		t.Errorf("after the fix: %q, want it to say restart", said)
	}
	got := snapshot(t, r.dir)
	want := map[string]string{konst.DLL: "arm64 conpty.dll", konst.DLL + konst.Backup: "old dll", konst.Host: "arm64 OpenConsole.exe", "rio.toml": "theme", konst.Rio: before[konst.Rio]}
	for name, body := range want {
		if got[name] != body {
			t.Errorf("after the fix %s is %q, want %q", name, got[name], body)
		}
	}
	if _, ok := got[konst.Manifest]; !ok || len(got) != len(want)+1 {
		t.Errorf("after the fix the folder holds %v", got)
	}
	fixed := got
	if _, err := r.sys.conpty(context.Background(), yes); !errors.As(err, new(Refused)) {
		t.Errorf("a second fix: got %v, want a refusal", err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, fixed) {
		t.Errorf("a second fix changed the folder to %v", got)
	}
	held, err := os.Open(filepath.Join(r.dir, konst.DLL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.sys.undo("")
	_ = held.Close()
	if err == nil || !strings.Contains(err.Error(), "close every") || !strings.Contains(err.Error(), r.dir) {
		t.Errorf("undo while the terminal holds the dll: %v, want it to say close every window and name the folder", err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, fixed) {
		t.Errorf("a failed undo changed the folder to %v", got)
	}
	r.sys.chain = nil
	if _, err := r.sys.undo(r.dir); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, before) {
		t.Errorf("after undo the folder is %v, want %v", got, before)
	}
}

func TestConPTYNeverOverwritesABackup(t *testing.T) {
	r := newRig(t, konst.Alacritty, pe.IMAGE_FILE_MACHINE_AMD64, map[string]string{konst.Host: "old host", konst.Host + konst.Backup: "older host"})
	before := snapshot(t, r.dir)
	if _, err := r.sys.conpty(context.Background(), yes); !errors.As(err, new(Refused)) {
		t.Errorf("got %v, want a refusal", err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, before) {
		t.Errorf("folder changed to %v", got)
	}
}

func TestUndoTouchesOnlyItsOwnFiles(t *testing.T) {
	r := newRig(t, konst.Alacritty, pe.IMAGE_FILE_MACHINE_AMD64, map[string]string{"notes.bak": "mine"})
	if _, err := r.sys.undo(""); !errors.As(err, new(Refused)) {
		t.Errorf("undo with no fix: got %v, want a refusal", err)
	}
	if _, err := r.sys.conpty(context.Background(), yes); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, konst.Host), []byte("the user's own"), konst.FilePerm); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sys.undo(""); err != nil {
		t.Fatal(err)
	}
	got := snapshot(t, r.dir)
	want := map[string]string{konst.Alacritty: string(fakeExe(pe.IMAGE_FILE_MACHINE_AMD64)), "notes.bak": "mine", konst.Host: "the user's own"}
	if !maps.Equal(got, want) {
		t.Errorf("after undo the folder is %v, want %v", got, want)
	}

	exe := filepath.Join(r.dir, konst.Alacritty)
	forged := fmt.Sprintf(`{"version":%q,"files":[{"name":%q,"sha256":%q}]}`, konst.ConPTYVersion, konst.Alacritty, sumOf(exe))
	if err := os.WriteFile(filepath.Join(r.dir, konst.Manifest), []byte(forged), konst.FilePerm); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sys.undo(""); err == nil {
		t.Error("undo followed a manifest naming the terminal's exe")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Errorf("undo removed the terminal: %v", err)
	}
}

func TestConPTYChecksumMismatchWritesNothing(t *testing.T) {
	r := newRig(t, konst.Alacritty, pe.IMAGE_FILE_MACHINE_AMD64, nil)
	before := snapshot(t, r.dir)
	r.sys.sum = strings.Repeat("0", sha256.Size*2)
	_, err := r.sys.conpty(context.Background(), yes)
	if err == nil || !strings.Contains(err.Error(), r.sys.sum) {
		t.Errorf("got %v, want a sha256 mismatch naming the pinned sum", err)
	}
	if got := snapshot(t, r.dir); !maps.Equal(got, before) {
		t.Errorf("folder changed to %v", got)
	}
	var left []string
	_ = filepath.WalkDir(r.sys.cache, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			left = append(left, path)
		}
		return nil
	})
	if len(left) != 0 {
		t.Errorf("the bad download stayed in the cache: %v", left)
	}
}

func TestScriptQuotesEveryPath(t *testing.T) {
	got := script([]op{{copyItem, `C:\a'b\conpty.dll`, "C:\\x\u2019y\\conpty.dll"}, {moveItem, `C:\m`, `C:\n`}, {removeItem, `C:\r`, ""}})
	for _, want := range []string{`Copy-Item -LiteralPath 'C:\a''b\conpty.dll' -Destination 'C:\x` + "\u2019\u2019" + `y\conpty.dll'`, `Move-Item -LiteralPath 'C:\m' -Destination 'C:\n'`, `Remove-Item -LiteralPath 'C:\r'`} {
		if !strings.Contains(got, want) {
			t.Errorf("script lacks %s:\n%s", want, got)
		}
	}
}

func TestAskWantsAnExplicitYes(t *testing.T) {
	for in, want := range map[string]bool{"yes\n": true, "YES\r\n": true, "y\n": false, "\n": false, "": false, "no\n": false} {
		var out bytes.Buffer
		if got := Ask(strings.NewReader(in), &out)(Plan{Terminal: `C:\a\alacritty.exe`}); got != want {
			t.Errorf("answer %q: %v, want %v", in, got, want)
		}
		if !strings.Contains(out.String(), "alacritty.exe") || !strings.Contains(out.String(), "yes") {
			t.Errorf("answer %q: the prompt was %q", in, out.String())
		}
	}
}
