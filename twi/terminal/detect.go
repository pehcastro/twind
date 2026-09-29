package terminal

import (
	"errors"
	"io"
	"strings"

	"github.com/twind-dev/twind/twi/color"
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
	_, _, err := Size(w)
	return profileFor(err == nil, env)
}

func profileFor(terminal bool, env func(string) string) color.Profile {
	if env("NO_COLOR") != "" {
		return color.None
	}
	forced := color.None
	switch env("FORCE_COLOR") {
	case "":
		if !terminal {
			return color.None
		}
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
	case term == "dumb" && forced == color.None:
		return color.None
	case env("COLORTERM") == "truecolor", env("COLORTERM") == "24bit", env("WT_SESSION") != "":
		return color.TrueColor
	case strings.Contains(term, "256color"):
		return max(forced, color.ANSI256)
	}
	return max(forced, color.ANSI16)
}
