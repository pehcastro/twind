package twi

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
)

func Fullscreen() RenderOption { return func(c *renderConfig) { c.fullscreen = true } }

func Graphics(mode terminal.Graphics) RenderOption {
	return func(c *renderConfig) { c.graphics = &mode }
}

func Backend(b runtime.Backend, c runtime.Clock) RenderOption {
	return func(cfg *renderConfig) { cfg.backend, cfg.clock = b, c }
}

type Timer = runtime.Timer

type Runtime struct {
	*runtime.Runtime
	cfg   renderConfig
	theme theme.Theme
}

func New(opts ...RenderOption) *Runtime {
	var cfg renderConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.backend == nil {
		cfg.clock = realClock{}
		if !cfg.profileSet {
			cfg.profile, cfg.profileSet = terminal.Profile(os.Stdout, os.Getenv), true
		}
	}
	if os.Getenv("TWIND_GRAPHICS") != "" {
		cfg.graphics = nil
	}
	r := &Runtime{cfg: cfg}
	if cfg.theme != nil {
		r.theme = *cfg.theme
	}
	r.Runtime = runtime.New(runtime.Config{Clock: cfg.clock, Sheet: cfg.sheet.WithTheme(&r.theme), Profile: cfg.profile, Graphics: cfg.graphics})
	return r
}

func (r *Runtime) SetTheme(t theme.Theme) {
	r.Restyle(func() { r.theme = t })
}

func (r *Runtime) Run(app func() Node) error {
	if !r.cfg.profileSet {
		return errors.New("twi: Backend needs a ColorProfile")
	}
	b := r.cfg.backend
	if b == nil {
		if !r.cfg.fullscreen {
			return errors.New("twi: Run needs a mode, Fullscreen()")
		}
		t, err := terminal.Enter(os.Stdin, os.Stdout, terminal.Options{})
		if err != nil {
			return err
		}
		b = terminalBackend{t}
	}
	return r.Runtime.Run(b, func() runtime.Tree {
		n := app()
		return runtime.Tree{Root: n.tree, Keys: n.keys, Events: n.events}
	})
}

type terminalBackend struct{ *terminal.Backend }

func (t terminalBackend) Events() <-chan input.Event { return t.Backend.Events }

func (t terminalBackend) Sync() bool { return t.Backend.Capabilities.Sync }

func (t terminalBackend) Capabilities() terminal.Capabilities { return t.Backend.Capabilities }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type Signal[T any] struct {
	owner *runtime.Runtime
	mu    sync.Mutex
	value T
}

func NewSignal[T any](rt *Runtime, value T) *Signal[T] {
	return &Signal[T]{owner: rt.Runtime, value: value}
}

func (s *Signal[T]) Get() T {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

func (s *Signal[T]) Set(value T) {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
	s.owner.Invalidate()
}
