package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

//go:embed posts/*.md
var postFiles embed.FS

func main() {
	themeName := flag.String("theme", "twind-dark", "built-in theme, name-scheme: twind-dark, dream-light, and so on")
	pageName := flag.String("page", "home", "page open at start: home, projects, blog, contact, or a post's file name")
	flag.Parse()
	if err := run(*themeName, *pageName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(themeName, pageName string) error {
	t, ok := builtin(themeName)
	if !ok {
		return fmt.Errorf("-theme %q: not a built-in theme", themeName)
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
	if !s.open(pageName) {
		return fmt.Errorf("-page %q: not a page or a post", pageName)
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
