package app

import (
	"embed"
	"flag"
	"fmt"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/theme"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

//go:embed posts/*.md
var postFiles embed.FS

func Run(args []string) error {
	fs := flag.NewFlagSet("portfolio", flag.ExitOnError)
	themeName := fs.String("theme", "twind-dark", "built-in theme, name-scheme: twind-dark, dream-light, and so on")
	pageName := fs.String("page", "home", "page open at start: home, projects, blog, contact, or a post's file name")
	_ = fs.Parse(args)
	t, ok := builtin(*themeName)
	if !ok {
		return fmt.Errorf("-theme %q: not a built-in theme", *themeName)
	}
	posts, err := load(postFiles)
	if err != nil {
		return err
	}
	sheet, err := Styles()
	if err != nil {
		return err
	}
	rt := twi.New(twi.Fullscreen(), twi.Styles(sheet))
	s := newSite(rt, t, posts)
	if !s.open(*pageName) {
		return fmt.Errorf("-page %q: not a page or a post", *pageName)
	}
	return rt.Run(s.view)
}

func builtin(name string) (theme.Theme, bool) {
	for _, t := range theme.Builtin() {
		if t.Name+map[theme.Scheme]string{theme.Light: "-light", theme.Dark: "-dark"}[t.Scheme] == name {
			return t, true
		}
	}
	return theme.Theme{}, false
}
