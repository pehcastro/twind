package ui

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

const (
	inputRing         = "shadow-[0_0_0_1px_var(--color-input)]"
	primaryRing       = "shadow-[0_0_0_1px_var(--color-primary)]"
	invalidRing       = "shadow-[0_0_0_1px_var(--color-destructive)]"
	focusRing         = "focus-visible:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	withinRing        = "focus-within:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	activeRing        = "data-[active=true]:shadow-[0_0_0_1px_var(--color-ring),0_0_0_3px_color-mix(in_oklab,var(--color-ring)_50%,transparent)]"
	invalidFocusRing  = "focus-visible:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:focus-visible:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
	invalidWithinRing = "focus-within:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:focus-within:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
	invalidActiveRing = "data-[active=true]:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_20%,transparent)] dark:data-[active=true]:shadow-[0_0_0_1px_var(--color-destructive),0_0_0_3px_color-mix(in_oklab,var(--color-destructive)_40%,transparent)]"
)

type ringAt uint8

const (
	onSelf ringAt = iota
	onGroup
	onItem
)

type control struct {
	Disabled, Invalid bool
	rt                *twi.Runtime
	focused           bool
}

func (c *control) ring(idle string, at ringAt) string {
	if c.Invalid {
		return invalidRing + " " + pick("ring", at, map[ringAt]string{onSelf: invalidFocusRing, onGroup: invalidWithinRing, onItem: invalidActiveRing})
	}
	return idle + " " + pick("ring", at, map[ringAt]string{onSelf: focusRing, onGroup: withinRing, onItem: activeRing})
}

func (c *control) dataActive(here bool) twi.NodeOption {
	return twi.Data("active", strconv.FormatBool(c.focused && here))
}

func (c *control) behave(keys func(input.KeyEvent) bool) []twi.NodeOption {
	options := []twi.NodeOption{twi.Class("disabled:opacity-50")}
	if c.Disabled {
		return append(options, twi.Disabled())
	}
	focus := func(on bool) func() {
		return func() {
			c.focused = on
			c.rt.Invalidate()
		}
	}
	options = append(options, twi.Focusable(), twi.OnFocus(focus(true)), twi.OnBlur(focus(false)))
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
