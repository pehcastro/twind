package terminal

import (
	"errors"
	"io"
	"strings"

	"github.com/pehcastro/twind/twi/color"
)

var errNotConsole = errors.New("terminal: writer is not a console")

func Size(w io.Writer) (width, height int, err error) {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return 0, 0, errNotConsole
	}
	return size(f.Fd())
}

func Profile(w io.Writer, env func(string) string) color.Profile {
	f, ok := w.(interface{ Fd() uintptr })
	if !ok {
		return profileFor(false, color.None, env)
	}
	_, _, err := size(f.Fd())
	return profileFor(err == nil, promised(f.Fd()), env)
}

func profileFor(terminal bool, console color.Profile, env func(string) string) color.Profile {
	forced := color.None
	switch env("FORCE_COLOR") {
	case "":
	case "0", "false":
		return color.None
	case "2":
		forced = color.ANSI256
	case "3":
		forced = color.TrueColor
	default:
		forced = color.ANSI16
	}
	term := env("TERM")
	switch {
	case forced == color.None && (!terminal || term == "dumb"):
		return color.None
	case env("NO_COLOR") != "":
		return color.Attributes
	case env("COLORTERM") == "truecolor", env("COLORTERM") == "24bit", env("WT_SESSION") != "":
		return color.TrueColor
	case strings.Contains(term, "256color"):
		return max(forced, console, color.ANSI256)
	}
	return max(forced, console, color.ANSI16)
}
