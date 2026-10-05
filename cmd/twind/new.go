package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"

	"golang.org/x/mod/module"

	"github.com/pehcastro/twind/templates"
)

const twindModule = "github.com/pehcastro/twind"

const newHelp = `Writes a fullscreen app into dir: a sidebar, cards, a dialog and a theme picker, its Style IR,
a driver script with a test that replays it, a go.mod and a README.
dir must not exist or must be empty; nothing is ever overwritten.
Twind has no published version yet: the README says how to point the app at a checkout.`

func newApp(args []string, stdout io.Writer) error {
	set := flags("new", "[-module path] dir", newHelp)
	modulePath := set.String("module", "", "module path of the app, the directory's name by default")
	positional, err := parse(set, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		set.Usage()
		return usageError("want one directory")
	}
	dir, err := filepath.Abs(positional[0])
	if err != nil {
		return err
	}
	mod := *modulePath
	if mod == "" {
		mod = filepath.Base(dir)
	}
	if err := module.CheckImportPath(mod); err != nil {
		return err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return fmt.Errorf("%s is a file: twind new writes only into a new or empty directory", positional[0])
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty: twind new writes only into a new or empty directory", positional[0])
	}
	version := "v0.0.0"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, m := range append([]*debug.Module{&info.Main}, info.Deps...) {
			if m.Path == twindModule && module.Check(m.Path, m.Version) == nil && !module.IsPseudoVersion(m.Version) {
				version = m.Version
			}
		}
	}
	data := map[string]string{"Module": mod, "Name": path.Base(mod), "Version": version}
	err = fs.WalkDir(templates.App, "app", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		src, err := templates.App.ReadFile(name)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(name, "app/")))
		if strings.HasSuffix(target, ".tmpl") {
			var out bytes.Buffer
			tmpl, err := template.New(name).Option("missingkey=error").Parse(string(src))
			if err == nil {
				err = tmpl.Execute(&out, data)
			}
			if err != nil {
				return err
			}
			src, target = out.Bytes(), strings.TrimSuffix(target, ".tmpl")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		_, err = file.Write(src)
		return errors.Join(err, file.Close())
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote module %s in %s\n", mod, dir)
	return err
}
