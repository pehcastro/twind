package input

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/input"
)

type Decoder struct {
	pending  []byte
	paste    []byte
	pasting  bool
	overflow bool
}

type pasteStart struct{}

func (pasteStart) event() {}

func (d *Decoder) Decode(b []byte) []Event {
	d.pending = append(d.pending, b...)
	return d.drain(false)
}

func (d *Decoder) Quiet() []Event {
	return d.drain(true)
}

func (d *Decoder) drain(quiet bool) []Event {
	var out []Event
	b := d.pending
	for len(b) > 0 {
		if d.pasting {
			end := bytes.Index(b, []byte(konst.PasteEnd))
			if end < 0 {
				keep := max(0, len(b)-len(konst.PasteEnd)+1)
				d.paste, b = append(d.paste, b[:keep]...), b[keep:]
				break
			}
			out = append(out, PasteEvent{Text: string(append(d.paste, b[:end]...))})
			d.paste, d.pasting, b = d.paste[:0], false, b[end+len(konst.PasteEnd):]
			continue
		}
		n, ev := next(b, quiet)
		if n == 0 {
			break
		}
		b = b[n:]
		if d.overflow {
			d.overflow, ev = false, nil
		}
		switch ev.(type) {
		case nil:
		case pasteStart:
			d.pasting = true
		default:
			out = append(out, ev)
		}
	}
	if len(b) > konst.MaxSequenceBytes {
		b, d.overflow = append(b[:3:3], b[len(b)-1]), true
	}
	d.pending = append(d.pending[:0], b...)
	return out
}

func next(b []byte, quiet bool) (int, Event) {
	c := b[0]
	switch {
	case c == konst.ESC:
		return escape(b, quiet)
	case c < konst.SP || c == konst.DEL:
		return 1, control(c)
	case c < utf8.RuneSelf:
		return 1, KeyEvent{Rune: rune(c)}
	case !utf8.FullRune(b) && !quiet:
		return 0, nil
	}
	r, n := utf8.DecodeRune(b)
	if (r == utf8.RuneError && n == 1) || unicode.IsControl(r) {
		return n, nil
	}
	return n, KeyEvent{Rune: r}
}

func control(c byte) KeyEvent {
	switch c {
	case konst.NUL:
		return KeyEvent{Rune: ' ', Modifiers: ModCtrl}
	case konst.BS, konst.DEL:
		return KeyEvent{Key: KeyBackspace}
	case konst.TAB:
		return KeyEvent{Key: KeyTab}
	case konst.CR:
		return KeyEvent{Key: KeyEnter}
	}
	if c <= 'z'-konst.CtrlLetterBase {
		return KeyEvent{Rune: rune(c) + konst.CtrlLetterBase, Modifiers: ModCtrl}
	}
	return KeyEvent{Rune: rune(c) + konst.CtrlSymbolBase, Modifiers: ModCtrl}
}

func escape(b []byte, quiet bool) (int, Event) {
	if len(b) == 1 {
		if quiet {
			return 1, KeyEvent{Key: KeyEscape}
		}
		return 0, nil
	}
	var n int
	var ev Event
	switch b[1] {
	case '[':
		n, ev = csi(b)
	case 'O':
		n, ev = ss3(b)
	case ']', 'P', '_', '^', 'X':
		n = controlString(b)
	case konst.ESC:
		if len(b) > 2 && b[2] != '[' && b[2] != 'O' {
			return 1, KeyEvent{Key: KeyEscape}
		}
		return alt(escape(b[1:], quiet))
	default:
		return alt(next(b[1:], quiet))
	}
	switch {
	case n == 0 && !quiet:
		return 0, nil
	case n == 0 && len(b) > 2:
		return len(b), nil
	case n == 0 || (n == 2 && ev == nil):
		return 2, KeyEvent{Rune: rune(b[1]), Modifiers: ModAlt}
	}
	return n, ev
}

func alt(n int, ev Event) (int, Event) {
	if n == 0 {
		return 0, nil
	}
	if ev == nil {
		return n + 1, nil
	}
	k, ok := ev.(KeyEvent)
	if !ok || k.Modifiers&ModAlt != 0 {
		return 1, KeyEvent{Key: KeyEscape}
	}
	k.Modifiers |= ModAlt
	return n + 1, k
}

func ss3(b []byte) (int, Event) {
	switch {
	case len(b) < 3:
		return 0, nil
	case b[2] < '@' || b[2] > '~':
		return 2, nil
	}
	for _, f := range finals() {
		if f.b == b[2] {
			return 3, KeyEvent{Key: f.key}
		}
	}
	return 3, nil
}

func controlString(b []byte) int {
	for i := 2; i < len(b); i++ {
		switch {
		case b[i] == konst.BEL:
			return i + 1
		case b[i] != konst.ESC:
		case i+1 == len(b):
			return 0
		case b[i+1] == '\\':
			return i + 2
		default:
			return i
		}
	}
	return 0
}

func csi(b []byte) (int, Event) {
	i := 2
	for i < len(b) && b[i] >= konst.SP && b[i] < '@' {
		i++
	}
	switch {
	case i == len(b):
		return 0, nil
	case b[i] < konst.SP || b[i] > '~':
		return i, nil
	case i >= konst.MaxSequenceBytes:
		return i + 1, nil
	}
	return i + 1, command(b[2:i], b[i])
}
