package ui

import (
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const (
	fade              = "disabled:opacity-50 "
	inputRing         = "shadow-[0_0_0_1px_var(--color-input)]"
	primaryRing       = "shadow-[0_0_0_1px_var(--color-primary)]"
	invalidRing       = "shadow-[0_0_0_1px_var(--color-destructive)]"
	focusRing         = "focus-visible:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	activeRing        = "in-focus-visible:data-[active=true]:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	invalidFocusRing  = "focus-visible:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:focus-visible:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
	invalidActiveRing = "in-focus-visible:data-[active=true]:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:in-focus-visible:data-[active=true]:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
	fieldEdge         = "border-[0.5px] border-input focus-visible:border-ring"
	areaEdge          = "border border-input focus-visible:border-ring"
	groupEdge         = "border border-input has-focus-visible:border-ring"
	invalidEdge       = "border-destructive"
)

type ringAt uint8

const (
	onSelf ringAt = iota
	onItem
)

type control struct {
	Disabled, Invalid bool
	rt                *twi.Runtime
	focused           bool
}

func (c *control) ring(idle string, at ringAt) string {
	if c.Invalid {
		return invalidRing + " " + pick("ring", at, map[ringAt]string{onSelf: invalidFocusRing, onItem: invalidActiveRing})
	}
	return idle + " " + pick("ring", at, map[ringAt]string{onSelf: focusRing, onItem: activeRing})
}

func (c *control) edge(valid string) string {
	if c.Invalid {
		width, _, _ := strings.Cut(valid, " ")
		return width + " " + invalidEdge
	}
	return valid
}

func (c *control) dataActive(here bool) twi.NodeOption {
	return twi.Data("active", strconv.FormatBool(c.focused && here))
}

func (c *control) behave(keys func(input.KeyEvent) bool) []twi.NodeOption {
	if c.Disabled {
		return []twi.NodeOption{twi.Disabled()}
	}
	focus := func(on bool) func() {
		return func() {
			c.focused = on
			c.rt.Invalidate()
		}
	}
	options := []twi.NodeOption{twi.Focusable(), twi.OnFocus(focus(true)), twi.OnBlur(focus(false))}
	if keys == nil {
		return options
	}
	return append(options, keyDown(c.rt, keys))
}

func keyDown(rt *twi.Runtime, keys func(input.KeyEvent) bool) twi.NodeOption {
	return twi.OnKeyDown(func(e *twi.Event) {
		if !e.Key.Release && keys(e.Key) {
			e.PreventDefault()
			e.StopPropagation()
			rt.Invalidate()
		}
	})
}

func notify[T any](onChange func(T), v T) {
	if onChange != nil {
		onChange(v)
	}
}

func space(k input.KeyEvent) bool {
	return k.Key == input.KeyRune && k.Rune == ' ' && k.Modifiers == 0
}

func press(k input.KeyEvent) bool {
	return space(k) || k.Key == input.KeyEnter && k.Modifiers == 0
}

func typed(k input.KeyEvent) bool {
	return k.Key == input.KeyRune && k.Modifiers&^input.ModShift == 0
}

func typeahead(n, from int, r rune, text func(int) string) int {
	for i := range n {
		if next := (from + 1 + i) % n; strings.HasPrefix(strings.ToLower(text(next)), strings.ToLower(string(r))) {
			return next
		}
	}
	return from
}

func arrow(k input.KeyEvent) int {
	switch k.Key {
	case input.KeyArrowDown, input.KeyArrowRight:
		return 1
	case input.KeyArrowUp, input.KeyArrowLeft:
		return -1
	}
	return 0
}

func (c *control) click(act func()) twi.NodeOption {
	return twi.OnClick(func(*twi.Event) {
		if !c.Disabled {
			act()
			c.rt.Invalidate()
		}
	})
}

func (c *control) pressable(accept func(input.KeyEvent) bool, act func()) []twi.NodeOption {
	return append(c.behave(func(k input.KeyEvent) bool {
		if accept(k) {
			act()
		}
		return press(k)
	}), c.click(act))
}

func flip(on *bool, onChange func(bool)) func() {
	return func() {
		*on = !*on
		notify(onChange, *on)
	}
}
