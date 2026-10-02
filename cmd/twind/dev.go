package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/twind-dev/twind/internal/dev"
	devkonst "github.com/twind-dev/twind/internal/dev/konst"
	"github.com/twind-dev/twind/internal/dev/snapshot"
	style "github.com/twind-dev/twind/internal/konst/style"
	term "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/terminal"
)

const devHelp = `Builds the package and runs it. On every save of a Go or embedded file the package imports from
this module, it rebuilds in the background and swaps the running program for the new build. A failed
build keeps the old program running and shows the first compiler error on the last row. When a saved
file's strings change, stale Style IRs are regenerated first. Arguments after the package go to the program.`

type devSession struct {
	pkg, work string
	irDirs    []string
	literals  map[string]string
	log       *log.Logger
	builds    int
	saved     time.Time
	built     time.Time
	running   string
}

func devRun(args []string, _ io.Writer) error {
	set := flags("dev", "[-log file] [package] [program arguments]", devHelp)
	logPath := set.String("log", "", "file to append build and swap timings to")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return usageError(err.Error())
	}
	s := &devSession{pkg: ".", literals: map[string]string{}, log: log.New(io.Discard, "", 0)}
	var appArgs []string
	if set.NArg() > 0 {
		s.pkg, appArgs = set.Arg(0), set.Args()[1:]
	}
	if *logPath != "" {
		if err := os.MkdirAll(filepath.Dir(*logPath), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(*logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		s.log = log.New(f, "", log.Ltime|log.Lmicroseconds)
	}
	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return fmt.Errorf("go env: %w", err)
	}
	if err := os.Setenv("PATH", filepath.Join(strings.TrimSpace(string(goroot)), "bin")+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		return err
	}
	listing, err := goList([]string{"-deps", s.pkg}, `{{if .Module}}{{if .Module.Main}}{{.Dir}}{{range .GoFiles}}{{"\t"}}{{.}}{{end}}{{range .EmbedFiles}}{{"\t"}}{{.}}{{end}}{{end}}{{end}}`)
	if err != nil {
		return err
	}
	var files []string
	for _, line := range listing {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		for _, name := range fields[1:] {
			if name == style.GeneratedFile {
				s.irDirs = append(s.irDirs, fields[0])
				continue
			}
			path := filepath.Join(fields[0], name)
			files = append(files, path)
			if filepath.Ext(name) == ".go" {
				s.literals[path] = literals(path)
			}
		}
	}
	if s.work, err = os.MkdirTemp("", "twind-dev-"); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(s.work) }()
	restore, err := dev.SaveConsole()
	if err != nil {
		return fmt.Errorf("twind dev needs a console: %w", err)
	}
	state := filepath.Join(s.work, "state.json")
	env := append(os.Environ(), devkonst.DevEnv+"=1", devkonst.StateEnv+"="+state)
	var last error
	start := func(exe string) (dev.Child, error) {
		if _, err := snapshot.Read(state); err != nil && !errors.Is(err, fs.ErrNotExist) {
			s.log.Print("snapshot dropped: ", err)
			_ = os.Remove(state)
		}
		c, err := dev.Spawn(exe, appArgs, env, restore)
		if err != nil {
			return nil, err
		}
		s.log.Printf("swap: stop %v, started %v after the save", time.Since(s.built).Round(time.Millisecond), time.Since(s.saved).Round(time.Millisecond))
		if s.running != "" {
			_ = os.Remove(s.running)
		}
		s.running = exe
		return c, nil
	}
	report := func(err error) {
		last = err
		if err == nil {
			return
		}
		s.log.Print("error: ", err)
		if width, height, sizeErr := terminal.Size(os.Stdout); sizeErr == nil {
			_, _ = os.Stdout.WriteString(dev.ErrorLine(err, width, height))
		}
	}
	_, _ = fmt.Fprintf(os.Stdout, "%stwind dev: building %s, watching %d files\r\n", devkonst.AltScreen, s.pkg, len(files))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	dev.Supervisor{Build: s.build, Start: start, Report: report}.Run(ctx, dev.Watch(ctx, files))
	_, _ = os.Stdout.WriteString(term.GraphemesOff + term.KittyPop + term.MouseOff + term.InBandOff + term.LeaveScreen)
	return errors.Join(restore(), last)
}

func (s *devSession) build(ctx context.Context, changed []string) (string, error) {
	began := time.Now()
	s.saved = time.Time{}
	fresh := map[string]string{}
	for _, file := range changed {
		if info, err := os.Stat(file); err == nil && info.ModTime().After(s.saved) {
			s.saved = info.ModTime()
		}
		if old, ok := s.literals[file]; ok {
			if now := literals(file); now != old {
				fresh[file] = now
			}
		}
	}
	if s.saved.IsZero() {
		s.saved = began
	}
	s.log.Printf("build: %d saves, %v after the last", len(changed), began.Sub(s.saved).Round(time.Millisecond))
	if len(fresh) > 0 && len(s.irDirs) > 0 {
		var out strings.Builder
		if err := generate(s.irDirs, &out); err != nil {
			var kept strings.Builder
			for line := range strings.Lines(out.String()) {
				if !strings.Contains(line, ": warning: ") {
					kept.WriteString(line)
				}
			}
			return "", fmt.Errorf("%s%w", kept.String(), err)
		}
		s.log.Printf("styles: %v", time.Since(began).Round(time.Millisecond))
	}
	maps.Copy(s.literals, fresh)
	s.builds++
	exe := filepath.Join(s.work, fmt.Sprintf("app-%d", s.builds))
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	compiled := time.Now()
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", exe, s.pkg).CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New(strings.TrimSpace(string(out)))
	}
	s.built = time.Now()
	s.log.Printf("go build: %v", s.built.Sub(compiled).Round(time.Millisecond))
	return exe, nil
}

func literals(path string) string {
	src, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var scan scanner.Scanner
	scan.Init(token.NewFileSet().AddFile(path, -1, len(src)), src, nil, 0)
	var all strings.Builder
	for {
		_, tok, lit := scan.Scan()
		if tok == token.EOF {
			return all.String()
		}
		if tok == token.STRING {
			all.WriteString(lit + "\x00")
		}
	}
}
