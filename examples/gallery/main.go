package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	themeName := flag.String("theme", "twind-dark", "built-in theme, palette-scheme: zinc-dark, violet-light, and so on")
	pageName := flag.String("page", "dashboard", "page open at start: dashboard, forms, overlays or settings")
	open := flag.String("open", "", "on the overlays page, the overlay open at start: "+strings.Join(openable(), ", "))
	profile := flag.String("profile", "", "force a colour profile: truecolor, 256, 16 or attributes")
	cells := flag.Bool("cells", false, "draw surfaces in cells, without terminal graphics")
	flag.Parse()
	s, err := parse(*themeName, *pageName, *open)
	if err != nil {
		fail(err)
	}
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
	sheet, err := Styles()
	if err != nil {
		fail(err)
	}
	rt := twi.New(append(opts, twi.Styles(sheet))...)
	if err := rt.Run(gallery(rt, s)); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type start struct {
	theme theme.Theme
	page  page
	open  string
}

func parse(themeName, pageName, open string) (start, error) {
	t, ok := builtin(themeName)
	if !ok {
		return start{}, fmt.Errorf("-theme %q: not a built-in theme", themeName)
	}
	p := slices.IndexFunc(pages(), func(p page) bool { return strings.EqualFold(p.String(), pageName) })
	if p < 0 {
		return start{}, fmt.Errorf("-page %q: want dashboard, forms, overlays or settings", pageName)
	}
	if open != "" && !slices.Contains(openable(), open) {
		return start{}, fmt.Errorf("-open %q: want %s", open, strings.Join(openable(), ", "))
	}
	return start{t, page(p), open}, nil
}

func builtin(name string) (theme.Theme, bool) {
	for _, t := range theme.Builtin() {
		if t.Name+map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme] == name {
			return t, true
		}
	}
	return theme.Theme{}, false
}
