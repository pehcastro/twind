package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const driveHelp = `Runs a driver script against an app package headlessly, under the fake clock, and writes
each frame line as NAME.txt and NAME.ansi in the out directory.

The package, main or not, declares in its source or its in-package tests

    func App(rt *twi.Runtime) func() twi.Node

and has the Styles function twirgen generates. twind adds a test through go test -overlay
that passes App and Styles to drive.RunScript; nothing is written into the package.

Script verbs: size WxH, press KEY, type TEXT, wait DURATION, resize WxH, frame NAME.`

const harness = `package %s

import (
	twindos "os"
	twindtesting "testing"

	twinddrive "github.com/twind-dev/twind/twi/drive"
)

func TestTwindDrive(t *twindtesting.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	script, err := twindos.Open(%s)
	if err != nil {
		t.Fatal(err)
	}
	defer script.Close()
	if err := twinddrive.RunScript(script, App, %s, twinddrive.Styles(sheet)); err != nil {
		t.Fatal(err)
	}
}
`

func drive(args []string, stdout io.Writer) error {
	fs := flags("drive", "[-out dir] script [package]", driveHelp)
	out := fs.String("out", "frames", "directory the frames are written to")
	positional, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(positional) == 0 || len(positional) > 2 {
		fs.Usage()
		return usageError("want a script and at most one package")
	}
	script, err := filepath.Abs(positional[0])
	if err != nil {
		return err
	}
	if _, err := os.Stat(script); err != nil {
		return err
	}
	frames, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	pkg := "."
	if len(positional) == 2 {
		pkg = positional[1]
	}
	listed, err := goList([]string{pkg}, "{{.Dir}}\t{{.Name}}")
	if err != nil {
		return err
	}
	if len(listed) != 1 {
		return fmt.Errorf("%s names %d packages, want one", pkg, len(listed))
	}
	dir, name, _ := strings.Cut(listed[0], "\t")
	tmp, err := os.MkdirTemp("", "twind-drive")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	source := filepath.Join(tmp, "harness.go")
	overlay, err := json.Marshal(map[string]map[string]string{"Replace": {filepath.Join(dir, "twind_drive_test.go"): source}})
	if err != nil {
		return err
	}
	if err := errors.Join(
		os.WriteFile(source, fmt.Appendf(nil, harness, name, strconv.Quote(script), strconv.Quote(frames)), 0o600),
		os.WriteFile(filepath.Join(tmp, "overlay.json"), overlay, 0o600),
	); err != nil {
		return err
	}
	test := exec.Command("go", "test", "-count=1", "-v", "-run", "^TestTwindDrive$", "-overlay", filepath.Join(tmp, "overlay.json"), ".")
	test.Dir = dir
	result, err := test.CombinedOutput()
	if err == nil && !strings.Contains(string(result), "--- PASS: TestTwindDrive") {
		err = errors.New("the drive harness did not run")
	}
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", pkg, err, result)
	}
	_, err = fmt.Fprintln(stdout, "frames in", frames)
	return err
}
