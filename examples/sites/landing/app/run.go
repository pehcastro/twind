package app

import (
	"flag"
	"fmt"

	"github.com/pehcastro/twind/twi"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func Run(args []string) error {
	fs := flag.NewFlagSet("landing", flag.ExitOnError)
	name := fs.String("style", "platform", "page style open at start: platform, product, studio, event, store or project")
	at := fs.String("section", "", "section to open at, like a #fragment: Features, Pricing, Work")
	_ = fs.Parse(args)
	s, ok := named(*name)
	if !ok {
		return fmt.Errorf("-style %q: want platform, product, studio, event, store or project", *name)
	}
	sheet, err := Styles()
	if err != nil {
		return err
	}
	rt := twi.New(twi.Styles(sheet))
	app := landing(rt, s)
	if *at != "" {
		rt.ScrollIntoView(sectionKey + *at)
	}
	return rt.Run(app)
}
