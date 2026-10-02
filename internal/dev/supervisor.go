package dev

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/twind-dev/twind/internal/dev/konst"
)

type Child interface {
	Stop() error
	Done() <-chan error
}

type Supervisor struct {
	Build  func(ctx context.Context, changed []string) (exe string, err error)
	Start  func(exe string) (Child, error)
	Report func(error)
}

type built struct {
	exe string
	err error
}

func (s Supervisor) Run(ctx context.Context, changes <-chan []string) {
	var child Child
	var quit <-chan error
	var pending []string
	results := make(chan built, 1)
	cancel := context.CancelFunc(func() {})
	building, again := false, false
	build := func() {
		var bctx context.Context
		bctx, cancel = context.WithCancel(ctx)
		building, again = true, false
		files := slices.Clone(pending)
		go func() {
			exe, err := s.Build(bctx, files)
			results <- built{exe, err}
		}()
	}
	defer func() {
		cancel()
		if child != nil {
			s.report(child.Stop())
		}
	}()
	build()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-quit:
			child, quit = nil, nil
			if err == nil {
				return
			}
			s.Report(err)
		case files := <-changes:
			pending = append(pending, files...)
			if building {
				cancel()
				again = true
				continue
			}
			build()
		case r := <-results:
			cancel()
			building = false
			if again {
				build()
				continue
			}
			if r.err != nil {
				s.Report(r.err)
				continue
			}
			pending = nil
			if child != nil {
				s.report(child.Stop())
			}
			child, quit = nil, nil
			next, err := s.Start(r.exe)
			if err == nil {
				child, quit = next, next.Done()
			}
			s.Report(err)
		}
	}
}

func (s Supervisor) report(err error) {
	if err != nil {
		s.Report(err)
	}
}

type stamp struct {
	modified time.Time
	size     int64
}

func Watch(ctx context.Context, files []string) <-chan []string {
	changes := make(chan []string)
	look := func(file string) stamp {
		info, err := os.Stat(file)
		if err != nil {
			return stamp{}
		}
		return stamp{info.ModTime(), info.Size()}
	}
	seen := make([]stamp, len(files))
	for i, f := range files {
		seen[i] = look(f)
	}
	wake := make(chan struct{}, 1)
	if len(files) > 0 {
		root := filepath.Dir(files[0])
		for _, f := range files {
			for !strings.HasPrefix(f, root+string(filepath.Separator)) && filepath.Dir(root) != root {
				root = filepath.Dir(root)
			}
		}
		go notify(ctx, root, wake)
	}
	go func() {
		tick := time.NewTicker(konst.Poll)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			case <-wake:
				time.Sleep(konst.Settle)
				select {
				case <-wake:
				default:
				}
			}
			var changed []string
			for i, f := range files {
				if now := look(f); !now.modified.Equal(seen[i].modified) || now.size != seen[i].size {
					seen[i] = now
					changed = append(changed, f)
				}
			}
			if changed == nil {
				continue
			}
			select {
			case changes <- changed:
			case <-ctx.Done():
				return
			}
		}
	}()
	return changes
}
