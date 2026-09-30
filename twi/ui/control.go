package ui

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const (
	inputRing        = "shadow-[0_0_0_1px_var(--color-input)]"
	primaryRing      = "shadow-[0_0_0_1px_var(--color-primary)]"
	focusRing        = "shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	invalidRing      = "shadow-[0_0_0_1px_var(--color-destructive)]"
	invalidFocusRing = "shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
)

type control struct {
	Disabled, Invalid bool
	rt                *twi.Runtime
	focused           bool
}

func (c *control) ring(idle string, here bool) string {
	focused := c.focused && here && !c.Disabled
	switch {
	case focused && c.Invalid:
		return invalidFocusRing
	case focused:
		return focusRing
	case c.Invalid:
		return invalidRing
	}
	return idle
}

func (c *control) behave(keys func(input.KeyEvent) bool) []twi.NodeOption {
	if c.Disabled {
		return []twi.NodeOption{twi.Disabled(), twi.Class("opacity-50")}
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

func arrow(k input.KeyEvent) int {
	switch k.Key {
	case input.KeyArrowDown, input.KeyArrowRight:
		return 1
	case input.KeyArrowUp, input.KeyArrowLeft:
		return -1
	}
	return 0
}

func flip(on *bool, onChange func(bool), accept func(input.KeyEvent) bool) func(input.KeyEvent) bool {
	return func(k input.KeyEvent) bool {
		if !accept(k) {
			return false
		}
		*on = !*on
		notify(onChange, *on)
		return true
	}
}
