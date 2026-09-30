package main

import (
	"io"

	docsapp "github.com/twind-dev/twind/apps/documentation"
	"github.com/twind-dev/twind/twi"
)

const docsHelp = `Opens the Twind documentation, itself a fullscreen Twind app: pages in a sidebar, live
component previews next to their Go code, Ctrl+K to search, t to pick a theme, Ctrl+C to quit.`

func docs(args []string, _ io.Writer) error {
	set := flags("docs", "[-page name] [-theme name]", docsHelp)
	var start docsapp.Start
	set.StringVar(&start.Page, "page", "introduction", "page to open: introduction, installation, theming, button, dialog or tabs")
	set.StringVar(&start.Theme, "theme", "zinc-dark", "theme to open with, name-scheme, such as zinc-light")
	positional, err := parse(set, args)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		set.Usage()
		return usageError("docs takes no arguments")
	}
	sheet, err := docsapp.Styles()
	if err != nil {
		return err
	}
	rt := twi.New(twi.Fullscreen(), twi.Styles(sheet))
	view, err := docsapp.New(rt, start)
	if err != nil {
		return err
	}
	return rt.Run(view)
}
