package ui

import (
	"slices"
	"strconv"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
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
	panels   []*Dialog
}

func NewToaster(rt *twi.Runtime) *Toaster { return &Toaster{Duration: konst.ToastDuration, rt: rt} }

func (t *Toaster) Avoid(panels ...*Dialog) {
	for _, p := range panels {
		if p.kind != sheet && p.kind != drawer {
			panic("ui: a toaster avoids sheets and drawers only")
		}
	}
	t.panels = append(t.panels, panels...)
}

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
	s.tick = t.rt.After(konst.ToastTick, func() {
		if s.left -= konst.ToastTick; s.left > 0 {
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
	shown := t.toasts[max(len(t.toasts)-konst.VisibleToasts, 0):]
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
	var before, after []twi.NodeOption
	top := 0
	for _, p := range t.panels {
		if !p.Open {
			continue
		}
		switch p.side {
		case Left:
			before = []twi.NodeOption{part("h-full shrink-0 "+sideWidth, nil)}
		case Right:
			after = []twi.NodeOption{part("h-full shrink-0 "+sideWidth, nil)}
		case Bottom:
			if box := p.box.Bounds(); !box.Empty() {
				top = min(top, box.Min.Y-t.rt.Viewport().Max.Y)
			}
		case Top:
		}
	}
	stack := part("flex flex-col w-52 max-w-full min-w-0 pointer-events-auto "+gap, children)
	return part("fixed w-full h-full z-50 flex flex-row pointer-events-none", slices.Concat([]twi.NodeOption{twi.At(0, top)}, before,
		[]twi.NodeOption{part("flex flex-col grow min-w-0 justify-end items-end px-2 pb-1", []twi.NodeOption{stack})}, after))
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
	closeButton := part(button(Ghost, SizeIcon, "text-muted-foreground"), []twi.NodeOption{twi.OnClick(func(*twi.Event) { t.dismiss(s) }), icon("✕", "")})
	return part("flex flex-row items-start gap-1 shrink-0 rounded-lg border bg-popover pl-2 pr-1 py-1 text-popover-foreground shadow-lg", slices.Concat(
		[]twi.NodeOption{twi.Key(strconv.Itoa(s.id))},
		mark,
		[]twi.NodeOption{part("flex flex-col grow min-w-0", body)},
		action,
		[]twi.NodeOption{closeButton},
	))
}
