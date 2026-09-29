package twi

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

type Node struct {
	text     string
	classes  []string
	children []Node
}

type NodeOption interface{ apply(*Node) }

func (n Node) apply(parent *Node) { parent.children = append(parent.children, n) }

type classList []string

func (c classList) apply(n *Node) { n.classes = append(n.classes, c...) }

func Element(options ...NodeOption) Node {
	var n Node
	for _, o := range options {
		o.apply(&n)
	}
	return n
}

func Text(s string) Node { return Node{text: s} }

func Class(classes ...string) NodeOption {
	var list classList
	for _, c := range classes {
		list = append(list, strings.Fields(c)...)
	}
	return list
}

type RenderOption func(*renderConfig)

type renderConfig struct {
	width      int
	profile    color.Profile
	profileSet bool
	sheet      style.Sheet
}

func Width(cells int) RenderOption { return func(c *renderConfig) { c.width = cells } }

func ColorProfile(p color.Profile) RenderOption {
	return func(c *renderConfig) { c.profile, c.profileSet = p, true }
}

func Styles(sheet style.Sheet) RenderOption { return func(c *renderConfig) { c.sheet = sheet } }

func RenderString(node Node, opts ...RenderOption) string {
	var out strings.Builder
	if err := Render(&out, node, opts...); err != nil {
		panic(err)
	}
	return out.String()
}

func Render(w io.Writer, node Node, opts ...RenderOption) (err error) {
	var cfg renderConfig
	for _, o := range opts {
		o(&cfg)
	}
	termWidth, _, sizeErr := terminal.Size(w)
	if cfg.width <= 0 {
		cfg.width = konst.DefaultWidth
		if sizeErr == nil {
			cfg.width = termWidth
		}
	}
	if !cfg.profileSet {
		cfg.profile = terminal.Profile(w, os.Getenv)
	}
	var styled []styledBox
	root, err := build(cfg.sheet, style.ComputedStyle{}, node, &styled)
	if err != nil {
		return err
	}
	layout.Layout(root, cfg.width, layout.Length{})
	nodes := make([]scene.Node, len(styled))
	for i, s := range styled {
		nodes[i] = scene.New(s.box, s.computed, s.text)
	}
	buf := buffer.New(cfg.width, root.BorderBox.H)
	paint.Paint(buf, nodes)
	if f, ok := w.(*os.File); ok && sizeErr == nil && cfg.profile != color.None {
		restore, vtErr := terminal.EnableVirtualTerminal(f)
		if vtErr != nil {
			return vtErr
		}
		defer func() { err = errors.Join(err, restore()) }()
	}
	return (&terminal.Writer{Out: w, Profile: cfg.profile}).Static(buf)
}

type styledBox struct {
	box      *layout.Box
	computed style.ComputedStyle
	text     string
}

func build(sheet style.Sheet, parent style.ComputedStyle, n Node, out *[]styledBox) (*layout.Box, error) {
	computed := sheet.Compute(parent, n.classes)
	s, err := boxStyle(computed)
	if err != nil {
		return nil, err
	}
	box := &layout.Box{Style: s}
	clean := text.Sanitize(n.text, text.RemoveBidi)
	*out = append(*out, styledBox{box, computed, clean})
	if clean != "" {
		box.Measure = func(availableWidth int) (width, height int) {
			lines := text.Wrap(clean, availableWidth)
			for _, line := range lines {
				width = max(width, text.Width(line))
			}
			return width, len(lines)
		}
	}
	for _, c := range n.children {
		child, err := build(sheet, computed, c, out)
		if err != nil {
			return nil, err
		}
		box.Children = append(box.Children, child)
	}
	return box, nil
}

func boxStyle(s style.ComputedStyle) (layout.Style, error) {
	var unsupported []string
	cells := func(name string, l style.Length) int {
		if l.Unit != style.Cells {
			unsupported = append(unsupported, name)
		}
		return int(math.Round(l.Value))
	}
	edges := func(name string, e style.Edges) layout.Edges {
		return layout.Edges{Top: cells(name, e.Top), Right: cells(name, e.Right), Bottom: cells(name, e.Bottom), Left: cells(name, e.Left)}
	}
	length := func(l style.Length) layout.Length {
		switch l.Unit {
		case style.Cells:
			return layout.Length{Unit: layout.Cells, Value: int(math.Round(l.Value))}
		case style.Percent:
			return layout.Length{Unit: layout.Percent, Value: int(math.Round(l.Value))}
		case style.Auto, style.None:
			return layout.Length{}
		}
		panic(fmt.Sprintf("twi: unknown length unit %d", l.Unit))
	}
	align := func(name string, a style.Align) layout.Align {
		if a == style.AlignBaseline {
			unsupported = append(unsupported, name+" baseline")
			return layout.AlignAuto
		}
		return [...]layout.Align{
			style.AlignAuto:    layout.AlignAuto,
			style.AlignStretch: layout.AlignStretch,
			style.AlignStart:   layout.AlignStart,
			style.AlignEnd:     layout.AlignEnd,
			style.AlignCenter:  layout.AlignCenter,
		}[a]
	}
	out := layout.Style{
		Justify: [...]layout.Justify{
			style.JustifyStart:   layout.JustifyStart,
			style.JustifyEnd:     layout.JustifyEnd,
			style.JustifyCenter:  layout.JustifyCenter,
			style.JustifyBetween: layout.JustifyBetween,
			style.JustifyAround:  layout.JustifyAround,
			style.JustifyEvenly:  layout.JustifyEvenly,
		}[s.Justify],
		AlignItems: align("align-items", s.AlignItems),
		AlignSelf:  align("align-self", s.AlignSelf),
		Grow:       int(math.Round(s.Grow)),
		Shrink:     int(math.Round(s.Shrink)),
		Basis:      length(s.Basis),
		Width:      length(s.Width),
		Height:     length(s.Height),
		MinWidth:   length(s.MinWidth),
		MinHeight:  length(s.MinHeight),
		MaxWidth:   length(s.MaxWidth),
		MaxHeight:  length(s.MaxHeight),
		RowGap:     cells("row-gap", s.RowGap),
		ColumnGap:  cells("column-gap", s.ColumnGap),
		Padding:    edges("padding", s.Padding),
		Margin:     edges("margin", s.Margin),
		Border:     edges("border-width", s.BorderWidth),
	}
	if s.BorderStyle == style.BorderNone {
		out.Border = layout.Edges{}
	}
	switch {
	case s.Display == style.DisplayNone:
		out.Display = layout.DisplayNone
	case s.Display == style.DisplayBlock:
		out.Direction = layout.Column
	case s.Display != style.DisplayFlex:
		unsupported = append(unsupported, "display")
	case s.Direction == style.Column:
		out.Direction = layout.Column
	case s.Direction != style.Row:
		unsupported = append(unsupported, "flex-direction")
	}
	if len(unsupported) > 0 {
		return layout.Style{}, fmt.Errorf("twi: layout does not support this %s yet", strings.Join(slices.Compact(unsupported), ", "))
	}
	return out, nil
}
