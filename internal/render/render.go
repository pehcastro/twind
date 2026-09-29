package render

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

type Node struct {
	Text     string
	Classes  []string
	Children []Node
}

type Frame struct {
	Sheet    style.Sheet
	Width    int
	Height   layout.Length
	Sanitize func(raw string) scene.Text
}

type styledBox struct {
	box      *layout.Box
	computed style.ComputedStyle
	text     scene.Text
	children []*styledBox
}

func Render(root Node, f Frame) (*buffer.Buffer, error) {
	if f.Sanitize == nil {
		f.Sanitize = scene.Sanitize
	}
	styled, err := build(f, style.ComputedStyle{}, root)
	if err != nil {
		return nil, err
	}
	layout.Layout(styled.box, f.Width, f.Height)
	buf := buffer.New(f.Width, styled.box.BorderBox.H)
	paint.Paint(buf, styled.scene())
	return buf, nil
}

func (s *styledBox) scene() scene.Node {
	n := scene.New(s.box, s.computed, s.text)
	for _, c := range s.children {
		n.Children = append(n.Children, c.scene())
	}
	return n
}

func build(f Frame, parent style.ComputedStyle, n Node) (*styledBox, error) {
	computed := f.Sheet.Compute(parent, n.Classes)
	s, err := boxStyle(computed)
	if err != nil {
		return nil, err
	}
	box := &layout.Box{Style: s}
	var clean scene.Text
	if n.Text != "" {
		clean = f.Sanitize(n.Text)
	}
	if clean != (scene.Text{}) {
		naturalWidth, naturalHeight := clean.Size(math.MaxInt)
		wrapped := map[int][2]int{}
		box.Measure = func(availableWidth int) (int, int) {
			if availableWidth >= naturalWidth {
				return naturalWidth, naturalHeight
			}
			size, ok := wrapped[availableWidth]
			if !ok {
				size[0], size[1] = clean.Size(availableWidth)
				wrapped[availableWidth] = size
			}
			return size[0], size[1]
		}
	}
	out := &styledBox{box: box, computed: computed, text: clean}
	for _, c := range n.Children {
		child, err := build(f, computed, c)
		if err != nil {
			return nil, err
		}
		box.Children = append(box.Children, child.box)
		out.children = append(out.children, child)
	}
	return out, nil
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
		panic(fmt.Sprintf("render: unknown length unit %d", l.Unit))
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
		Inset:      layout.Insets{Top: length(s.Inset.Top), Right: length(s.Inset.Right), Bottom: length(s.Inset.Bottom), Left: length(s.Inset.Left)},
		ZIndex:     s.ZIndex,
	}
	if s.OverflowX != style.OverflowVisible || s.OverflowY != style.OverflowVisible {
		out.Overflow = layout.OverflowHidden
	}
	switch s.Position {
	case style.PositionStatic:
	case style.PositionRelative:
		out.Position = layout.PositionRelative
	case style.PositionAbsolute:
		out.Position = layout.PositionAbsolute
	case style.PositionFixed:
		out.Position = layout.PositionFixed
	case style.PositionSticky:
		unsupported = append(unsupported, "position sticky")
	default:
		panic(fmt.Sprintf("render: unknown position %d", s.Position))
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
