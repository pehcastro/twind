package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	gallery "github.com/pehcastro/twind/examples/gallery/app"
	playground "github.com/pehcastro/twind/examples/playground/app"
	landing "github.com/pehcastro/twind/examples/sites/landing/app"
	portfolio "github.com/pehcastro/twind/examples/sites/portfolio/app"
)

type demo struct {
	name, about string
	run         func([]string) error
}

func try(args []string, w io.Writer) error {
	demos := []demo{
		{"docs", "this documentation, with live component previews", func(a []string) error { return docs(a, w) }},
		{"landing", "six landing pages: platform, product, studio, event, store, project", landing.Run},
		{"portfolio", "a personal site with a blog written in Markdown", portfolio.Run},
		{"playground", "every component on seven pages, with a theme picker", playground.Run},
		{"gallery", "a dashboard: tables, forms, overlays and settings", gallery.Run},
	}
	var list strings.Builder
	list.WriteString("usage: twind try name [arguments]\n\nRuns a demo built into twind. name -h prints its flags.\n\n")
	for _, d := range demos {
		fmt.Fprintf(&list, "  %-10s %s\n", d.name, d.about)
	}
	if len(args) == 0 || args[0] == "-h" || args[0] == "-help" {
		_, err := io.WriteString(w, list.String())
		return err
	}
	i := slices.IndexFunc(demos, func(d demo) bool { return d.name == args[0] })
	if i < 0 {
		fmt.Fprintf(os.Stderr, "twind try: no demo %q\n\n%s", args[0], list.String())
		return usageError("no demo " + args[0])
	}
	return demos[i].run(args[1:])
}
