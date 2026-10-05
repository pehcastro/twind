package main

import (
	"flag"
	"fmt"
	"os"

	docsapp "github.com/pehcastro/twind/apps/documentation"
	"github.com/pehcastro/twind/twi"
)

func main() {
	var start docsapp.Start
	flag.StringVar(&start.Page, "page", "introduction", "page to open")
	flag.StringVar(&start.Theme, "theme", "twind-dark", "theme to open with, name-scheme")
	flag.Parse()
	if err := run(start); err != nil {
		fmt.Fprintln(os.Stderr, "docsdev:", err)
		os.Exit(1)
	}
}

func run(start docsapp.Start) error {
	sheet, err := docsapp.Styles()
	if err != nil {
		return err
	}
	rt := twi.New(twi.Styles(sheet))
	view, err := docsapp.New(rt, start)
	if err != nil {
		return err
	}
	return rt.Run(view)
}
