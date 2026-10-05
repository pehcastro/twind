package twi

import (
	"errors"
	"image"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	devkonst "github.com/pehcastro/twind/internal/dev/konst"
	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	konst "github.com/pehcastro/twind/internal/konst/twi"
	"github.com/pehcastro/twind/internal/runtime"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/theme"
)

type Timer = runtime.Timer

type Runtime struct {
	rt    *runtime.Runtime
	cfg   config
	theme theme.Theme
}

func New(opts ...Option) *Runtime {
	var cfg config
	for _, o := range opts {
		o.applyRun(&cfg)
	}
	var devState string
	if cfg.backend == nil {
		cfg.clock, devState = realClock{}, os.Getenv(devkonst.StateEnv)
		if !cfg.profileSet {
			cfg.profile, cfg.profileSet = terminal.Profile(os.Stdout, os.Getenv), true
		}
	}
	if os.Getenv(termkonst.GraphicsEnv) != "" {
		cfg.graphics = nil
	}
	r := &Runtime{cfg: cfg, theme: theme.Default()}
	if cfg.theme != nil {
		r.theme = *cfg.theme
	}
	r.rt = runtime.New(runtime.Config{Clock: cfg.clock, Sheet: cfg.sheet.WithTheme(&r.theme), Profile: cfg.profile, Graphics: cfg.graphics, DevState: devState})
	return r
}

func (r *Runtime) SetTheme(t theme.Theme) {
	r.rt.Restyle(func() { r.theme = t })
}

func (r *Runtime) Theme() theme.Theme { return r.theme }

func (r *Runtime) Invalidate() { r.rt.Invalidate() }

func (r *Runtime) Quit() { r.rt.Quit() }

func (r *Runtime) Dispatch(f func()) { r.rt.Dispatch(f) }

func (r *Runtime) After(d time.Duration, fn func()) *Timer { return r.rt.After(d, fn) }

func (r *Runtime) Now() time.Time { return r.rt.Now() }

func (r *Runtime) Focus(key string) { r.rt.Focus(key) }

func (r *Runtime) HideFocusRings() { r.rt.HideFocusRings() }

func (r *Runtime) ScrollIntoView(key string) { r.rt.ScrollIntoView(key) }

func (r *Runtime) Viewport() image.Rectangle { return r.rt.Viewport() }

func (r *Runtime) ContentBox(e *Event) image.Rectangle { return r.rt.ContentBox(e.Current()) }

func (r *Runtime) Clicks() int { return r.rt.Clicks() }

func (r *Runtime) Copy(text string) { r.rt.Copy(text) }

func (r *Runtime) Widths() text.Widths { return r.rt.Widths() }

func (r *Runtime) Remember(key string, save func() string, restore func(string)) {
	r.rt.Remember(key, save, restore)
}

func (r *Runtime) Run(app func() Node) error {
	if !r.cfg.profileSet {
		return errors.New("twi: Backend needs a ColorProfile")
	}
	var b runtime.Backend = r.cfg.backend
	if b == nil {
		t, err := terminal.Enter(os.Stdin, os.Stdout, terminal.Options{})
		if err != nil {
			return err
		}
		b = terminalBackend{t}
	}
	return r.rt.Run(b, func() runtime.Tree {
		r.rt.Remember(konst.ThemeState, func() string { return themeKey(r.theme) }, r.restoreTheme)
		return app().runtimeTree()
	})
}

func themeKey(t theme.Theme) string {
	return t.Name + "/" + strconv.Itoa(int(t.Scheme))
}

func (r *Runtime) restoreTheme(key string) {
	themes := theme.Builtin()
	if r.cfg.theme != nil {
		themes = append([]theme.Theme{*r.cfg.theme}, themes...)
	}
	if i := slices.IndexFunc(themes, func(t theme.Theme) bool { return themeKey(t) == key }); i >= 0 {
		r.theme = themes[i]
	}
}

type terminalBackend struct{ *terminal.Backend }

func (t terminalBackend) Events() <-chan input.Event { return t.Backend.Events }

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
	return &Signal[T]{owner: rt.rt, value: value}
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
