package app

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func Run(args []string) error {
	fs := flag.NewFlagSet("gallery", flag.ExitOnError)
	themeName := fs.String("theme", "twind-dark", "built-in theme, name-scheme: twind-dark, dream-light, and so on")
	pageName := fs.String("page", "dashboard", "page open at start: dashboard, forms, overlays or settings")
	open := fs.String("open", "", "on the overlays page, the overlay open at start: "+strings.Join(openable(), ", "))
	profile := fs.String("profile", "", "force a colour profile: truecolor, 256, 16 or attributes")
	cells := fs.Bool("cells", false, "draw surfaces in cells, without terminal graphics")
	_ = fs.Parse(args)
	s, err := parse(*themeName, *pageName, *open)
	if err != nil {
		return err
	}
	var opts []twi.Option
	if *cells {
		opts = append(opts, twi.Graphics(terminal.GraphicsNone))
	}
	if *profile != "" {
		p, ok := map[string]color.Profile{"truecolor": color.TrueColor, "256": color.ANSI256, "16": color.ANSI16, "attributes": color.Attributes}[*profile]
		if !ok {
			return fmt.Errorf("-profile %q: want truecolor, 256, 16 or attributes", *profile)
		}
		opts = append(opts, twi.ColorProfile(p))
	}
	sheet, err := Styles()
	if err != nil {
		return err
	}
	rt := twi.New(append(opts, twi.Styles(sheet))...)
	return rt.Run(gallery(rt, s))
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
