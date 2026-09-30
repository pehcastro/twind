package ui

import (
	"slices"
	"strconv"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const (
	toastDuration = 4 * time.Second
	toastTick     = 100 * time.Millisecond
	visibleToasts = 3
)

type ToastKind uint8

const (
	ToastDefault ToastKind = iota
	ToastSuccess
	ToastError
)

type ToastAction struct {
	Label   string
	OnClick func()
}

type toast struct {
	id                 int
	kind               ToastKind
	title, description string
	action             ToastAction
	left               time.Duration
	tick               *twi.Timer
}

type Toaster struct {
	Duration time.Duration
	rt       *twi.Runtime
	toasts   []*toast
	made     int
	hovered  bool
}

func NewToaster(rt *twi.Runtime) *Toaster { return &Toaster{Duration: toastDuration, rt: rt} }

func (t *Toaster) Show(title, description string, action ToastAction) {
	t.add(ToastDefault, title, description, action)
}

func (t *Toaster) Success(title, description string, action ToastAction) {
	t.add(ToastSuccess, title, description, action)
}

func (t *Toaster) Error(title, description string, action ToastAction) {
	t.add(ToastError, title, description, action)
}

func (t *Toaster) add(kind ToastKind, title, description string, action ToastAction) {
	t.made++
	s := &toast{id: t.made, kind: kind, title: title, description: description, action: action, left: t.Duration}
	t.toasts = append(t.toasts, s)
	if !t.hovered {
		t.run(s)
	}
	t.rt.Invalidate()
}

func (t *Toaster) run(s *toast) {
	s.tick = t.rt.After(toastTick, func() {
		if s.left -= toastTick; s.left > 0 {
			t.run(s)
			return
		}
		t.dismiss(s)
	})
}

func (s *toast) stop() {
	if s.tick != nil {
		s.tick.Stop()
		s.tick = nil
	}
}

func (t *Toaster) dismiss(s *toast) {
	s.stop()
	t.toasts = slices.DeleteFunc(t.toasts, func(o *toast) bool { return o == s })
	t.hovered = t.hovered && len(t.toasts) > 0
	t.rt.Invalidate()
}

func (t *Toaster) hover(on bool) {
	if t.hovered == on {
		return
	}
	t.hovered = on
	for _, s := range t.toasts {
		if on {
			s.stop()
		} else {
			t.run(s)
		}
	}
	t.rt.Invalidate()
}

func (t *Toaster) Node() twi.Node {
	if len(t.toasts) == 0 {
		return closed()
	}
	newest := twi.OnKey(func(k input.KeyEvent) {
		if !k.Release && escape(k) && len(t.toasts) > 0 {
			t.dismiss(t.toasts[len(t.toasts)-1])
		}
	})
	children := []twi.NodeOption{newest, twi.OnPointerEnter(func() { t.hover(true) }), twi.OnPointerLeave(func() { t.hover(false) })}
	shown := t.toasts[max(len(t.toasts)-visibleToasts, 0):]
	for i, s := range shown {
		depth := len(shown) - 1 - i
		if t.hovered || depth == 0 {
			children = append(children, t.toast(s))
			continue
		}
		children = append(children, part("h-1 shrink-0 rounded-t-lg border-x border-t bg-popover "+[]string{"", "mx-1", "mx-2"}[depth], []twi.NodeOption{twi.Key(strconv.Itoa(s.id))}))
	}
	gap := ""
	if t.hovered {
		gap = "gap-1"
	}
	return part("fixed bottom-1 right-2 z-50 flex flex-col w-52 "+gap, children)
}

func (t *Toaster) toast(s *toast) twi.Node {
	var mark []twi.NodeOption
	switch s.kind {
	case ToastDefault:
	case ToastSuccess:
		mark = []twi.NodeOption{icon("✓", "shrink-0")}
	case ToastError:
		mark = []twi.NodeOption{icon("⊗", "shrink-0")}
	default:
		panic("ui: unknown toast kind")
	}
	body := []twi.NodeOption{part("font-medium", []twi.NodeOption{twi.Text(s.title)})}
	if s.description != "" {
		body = append(body, part("text-muted-foreground", []twi.NodeOption{twi.Text(s.description)}))
	}
	var action []twi.NodeOption
	if s.action.Label != "" {
		action = []twi.NodeOption{Button(Default, SizeXS, twi.OnClick(func(*twi.Event) {
			if s.action.OnClick != nil {
				s.action.OnClick()
			}
			t.dismiss(s)
		}), twi.Text(s.action.Label))}
	}
	closeButton := part("absolute top-0 right-1 text-muted-foreground hover:text-foreground", []twi.NodeOption{twi.OnClick(func(*twi.Event) { t.dismiss(s) }), icon("✕", "")})
	return part("relative flex flex-row items-center gap-1 shrink-0 rounded-lg border bg-popover px-2 py-1 text-popover-foreground shadow-lg", slices.Concat(
		[]twi.NodeOption{twi.Key(strconv.Itoa(s.id))},
		mark,
		[]twi.NodeOption{part("flex flex-col grow min-w-0", body)},
		action,
		[]twi.NodeOption{closeButton},
	))
}
