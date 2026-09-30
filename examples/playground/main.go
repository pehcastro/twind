package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	page := flag.Int("page", 1, "page to open on, 1 to 5")
	themeName := flag.String("theme", "zinc-dark", "theme to open with, name-scheme")
	picker := flag.Bool("picker", false, "open with the theme picker showing")
	graphicsName := flag.String("graphics", "", "surface protocol, none, sixel, kitty or iterm2; empty detects it")
	focus := flag.String("focus", "input", "control focused at start: input, theme, or a page name")
	value := flag.String("value", "", "text the input starts with")
	flag.Parse()
	start := state{page: *page - 1, theme: themeIndex(*themeName), picker: *picker, focus: *focus}
	sheet, err := Styles()
	opts := []twi.RenderOption{twi.Fullscreen(), twi.Styles(sheet)}
	graphics, known := map[string]terminal.Graphics{"none": terminal.GraphicsNone, "sixel": terminal.GraphicsSixel, "kitty": terminal.GraphicsKitty, "iterm2": terminal.GraphicsITerm2}[*graphicsName]
	if known {
		opts = append(opts, twi.Graphics(graphics))
	}
	switch {
	case !known && *graphicsName != "":
		err = fmt.Errorf("-graphics %q: want none, sixel, kitty or iterm2", *graphicsName)
	case start.page < 0 || start.page >= len(pages()):
		err = fmt.Errorf("-page %d: want 1 to %d", *page, len(pages()))
	case start.theme < 0:
		err = fmt.Errorf("-theme %q: not a built-in theme", *themeName)
	case err == nil:
		cwd, _ := os.Getwd()
		env := env{cwd: cwd, profile: profileName(terminal.Profile(os.Stdout, os.Getenv)), size: func() string {
			w, h, err := terminal.Size(os.Stdout)
			if err != nil {
				return "no size"
			}
			return strconv.Itoa(w) + "x" + strconv.Itoa(h)
		}}
		rt := twi.New(opts...)
		err = rt.Run(playground(rt, env, start, *value))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func profileName(p color.Profile) string {
	switch p {
	case color.None:
		return "no colour"
	case color.Attributes:
		return "attributes"
	case color.ANSI16:
		return "16 colours"
	case color.ANSI256:
		return "256 colours"
	case color.TrueColor:
		return "truecolor"
	}
	panic("playground: unknown colour profile " + strconv.Itoa(int(p)))
}
