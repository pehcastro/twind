package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/twind-dev/twind/twi"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	name := flag.String("style", "platform", "page style open at start: platform, product or studio")
	at := flag.String("section", "", "section to open at, like a #fragment: Features, Pricing, Work")
	flag.Parse()
	s, ok := named(*name)
	if !ok {
		fail(fmt.Errorf("-style %q: want platform, product or studio", *name))
	}
	sheet, err := Styles()
	if err != nil {
		fail(err)
	}
	rt := twi.New(twi.Fullscreen(), twi.Styles(sheet))
	app := landing(rt, s)
	if *at != "" {
		rt.ScrollIntoView(sectionKey + *at)
	}
	if err := rt.Run(app); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
