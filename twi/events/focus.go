package events

import (
	"cmp"
	"slices"

	"github.com/twind-dev/twind/twi/input"
)

type FocusManager[N comparable] struct {
	current N
	focused bool
	scopes  []scope[N]
}

type scope[N comparable] struct {
	root, previous N
	hadPrevious    bool
}

func (f *FocusManager[N]) Current() (N, bool) { return f.current, f.focused }

func (f *FocusManager[N]) Set(t Tree[N], n N) bool {
	if !eligible(t, n) || !f.inScope(t, n) {
		return false
	}
	f.move(t, n)
	return true
}

func (f *FocusManager[N]) Key(t Tree[N], k input.KeyEvent) *Event[N] {
	e := &Event[N]{Type: KeyDown, Key: k}
	if k.Release {
		e.Type = KeyUp
	}
	target := f.root(t)
	if f.focused {
		target = f.current
	}
	Dispatch(t, target, e)
	if e.prevented || k.Release || k.Key != input.KeyTab {
		return e
	}
	order := f.order(t)
	if len(order) == 0 {
		return e
	}
	i := -1
	if f.focused {
		i = slices.Index(order, f.current)
	}
	switch {
	case k.Modifiers&input.ModShift == 0:
		i = (i + 1) % len(order)
	case i <= 0:
		i = len(order) - 1
	default:
		i--
	}
	f.move(t, order[i])
	return e
}

func (f *FocusManager[N]) Open(t Tree[N], root N) {
	f.scopes = append(f.scopes, scope[N]{root: root, previous: f.current, hadPrevious: f.focused})
	if order := f.order(t); len(order) > 0 {
		f.move(t, order[0])
		return
	}
	f.blur(t)
}

func (f *FocusManager[N]) Close(t Tree[N]) {
	s := f.scopes[len(f.scopes)-1]
	f.scopes = f.scopes[:len(f.scopes)-1]
	if s.hadPrevious && eligible(t, s.previous) && f.inScope(t, s.previous) {
		f.move(t, s.previous)
		return
	}
	f.blur(t)
}

func (f *FocusManager[N]) move(t Tree[N], n N) {
	if f.focused && f.current == n {
		return
	}
	f.blur(t)
	f.current, f.focused = n, true
	Dispatch(t, n, &Event[N]{Type: Focus})
}

func (f *FocusManager[N]) blur(t Tree[N]) {
	if !f.focused {
		return
	}
	f.focused = false
	Dispatch(t, f.current, &Event[N]{Type: Blur})
}

func (f *FocusManager[N]) root(t Tree[N]) N {
	if len(f.scopes) == 0 {
		return t.Root()
	}
	return f.scopes[len(f.scopes)-1].root
}

func (f *FocusManager[N]) inScope(t Tree[N], n N) bool {
	root := f.root(t)
	for ok := true; ok; n, ok = t.Parent(n) {
		if n == root {
			return true
		}
	}
	return false
}

func (f *FocusManager[N]) order(t Tree[N]) []N {
	var positive, zero []N
	var walk func(N)
	walk = func(n N) {
		if eligible(t, n) && t.TabIndex(n) > 0 {
			positive = append(positive, n)
		}
		if eligible(t, n) && t.TabIndex(n) == 0 {
			zero = append(zero, n)
		}
		for _, c := range t.Children(n) {
			walk(c)
		}
	}
	walk(f.root(t))
	slices.SortStableFunc(positive, func(a, b N) int { return cmp.Compare(t.TabIndex(a), t.TabIndex(b)) })
	return append(positive, zero...)
}

func eligible[N comparable](t Tree[N], n N) bool {
	return t.Focusable(n) && !t.Disabled(n)
}
