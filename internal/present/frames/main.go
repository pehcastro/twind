package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"time"

	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/present"
	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/terminal"
)

func main() {
	cols := flag.Int("cols", 90, "columns")
	rows := flag.Int("rows", 28, "rows")
	cellW := flag.Int("cell-width", 10, "cell width in pixels")
	cellH := flag.Int("cell-height", 20, "cell height in pixels")
	hold := flag.Duration("hold", 4*time.Second, "time each tree stays on screen")
	flag.Parse()
	if err := run(*cols, *rows, image.Pt(*cellW, *cellH), *hold, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "frames:", err)
		os.Exit(1)
	}
}

func run(cols, rows int, cell image.Point, hold time.Duration, names []string) error {
	graphics := map[string]terminal.Graphics{"": terminal.GraphicsSixel, "sixel": terminal.GraphicsSixel, "none": terminal.GraphicsNone, "kitty": terminal.GraphicsKitty, "iterm2": terminal.GraphicsITerm2}
	mode, ok := graphics[os.Getenv("TWIND_GRAPHICS")]
	if !ok {
		return fmt.Errorf("TWIND_GRAPHICS=%q, want none, sixel, iterm2 or kitty", os.Getenv("TWIND_GRAPHICS"))
	}
	trees := map[string][]render.Node{
		"hello":  {demo.Hello()},
		"dialog": {demo.Dialog()},
		"list":   {demo.List(0), demo.List(2)},
	}
	sheet, err := demo.Styles()
	if err != nil {
		return err
	}
	for _, name := range names {
		frames, ok := trees[name]
		if !ok {
			return errors.New("unknown tree " + name + ", want hello, dialog or list")
		}
		if _, err := os.Stdout.WriteString(termkonst.LeaveScreen + termkonst.EnterScreen); err != nil {
			return err
		}
		s := present.Screen{Out: os.Stdout, Profile: color.TrueColor, Graphics: mode, Cell: cell, Sync: true}
		for _, n := range frames {
			root, err := render.Scene(n, render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}})
			if err == nil {
				err = s.Frame(root, cols, rows)
			}
			if err != nil {
				return err
			}
		}
		time.Sleep(hold)
	}
	return nil
}
