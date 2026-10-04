package twi

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	devkonst "github.com/twind-dev/twind/internal/dev/konst"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
)

func Fullscreen() RenderOption { return func(c *renderConfig) { c.fullscreen = true } }

func Graphics(mode terminal.Graphics) RenderOption {
	return func(c *renderConfig) { c.graphics = &mode }
}

func NoClipboard() RenderOption { return func(c *renderConfig) { c.noClipboard = true } }

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
	var devState string
	if cfg.backend == nil {
		cfg.clock, devState = realClock{}, os.Getenv(devkonst.StateEnv)
		if !cfg.profileSet {
			cfg.profile, cfg.profileSet = terminal.Profile(os.Stdout, os.Getenv), true
		}
	}
	if os.Getenv("TWIND_GRAPHICS") != "" {
		cfg.graphics = nil
	}
	r := &Runtime{cfg: cfg, theme: theme.Default()}
	if cfg.theme != nil {
		r.theme = *cfg.theme
	}
	r.Runtime = runtime.New(runtime.Config{Clock: cfg.clock, Sheet: cfg.sheet.WithTheme(&r.theme), Profile: cfg.profile, Graphics: cfg.graphics, NoClipboard: cfg.noClipboard, DevState: devState})
	return r
}

func (r *Runtime) SetTheme(t theme.Theme) {
	r.Restyle(func() { r.theme = t })
}

func (r *Runtime) Theme() theme.Theme { return r.theme }

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
		r.Remember("twi.theme", r.themeKey, r.restoreTheme)
		return app().runtimeTree()
	})
}

func (r *Runtime) themeKey() string {
	return r.theme.Name + "/" + strconv.Itoa(int(r.theme.Scheme))
}

func (r *Runtime) restoreTheme(key string) {
	for _, t := range theme.Builtin() {
		if t.Name+"/"+strconv.Itoa(int(t.Scheme)) == key {
			r.theme = t
		}
	}
}

type terminalBackend struct{ *terminal.Backend }

func (t terminalBackend) Events() <-chan input.Event { return t.Backend.Events }

func (t terminalBackend) Sync() bool { return t.Backend.Capabilities.Sync }

func (t terminalBackend) Capabilities() terminal.Capabilities { return t.Current() }

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
