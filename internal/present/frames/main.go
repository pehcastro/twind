package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/present"
	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

func main() {
	cols := flag.Int("cols", 90, "columns")
	rows := flag.Int("rows", 28, "rows")
	cellW := flag.Int("cell-width", 10, "cell width in pixels")
	cellH := flag.Int("cell-height", 20, "cell height in pixels")
	hold := flag.Duration("hold", 4*time.Second, "time each tree stays on screen")
	margins := flag.Bool("margins", true, "the terminal honours DECLRMM left and right margins")
	flag.Parse()
	if err := run(*cols, *rows, image.Pt(*cellW, *cellH), *hold, *margins, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "frames:", err)
		os.Exit(1)
	}
}

func run(cols, rows int, cell image.Point, hold time.Duration, margins bool, names []string) error {
	graphics := map[string]terminal.Graphics{"": terminal.GraphicsSixel, "sixel": terminal.GraphicsSixel, "none": terminal.GraphicsNone, "kitty": terminal.GraphicsKitty, "iterm2": terminal.GraphicsITerm2}
	mode, ok := graphics[os.Getenv("TWIND_GRAPHICS")]
	if !ok {
		return fmt.Errorf("TWIND_GRAPHICS=%q, want none, sixel, iterm2 or kitty", os.Getenv("TWIND_GRAPHICS"))
	}
	trees := map[string][]render.Node{
		"hello":  {demo.Hello()},
		"page":   {demo.Page()},
		"dialog": {demo.Dialog()},
		"list":   {demo.List(0), demo.List(2)},
		"cards":  {demo.Cards("", 3), demo.Cards("w-40", 3)},
	}
	sheet, err := demo.Styles()
	if err != nil {
		return err
	}
	for _, name := range names {
		kind, arg, _ := strings.Cut(name, ":")
		frames, ok := trees[kind]
		var tree render.Tree
		var steps []func()
		if n, err := strconv.Atoi(arg); err == nil {
			steps, ok = map[string][]func(){
				"scroll": slices.Repeat([]func(){func() { tree.ScrollBy(demo.ScrollerPath, 0, 1) }}, n),
				"jump":   {func() { tree.ScrollTo(demo.ScrollerPath, 0, n) }},
				"reveal": {func() { tree.ScrollIntoView(append(slices.Clone(demo.ScrollerPath), n)) }},
			}[kind]
			frames = slices.Repeat([]render.Node{demo.Scroller(200)}, len(steps)+1)
		}
		if !ok {
			return errors.New("unknown tree " + name + ", want hello, page, dialog, list, cards, scroll:N, jump:N or reveal:N")
		}
		if _, err := os.Stdout.WriteString(termkonst.LeaveScreen + termkonst.EnterScreen); err != nil {
			return err
		}
		s := present.Screen{Out: os.Stdout, Profile: color.TrueColor, Graphics: mode, Cell: cell, Sync: true, Margins: margins}
		for i, n := range frames {
			if i > 0 && i <= len(steps) {
				steps[i-1]()
			}
			root, err := tree.Scene(n, render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}})
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
