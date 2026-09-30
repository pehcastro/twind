package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	themeName := flag.String("theme", "zinc-light", "built-in theme, name-scheme, zinc-dark for the dark page")
	profile := flag.String("profile", "", "force a colour profile: truecolor, 256, 16 or attributes")
	cells := flag.Bool("cells", false, "draw surfaces in cells, without terminal graphics")
	flag.Parse()
	opts := []twi.RenderOption{twi.Fullscreen()}
	if *cells {
		opts = append(opts, twi.Graphics(terminal.GraphicsNone))
	}
	if *profile != "" {
		p, ok := map[string]color.Profile{"truecolor": color.TrueColor, "256": color.ANSI256, "16": color.ANSI16, "attributes": color.Attributes}[*profile]
		if !ok {
			fail(fmt.Errorf("-profile %q: want truecolor, 256, 16 or attributes", *profile))
		}
		opts = append(opts, twi.ColorProfile(p))
	}
	t, ok := builtin(*themeName)
	if !ok {
		fail(fmt.Errorf("-theme %q: not a built-in theme", *themeName))
	}
	sheet, err := Styles()
	if err != nil {
		fail(err)
	}
	rt := twi.New(append(opts, twi.Styles(sheet), twi.Theme(t))...)
	if err := rt.Run(gallery(rt)); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func builtin(name string) (theme.Theme, bool) {
	for _, t := range theme.Builtin() {
		if t.Name+map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme] == name {
			return t, true
		}
	}
	return theme.Theme{}, false
}
