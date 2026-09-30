package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/style"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

func main() {
	cols := flag.Int("cols", 100, "columns")
	rows := flag.Int("rows", 30, "rows")
	top := flag.Bool("top", false, "open the popover in the top layer")
	hold := flag.Duration("hold", 8*time.Second, "time the frame stays on screen")
	flag.Parse()
	if err := run(*cols, *rows, *top, *hold); err != nil {
		fmt.Fprintln(os.Stderr, "toplayer:", err)
		os.Exit(1)
	}
}

func run(cols, rows int, top bool, hold time.Duration) error {
	sheet, err := styles()
	if err != nil {
		return err
	}
	title, layer := "before: the popover stays in the card's flow", 0
	if top {
		title, layer = "after: the popover opens in the top layer", 1
	}
	var items []render.Node
	for _, s := range []string{"New file", "Open", "Save", "Save as", "Rename", "Delete"} {
		items = append(items, render.Node{Text: s})
	}
	popover := el("popover", items...)
	popover.TopLayer = layer
	root := el("page",
		render.Node{Text: title},
		el("card", render.Node{Text: "Card: relative overflow-hidden"}, el("button", render.Node{Text: "Actions"}), popover),
		el("later", render.Node{Text: "Later sibling: relative z-10"}),
	)
	buf, err := render.Render(root, render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}})
	if err != nil {
		return err
	}
	restore, err := terminal.EnableVirtualTerminal(os.Stdout)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.WriteString(termkonst.EnterScreen); err != nil {
		return err
	}
	w := terminal.Writer{Out: os.Stdout, Profile: color.TrueColor, Sync: true}
	err = w.Diff(buffer.New(cols, rows), buf)
	time.Sleep(hold)
	_, leave := os.Stdout.WriteString(termkonst.LeaveScreen)
	return errors.Join(err, leave, restore())
}

func el(classes string, children ...render.Node) render.Node {
	return render.Node{Classes: strings.Fields(classes), Children: children}
}

func styles() (style.Sheet, error) {
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	rgb := func(r, g, b uint8) color.Color {
		return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: 255}}
	}
	fill := func(c color.Color) style.Declaration {
		return style.Declaration{Property: style.PropBackground, Color: c}
	}
	at := func(p style.Position) style.Declaration {
		return style.Declaration{Property: style.PropPosition, Position: p}
	}
	size := func(p style.Property, n float64) style.Declaration {
		return style.Declaration{Property: p, Length: cells(n)}
	}
	border := func(c color.Color) []style.Declaration {
		return []style.Declaration{
			size(style.PropBorderTopWidth, 1), size(style.PropBorderRightWidth, 1), size(style.PropBorderBottomWidth, 1), size(style.PropBorderLeftWidth, 1),
			{Property: style.PropBorderStyle, BorderStyle: style.BorderSingle}, {Property: style.PropBorderColor, Color: c},
			{Property: style.PropRadius, Radius: style.RadiusMd},
		}
	}
	pad := func(x, y float64) []style.Declaration {
		return []style.Declaration{size(style.PropPaddingLeft, x), size(style.PropPaddingRight, x), size(style.PropPaddingTop, y), size(style.PropPaddingBottom, y)}
	}
	shadow := style.Declaration{Property: style.PropShadow, Shadows: []style.Shadow{{Y: 10, Blur: 15, Spread: -3, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{A: 90}}}}}
	return style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "page", Decls: append(pad(2, 1), fill(rgb(9, 9, 11)), style.Declaration{Property: style.PropColor, Color: rgb(250, 250, 250)},
			size(style.PropRowGap, 1), style.Declaration{Property: style.PropHeight, Length: style.Length{Unit: style.Percent, Value: 100}})},
		{Class: "card", Decls: append(append(border(rgb(63, 63, 70)), pad(1, 0)...), at(style.PositionRelative), fill(rgb(24, 24, 27)),
			size(style.PropWidth, 44), size(style.PropHeight, 6), style.Declaration{Property: style.PropShrink},
			style.Declaration{Property: style.PropOverflowX, Overflow: style.OverflowHidden}, style.Declaration{Property: style.PropOverflowY, Overflow: style.OverflowHidden})},
		{Class: "button", Decls: append(pad(1, 0), fill(rgb(250, 250, 250)), style.Declaration{Property: style.PropColor, Color: rgb(24, 24, 27)},
			style.Declaration{Property: style.PropAlignSelf, Align: style.AlignStart}, style.Declaration{Property: style.PropWidth, Length: style.Length{Unit: style.FitContent}})},
		{Class: "popover", Decls: append(append(border(rgb(82, 82, 91)), pad(1, 0)...), at(style.PositionAbsolute), size(style.PropTop, 3), size(style.PropLeft, 2),
			size(style.PropWidth, 22), fill(rgb(39, 39, 42)), shadow)},
		{Class: "later", Decls: append(append(border(rgb(153, 27, 27)), pad(1, 0)...), at(style.PositionRelative), style.Declaration{Property: style.PropZIndex, Number: 10},
			fill(rgb(69, 10, 10)), size(style.PropWidth, 60), size(style.PropHeight, 8), style.Declaration{Property: style.PropShrink})},
	})
}
