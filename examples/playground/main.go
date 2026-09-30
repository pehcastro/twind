package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/twind-dev/twind/examples/playground/app"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
)

func main() {
	var start app.Start
	flag.StringVar(&start.Page, "page", "1", "page to open on: 1 to 7, a page name or a component name")
	flag.StringVar(&start.Open, "open", "", "overlay open at start: dialog or sheet")
	flag.BoolVar(&start.Slow, "slow", false, "the dialog opens and closes over 8 s instead of 200 ms, to see it mid-motion")
	flag.StringVar(&start.Theme, "theme", "zinc-dark", "theme to open with, name-scheme")
	flag.BoolVar(&start.Picker, "picker", false, "open with the theme picker showing")
	graphicsName := flag.String("graphics", "", "surface protocol, none, sixel, kitty or iterm2; empty detects it")
	flag.StringVar(&start.Focus, "focus", "input", "control focused at start: input, theme, or a page name")
	flag.StringVar(&start.Value, "value", "", "text the input starts with")
	flag.Parse()
	if err := run(start, *graphicsName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(start app.Start, graphicsName string) error {
	sheet, err := app.Styles()
	if err != nil {
		return err
	}
	opts := []twi.RenderOption{twi.Fullscreen(), twi.Styles(sheet)}
	graphics, known := map[string]terminal.Graphics{"none": terminal.GraphicsNone, "sixel": terminal.GraphicsSixel, "kitty": terminal.GraphicsKitty, "iterm2": terminal.GraphicsITerm2}[graphicsName]
	if known {
		opts = append(opts, twi.Graphics(graphics))
	} else if graphicsName != "" {
		return fmt.Errorf("-graphics %q: want none, sixel, kitty or iterm2", graphicsName)
	}
	cwd, _ := os.Getwd()
	env := app.Env{Cwd: cwd, Profile: profileName(terminal.Profile(os.Stdout, os.Getenv)), Size: func() string {
		w, h, err := terminal.Size(os.Stdout)
		if err != nil {
			return "no size"
		}
		return strconv.Itoa(w) + "x" + strconv.Itoa(h)
	}}
	rt := twi.New(opts...)
	playground, err := app.New(rt, env, start)
	if err != nil {
		return err
	}
	return rt.Run(playground)
}

func profileName(p color.Profile) string {
	switch p {
	case color.None:
		return "no colour"
	case color.Attributes:
		return "attributes"
	case color.ANSI16:
		return "16 colours"
	case color.ANSI256:
		return "256 colours"
	case color.TrueColor:
		return "truecolor"
	}
	panic("playground: unknown colour profile " + strconv.Itoa(int(p)))
}
