package ui

import (
	"slices"
	"strings"
	"unicode"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/text"
)

type InputOTP struct {
	control
	Length   int
	Value    string
	OnChange func(string)
}

func NewInputOTP(rt *twi.Runtime, length int) *InputOTP {
	return &InputOTP{control: control{rt: rt}, Length: length}
}

func (o *InputOTP) Slot(i int) twi.Node {
	chars := slices.Collect(text.Graphemes(o.Value))
	active := i == min(len(chars), o.Length-1)
	var content []twi.NodeOption
	switch {
	case i < len(chars):
		content = []twi.NodeOption{twi.Text(chars[i])}
	case active && o.focused:
		content = []twi.NodeOption{part("text-foreground", []twi.NodeOption{twi.Text("│")})}
	}
	return part("flex flex-row w-3 h-1 shrink-0 items-center justify-center dark:bg-input/30 "+o.ring(inputRing, active), content)
}

func (o *InputOTP) Node(options ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", append(o.behave(o.key), options...))
}

func (o *InputOTP) key(k input.KeyEvent) bool {
	typed := k.Key == input.KeyRune && k.Modifiers&^input.ModShift == 0
	if !typed && k.Key != input.KeyBackspace {
		return false
	}
	chars := slices.Collect(text.Graphemes(o.Value))
	switch {
	case !typed && len(chars) > 0:
		chars = chars[:len(chars)-1]
	case typed && len(chars) < o.Length && (unicode.IsLetter(k.Rune) || unicode.IsDigit(k.Rune)):
		chars = append(chars, string(k.Rune))
	default:
		return true
	}
	o.Value = strings.Join(chars, "")
	notify(o.OnChange, o.Value)
	return true
}

func InputOTPGroup(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}

func InputOTPSeparator() twi.Node {
	return part("flex flex-row w-2 justify-center text-muted-foreground", []twi.NodeOption{twi.Text("-")})
}
