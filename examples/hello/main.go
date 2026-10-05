package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func main() {
	width := flag.Int("width", 0, "columns; 0 is the terminal width, or 80 when not a terminal")
	truecolor := flag.Bool("truecolor", false, "force 24-bit colour instead of detecting it")
	flag.Parse()
	sheet, err := Styles()
	if err == nil {
		opts := []twi.RenderOption{twi.Styles(sheet), twi.Width(*width)}
		if *truecolor {
			opts = append(opts, twi.ColorProfile(color.TrueColor))
		}
		err = twi.Render(os.Stdout, App(), opts...)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func App() twi.Node {
	return twi.Element(
		twi.Class(
			"flex flex-col gap-2 p-4 "+
				"bg-zinc-950 text-zinc-100",
		),

		twi.Text("Hello Twind"),

		twi.Element(
			twi.Class(
				"border rounded-lg p-2",
			),

			twi.Text("Terminal DOM"),
		),
	)
}
