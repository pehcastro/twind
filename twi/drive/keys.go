package drive

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pehcastro/twind/twi/input"
)

func namedKeys() map[string]input.KeyEvent {
	return map[string]input.KeyEvent{
		"enter": {Key: input.KeyEnter}, "escape": {Key: input.KeyEscape}, "esc": {Key: input.KeyEscape},
		"tab": {Key: input.KeyTab}, "backspace": {Key: input.KeyBackspace}, "space": {Rune: ' '},
		"delete": {Key: input.KeyDelete}, "insert": {Key: input.KeyInsert},
		"home": {Key: input.KeyHome}, "end": {Key: input.KeyEnd},
		"pageup": {Key: input.KeyPageUp}, "pagedown": {Key: input.KeyPageDown},
		"up": {Key: input.KeyArrowUp}, "down": {Key: input.KeyArrowDown},
		"left": {Key: input.KeyArrowLeft}, "right": {Key: input.KeyArrowRight},
		"f1": {Key: input.KeyF1}, "f2": {Key: input.KeyF2}, "f3": {Key: input.KeyF3}, "f4": {Key: input.KeyF4},
		"f5": {Key: input.KeyF5}, "f6": {Key: input.KeyF6}, "f7": {Key: input.KeyF7}, "f8": {Key: input.KeyF8},
		"f9": {Key: input.KeyF9}, "f10": {Key: input.KeyF10}, "f11": {Key: input.KeyF11}, "f12": {Key: input.KeyF12},
	}
}

func modifierNames() map[string]input.Modifiers {
	return map[string]input.Modifiers{
		"shift": input.ModShift, "alt": input.ModAlt, "ctrl": input.ModCtrl, "meta": input.ModMeta, "super": input.ModMeta,
	}
}

func parseKey(name string) (input.KeyEvent, error) {
	if name == "" {
		return input.KeyEvent{}, errors.New("drive: empty key name")
	}
	mods, key := "", name
	if i := strings.LastIndex(name[:len(name)-1], "+"); i >= 0 {
		mods, key = name[:i], name[i+1:]
	}
	k, named := namedKeys()[strings.ToLower(key)]
	if !named {
		r, n := utf8.DecodeRuneInString(key)
		if n != len(key) || r == utf8.RuneError || unicode.IsControl(r) {
			return input.KeyEvent{}, fmt.Errorf("drive: unknown key %q in %q", key, name)
		}
		k.Rune = r
	}
	var err error
	k.Modifiers, err = parseModifiers(mods, name)
	return k, err
}

func parseModifiers(mods, name string) (input.Modifiers, error) {
	if mods == "" {
		return 0, nil
	}
	var all input.Modifiers
	for m := range strings.SplitSeq(mods, "+") {
		bit, ok := modifierNames()[strings.ToLower(m)]
		if !ok {
			return 0, fmt.Errorf("drive: unknown modifier %q in %q", m, name)
		}
		all |= bit
	}
	return all, nil
}
