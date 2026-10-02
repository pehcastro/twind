package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/twind-dev/twind/internal/dev"
	devkonst "github.com/twind-dev/twind/internal/dev/konst"
	"github.com/twind-dev/twind/internal/dev/snapshot"
	style "github.com/twind-dev/twind/internal/konst/style"
	term "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/terminal"
)

const devHelp = `Builds the package and runs it. On every save of a Go or embedded file the package imports from
this module, it rebuilds in the background and swaps the running program for the new build. A failed
build keeps the old program running and shows the first compiler error on the last row. Stale Style
IRs are regenerated before each build. Arguments after the package go to the program.`

type devSession struct {
	session   context.Context
	pkg, work string
	irs       []string
	checker   tailwind.Checker
	styles    *styler
	plan      *dev.Plan
	planning  chan *dev.Plan
	log       *log.Logger
	builds    int
	saved     time.Time
	built     time.Time
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
	s := &devSession{pkg: ".", log: log.New(io.Discard, "", 0)}
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
	goenv, err := exec.Command("go", "env", "GOROOT", "GOMOD").Output()
	if err != nil {
		return fmt.Errorf("go env: %w", err)
	}
	goroot, gomod, _ := strings.Cut(strings.TrimSpace(string(goenv)), "\n")
	if err := os.Setenv("PATH", filepath.Join(strings.TrimSpace(goroot), "bin")+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		return err
	}
	root, err := os.Getwd()
	if gomod = strings.TrimSpace(gomod); filepath.IsAbs(gomod) {
		root = filepath.Dir(gomod)
	}
	if err != nil {
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
				s.irs = append(s.irs, fields[0])
				continue
			}
			files = append(files, filepath.Join(fields[0], name))
		}
	}
	if s.work, err = devWorkDir(root); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(s.work) }()
	restore, err := dev.SaveConsole()
	if err != nil {
		return fmt.Errorf("twind dev needs a console: %w", err)
	}
	state := filepath.Join(s.work, "state.json")
	env := append(os.Environ(), devkonst.DevEnv+"=1", devkonst.StateEnv+"="+state)
	if *logPath != "" && os.Getenv(term.TraceEnv) == "" {
		traced, err := filepath.Abs(*logPath)
		if err != nil {
			return err
		}
		env = append(env, term.TraceEnv+"="+traced)
	}
	var last error
	start := func(exe string) (dev.Child, error) {
		stopped := time.Since(s.built).Round(time.Millisecond)
		kept, err := snapshot.Read(state)
		switch {
		case err == nil:
			s.log.Printf("snapshot: focus %q, %d scroll offsets, %d values", kept.Focus, len(kept.Scroll), len(kept.Values))
		case !errors.Is(err, fs.ErrNotExist):
			s.log.Print("snapshot dropped: ", err)
			_ = os.Remove(state)
		}
		c, err := dev.Spawn(exe, appArgs, env, restore)
		if err != nil {
			return nil, err
		}
		s.log.Printf("swap: stop %v, start %v, started %v after the save, %s", stopped, time.Since(s.built).Round(time.Millisecond)-stopped, time.Since(s.saved).Round(time.Millisecond), filepath.Base(exe))
		now := time.Now()
		_ = os.Chtimes(exe, now, now)
		prune(s.work, devkonst.KeptBuilds, exe)
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
	s.styles = newStyler(s.work, s.log.Writer())
	if len(s.irs) > 0 {
		go func() {
			_, err := s.styles.twirgen()
			s.log.Printf("twirgen: built, error %v", err)
		}()
		go func() {
			bin, err := s.styles.tailwind()
			s.log.Printf("tailwind: %s %v", bin, err)
		}()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	s.session = ctx
	dev.Supervisor{Build: s.build, Start: start, Report: report}.Run(ctx, dev.Watch(ctx, files))
	_, _ = os.Stdout.WriteString(term.GraphemesOff + term.KittyPop + term.MouseOff + term.InBandOff + term.LeaveScreen)
	return errors.Join(restore(), last)
}

func (s *devSession) build(ctx context.Context, changed []string) (string, error) {
	began := time.Now()
	s.saved = time.Time{}
	for _, file := range changed {
		if info, err := os.Stat(file); err == nil && info.ModTime().After(s.saved) {
			s.saved = info.ModTime()
		}
	}
	if s.saved.IsZero() {
		s.saved = began
	}
	s.log.Printf("build: %d saves, %v after the last", len(changed), began.Sub(s.saved).Round(time.Millisecond))
	if s.planning != nil {
		s.plan, s.planning = <-s.planning, nil
		s.log.Printf("plan: ready after %v", time.Since(began).Round(time.Millisecond))
	}
	next := func() string {
		s.builds++
		return executable(filepath.Join(s.work, fmt.Sprintf("app-%d", s.builds)))
	}
	exe, compiled := next(), time.Now()
	var steps []dev.Step
	var fast string
	var ahead chan error
	speculative, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		if ahead != nil {
			<-ahead
		}
	}()
	if s.plan != nil {
		ahead = make(chan error, 1)
		go func() {
			var err error
			fast, steps, err = s.plan.Build(speculative, changed, exe)
			ahead <- err
		}()
	}
	for _, dir := range s.irs {
		stale, err := s.checker.Stale(dir, style.GeneratedFile)
		if err != nil {
			return "", err
		}
		s.log.Printf("styles: %s stale %v after %v", filepath.Base(dir), stale, time.Since(began).Round(time.Millisecond))
		if !stale {
			continue
		}
		if ahead != nil {
			cancel()
			<-ahead
			ahead = nil
			s.log.Print("styles: the build started beside the check is dropped")
		}
		if err := s.styles.regenerate(dir); err != nil {
			return "", err
		}
		changed = append(changed, filepath.Join(dir, style.GeneratedFile))
		s.log.Printf("styles: regenerated after %v", time.Since(began).Round(time.Millisecond))
	}
	if s.plan != nil {
		var err error
		if ahead != nil {
			err, ahead = <-ahead, nil
		} else {
			exe, compiled = next(), time.Now()
			fast, steps, err = s.plan.Build(ctx, changed, exe)
		}
		var stale dev.StaleError
		switch {
		case err == nil:
			s.built = time.Now()
			for _, step := range steps {
				s.log.Printf("fast build: %s %v, export changed %v, reused %v", step.Package, step.Took.Round(time.Millisecond), step.Exported, step.Reused)
			}
			s.log.Printf("fast build: %v", s.built.Sub(compiled).Round(time.Millisecond))
			return fast, nil
		case errors.As(err, &stale):
			s.log.Print(err)
			s.plan = nil
		case ctx.Err() != nil:
			return "", ctx.Err()
		default:
			root, _ := os.Getwd()
			return "", errors.New(strings.ReplaceAll(err.Error(), root+string(filepath.Separator), ""))
		}
	}
	if out, err := exec.CommandContext(ctx, "go", "build", "-o", exe, s.pkg).CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New(strings.TrimSpace(string(out)))
	}
	s.built = time.Now()
	s.log.Printf("go build: %v", s.built.Sub(compiled).Round(time.Millisecond))
	planning, dir := make(chan *dev.Plan, 1), filepath.Join(s.work, fmt.Sprintf("plan-%d", s.builds))
	s.planning = planning
	go func() {
		began := time.Now()
		p, err := dev.Capture(s.session, ".", s.pkg, dir)
		s.log.Printf("plan: captured in %v, error %v", time.Since(began).Round(time.Millisecond), err)
		planning <- p
		if p != nil {
			began = time.Now()
			err = p.Warm(s.session)
			s.log.Printf("plan: every package of the module compiled for dev in %v, error %v", time.Since(began).Round(time.Millisecond), err)
		}
	}()
	return exe, nil
}
