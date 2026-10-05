package input

import (
	"fmt"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/input"
)

type final struct {
	b   byte
	key Key
}

func finals() [10]final {
	return [10]final{
		{'A', KeyArrowUp}, {'B', KeyArrowDown}, {'C', KeyArrowRight}, {'D', KeyArrowLeft},
		{'H', KeyHome}, {'F', KeyEnd},
		{'P', KeyF1}, {'Q', KeyF2}, {'R', KeyF3}, {'S', KeyF4},
	}
}

func tildeKeys() [30]Key {
	return [30]Key{
		1: KeyHome, 2: KeyInsert, 3: KeyDelete, 4: KeyEnd, 5: KeyPageUp, 6: KeyPageDown, 7: KeyHome, 8: KeyEnd,
		11: KeyF1, 12: KeyF2, 13: KeyF3, 14: KeyF4, 15: KeyF5,
		17: KeyF6, 18: KeyF7, 19: KeyF8, 20: KeyF9, 21: KeyF10,
		23: KeyF11, 24: KeyF12, 29: KeyMenu,
	}
}

func Encode(k KeyEvent) []byte {
	plain := k.Modifiers == 0 && !k.Repeat && !k.Release
	event := konst.EventPress
	switch {
	case k.Release:
		event = konst.EventRelease
	case k.Repeat:
		event = konst.EventRepeat
	}
	tail := fmt.Sprintf("%d:%d", 1+int(k.Modifiers), event)
	code := int(k.Rune)
	switch k.Key {
	case KeyEnter:
		code = konst.CR
	case KeyTab:
		code = konst.TAB
	case KeyEscape:
		code = konst.ESC
	case KeyBackspace:
		code = konst.DEL
	case KeyRune:
	default:
		return navigation(k.Key, plain, tail)
	}
	if plain {
		return utf8.AppendRune(nil, rune(code))
	}
	return fmt.Appendf(nil, "%s%d;%su", konst.CSI, code, tail)
}

func navigation(key Key, plain bool, tail string) []byte {
	for _, f := range finals() {
		if f.key == key && plain {
			return append([]byte(konst.SS3), f.b)
		}
	}
	for n, k := range tildeKeys() {
		if k == key {
			if plain {
				return fmt.Appendf(nil, "%s%d~", konst.CSI, n)
			}
			return fmt.Appendf(nil, "%s%d;%s~", konst.CSI, n, tail)
		}
	}
	for _, f := range finals() {
		if f.key == key {
			return fmt.Appendf(nil, "%s%d;%s%c", konst.CSI, konst.LegacyKeyParam, tail, f.b)
		}
	}
	return nil
}
