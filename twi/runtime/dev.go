package runtime

import (
	"maps"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twind-dev/twind/internal/dev/snapshot"
)

type devState struct {
	pending snapshot.Snapshot
	saves   map[string]func() string
}

func (r *Runtime) Remember(key string, save func() string, restore func(string)) {
	if r.cfg.DevState == "" {
		return
	}
	r.dev.saves[key] = save
	if v, ok := r.dev.pending.Values[key]; ok {
		delete(r.dev.pending.Values, key)
		restore(v)
	}
}

func (r *Runtime) listenDev() (stop func()) {
	r.dev.saves = map[string]func() string{}
	if s, err := snapshot.Read(r.cfg.DevState); err == nil {
		r.dev.pending = s
	}
	_ = os.Remove(r.cfg.DevState)
	signals, done := make(chan os.Signal, 1), make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case got := <-signals:
			r.phases.log("runtime: %v received", got)
			r.Quit()
		case <-done:
		}
	}()
	return func() {
		signal.Stop(signals)
		close(done)
	}
}

func (r *Runtime) restoreFocus() {
	if key := r.dev.pending.Focus; key != "" && r.Focus(key) {
		r.dev.pending.Focus = ""
	}
}

func (r *Runtime) restoreScroll() bool {
	moved := false
	for key, at := range r.dev.pending.Scroll {
		if e := r.doc.root.keyed(key); e != nil {
			delete(r.dev.pending.Scroll, key)
			moved = r.tree.ScrollTo(e.path(), at.X, at.Y) || moved
		}
	}
	return moved
}

func (r *Runtime) saveDev() error {
	began := time.Now()
	defer func() { r.phases.log("runtime: snapshot written in %v", time.Since(began).Round(time.Microsecond)) }()
	s := snapshot.Snapshot{Focus: r.dev.pending.Focus, Scroll: map[string]snapshot.Offset{}, Values: map[string]string{}}
	maps.Copy(s.Scroll, r.dev.pending.Scroll)
	maps.Copy(s.Values, r.dev.pending.Values)
	if current, ok := r.focus.Current(); ok && current.node.Key != "" {
		s.Focus = current.node.Key
	}
	var walk func(e *Elem)
	walk = func(e *Elem) {
		if n := r.doc.sceneOf(e); e.node.Key != "" && n.Scroll {
			s.Scroll[e.node.Key] = snapshot.Offset{X: n.Padding.X - n.ScrollContent.X, Y: n.Padding.Y - n.ScrollContent.Y}
		}
		for _, c := range e.children {
			walk(c)
		}
	}
	if r.doc.root != nil {
		walk(r.doc.root)
	}
	for key, save := range r.dev.saves {
		s.Values[key] = save()
	}
	return snapshot.Write(r.cfg.DevState, s)
}
