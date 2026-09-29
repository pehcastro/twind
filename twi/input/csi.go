package input

import (
	"bytes"
	"strconv"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/input"
)

type cmd struct{ prefix, inter, final byte }

func command(body []byte, final byte) Event {
	c := cmd{final: final}
	if len(body) > 0 && body[0] >= '<' {
		c.prefix, body = body[0], body[1:]
	}
	if n := len(body); n > 0 && body[n-1] < '0' {
		c.inter, body = body[n-1], body[:n-1]
	}
	p, ok := parse(body)
	if !ok {
		return nil
	}
	switch c {
	case cmd{'<', 0, 'M'}, cmd{'<', 0, 'm'}:
		return mouse(p, final == 'm')
	case cmd{'?', 0, 'c'}:
		return reply(ReplyPrimaryAttributes, p)
	case cmd{'>', 0, 'c'}:
		return reply(ReplySecondaryAttributes, p)
	case cmd{'?', '$', 'y'}:
		return reply(ReplyMode, p)
	case cmd{'?', 0, 'u'}:
		return reply(ReplyKeyboardFlags, p)
	case cmd{'?', 0, 'R'}:
		return reply(ReplyCursorPosition, p)
	case cmd{0, 0, 'R'}:
		if len(p) == 2 {
			return reply(ReplyCursorPosition, p)
		}
	case cmd{0, 0, 'I'}, cmd{0, 0, 'O'}:
		if len(p) == 0 {
			return FocusEvent{Focused: final == 'I'}
		}
		return nil
	case cmd{0, 0, 't'}:
		if len(p) != konst.InBandResizeParams || p.get(0, 0, 0) != konst.InBandResize {
			return nil
		}
		return ResizeEvent{Width: p.get(2, 0, 0), Height: p.get(1, 0, 0)}
	case cmd{0, 0, 'u'}:
		return kitty(p)
	case cmd{0, 0, '~'}:
		return tilde(p)
	case cmd{0, 0, 'Z'}:
		return withModifiers(KeyEvent{Key: KeyTab, Modifiers: ModShift}, p)
	}
	if c.prefix != 0 || c.inter != 0 || p.get(0, 0, konst.LegacyKeyParam) != konst.LegacyKeyParam {
		return nil
	}
	for _, f := range finals() {
		if f.b == final {
			return withModifiers(KeyEvent{Key: f.key}, p)
		}
	}
	return nil
}

type params [][]int

func parse(body []byte) (params, bool) {
	if len(body) == 0 {
		return nil, true
	}
	var p params
	for _, field := range bytes.Split(body, []byte{';'}) {
		var subs []int
		for _, s := range bytes.Split(field, []byte{':'}) {
			n := -1
			if len(s) > 0 {
				v, err := strconv.Atoi(string(s))
				if err != nil || s[0] < '0' || s[0] > '9' {
					return nil, false
				}
				n = v
			}
			subs = append(subs, n)
		}
		p = append(p, subs)
	}
	return p, true
}

func (p params) get(i, j, missing int) int {
	if i < len(p) && j < len(p[i]) && p[i][j] >= 0 {
		return p[i][j]
	}
	return missing
}

func reply(kind ReplyKind, p params) Event {
	values := make([]int, len(p))
	for i := range p {
		values[i] = p.get(i, 0, 0)
	}
	return ReplyEvent{Kind: kind, Params: values}
}

func withModifiers(k KeyEvent, p params) KeyEvent {
	m := max(p.get(1, 0, 1)-1, 0)
	k.Modifiers |= Modifiers(m & konst.KittyModifierMask)
	if m&konst.KittyMeta != 0 {
		k.Modifiers |= ModMeta
	}
	switch p.get(1, 1, konst.EventPress) {
	case konst.EventRepeat:
		k.Repeat = true
	case konst.EventRelease:
		k.Release = true
	}
	return k
}

func kitty(p params) Event {
	k, ok := codeKey(p.get(0, 0, -1))
	if !ok {
		return nil
	}
	k = withModifiers(k, p)
	text, hasText := printable(p.get(2, 0, -1))
	shifted, hasShifted := printable(p.get(0, 1, -1))
	shift := k.Modifiers&ModShift != 0
	switch {
	case k.Key != KeyRune:
	case hasText:
		k.Rune = text
	case shift && hasShifted:
		k.Rune = shifted
	case shift:
		k.Rune = unicode.ToUpper(k.Rune)
	}
	return k
}

func tilde(p params) Event {
	n := p.get(0, 0, 0)
	switch n {
	case konst.PasteStart:
		return pasteStart{}
	case konst.ModifyOtherKeys:
		k, ok := codeKey(p.get(2, 0, -1))
		if !ok {
			return nil
		}
		return withModifiers(k, p)
	}
	keys := tildeKeys()
	if n >= len(keys) || keys[n] == KeyRune {
		return nil
	}
	return withModifiers(KeyEvent{Key: keys[n]}, p)
}

func codeKey(n int) (KeyEvent, bool) {
	switch n {
	case konst.CR:
		return KeyEvent{Key: KeyEnter}, true
	case konst.TAB:
		return KeyEvent{Key: KeyTab}, true
	case konst.ESC:
		return KeyEvent{Key: KeyEscape}, true
	case konst.BS, konst.DEL:
		return KeyEvent{Key: KeyBackspace}, true
	}
	keypad := [...]KeyEvent{
		{Rune: '0'}, {Rune: '1'}, {Rune: '2'}, {Rune: '3'}, {Rune: '4'},
		{Rune: '5'}, {Rune: '6'}, {Rune: '7'}, {Rune: '8'}, {Rune: '9'},
		{Rune: '.'}, {Rune: '/'}, {Rune: '*'}, {Rune: '-'}, {Rune: '+'},
		{Key: KeyEnter}, {Rune: '='}, {Rune: ','},
		{Key: KeyArrowLeft}, {Key: KeyArrowRight}, {Key: KeyArrowUp}, {Key: KeyArrowDown},
		{Key: KeyPageUp}, {Key: KeyPageDown}, {Key: KeyHome}, {Key: KeyEnd},
		{Key: KeyInsert}, {Key: KeyDelete},
	}
	if i := n - konst.KittyKeypadFirst; i >= 0 && i < len(keypad) {
		return keypad[i], true
	}
	r, ok := printable(n)
	return KeyEvent{Rune: r}, ok
}

func printable(n int) (rune, bool) {
	if n < 0 || n > unicode.MaxRune {
		return 0, false
	}
	return rune(n), unicode.IsPrint(rune(n))
}

func mouse(p params, release bool) Event {
	b, x, y := p.get(0, 0, -1), p.get(1, 0, 0), p.get(2, 0, 0)
	if len(p) != konst.MouseParams || b < 0 || x < 1 || y < 1 || b&konst.MouseExtraButtons != 0 {
		return nil
	}
	button := b & konst.MouseButtonMask
	ev := MouseEvent{
		X:         x - 1,
		Y:         y - 1,
		Button:    MouseLeft + MouseButton(button),
		Modifiers: Modifiers(b>>konst.MouseModifierShift) & konst.MouseModifierMask,
	}
	if button == konst.MouseNoButton {
		ev.Button = MouseNone
	}
	switch {
	case b&konst.MouseWheel != 0:
		ev.Button, ev.Action = MouseWheelUp+MouseButton(button), MouseScroll
	case b&konst.MouseMotion != 0:
		ev.Action = MouseMove
	case release:
		ev.Action = MouseRelease
	}
	return ev
}
