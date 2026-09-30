package render

import (
	"fmt"
	"math"
	"reflect"
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
	Look     paint.Look
}

type styledBox struct {
	box      *layout.Box
	classes  []string
	computed style.ComputedStyle
	raw      string
	text     scene.Text
	natural  [2]int
	sizes    map[int][2]int
	children []*styledBox
	node     scene.Node
	painted  bool
}

func Render(root Node, f Frame) (*buffer.Buffer, error) {
	tree, err := Scene(root, f)
	if err != nil {
		return nil, err
	}
	height := tree.Bounds.H
	if f.Height.Unit == layout.Cells {
		height = f.Height.Value
	}
	buf := buffer.New(f.Width, height)
	paint.Paint(buf, tree, f.Look)
	return buf, nil
}

type Tree struct {
	root              *styledBox
	width             int
	height            layout.Length
	restyle, relayout bool
	cascades          int
}

func (t *Tree) Restyle() { t.restyle = true }

func Scene(root Node, f Frame) (scene.Node, error) { return new(Tree).Scene(root, f) }

func (t *Tree) Scene(root Node, f Frame) (scene.Node, error) {
	if f.Sanitize == nil {
		f.Sanitize = scene.Sanitize
	}
	t.relayout = t.root == nil || f.Width != t.width || f.Height != t.height
	styled, err := t.build(f, t.root, style.ComputedStyle{}, false, root)
	t.root, t.width, t.height, t.restyle = styled, f.Width, f.Height, false
	if err != nil {
		return scene.Node{}, err
	}
	if t.relayout {
		layout.Layout(styled.box, f.Width, f.Height)
	}
	return styled.scene(t.relayout), nil
}

func (s *styledBox) scene(moved bool) scene.Node {
	if s.painted && !moved {
		return s.node
	}
	n := scene.New(s.box, s.computed, s.text)
	for _, c := range s.children {
		n.Children = append(n.Children, c.scene(moved))
	}
	s.node, s.painted = n, true
	return n
}

func (t *Tree) build(f Frame, prev *styledBox, parent style.ComputedStyle, parentChanged bool, n Node) (*styledBox, error) {
	s := prev
	if s == nil {
		s = &styledBox{box: &layout.Box{}}
		t.relayout = true
	}
	changed := false
	if prev == nil || t.restyle || parentChanged || !slices.Equal(s.classes, n.Classes) {
		t.cascades++
		computed := f.Sheet.Compute(parent, n.Classes)
		ls, err := boxStyle(computed)
		if err != nil {
			return nil, err
		}
		if prev == nil || ls != s.box.Style {
			s.box.Style, t.relayout = ls, true
		}
		if changed = prev == nil || t.restyle || !reflect.DeepEqual(computed, s.computed); changed {
			s.computed, s.painted = computed, false
		}
		s.classes = n.Classes
	}
	if n.Text != s.raw && s.retext(f, n.Text) {
		t.relayout = true
	}
	old := s.children
	if len(old) != len(n.Children) {
		s.children, s.box.Children = make([]*styledBox, len(n.Children)), make([]*layout.Box, len(n.Children))
		s.painted = false
	}
	for i, c := range n.Children {
		var p *styledBox
		if i < len(old) {
			p = old[i]
		}
		child, err := t.build(f, p, s.computed, changed, c)
		if err != nil {
			return nil, err
		}
		s.children[i], s.box.Children[i] = child, child.box
		s.painted = s.painted && child.painted
	}
	return s, nil
}

func (s *styledBox) retext(f Frame, raw string) (moved bool) {
	var next styledBox
	if raw != "" {
		next.text = f.Sanitize(raw)
	}
	moved = (s.text == scene.Text{}) != (next.text == scene.Text{})
	if next.text != (scene.Text{}) {
		next.natural[0], next.natural[1] = next.text.Size(math.MaxInt)
		next.sizes = map[int][2]int{}
		for width, size := range s.sizes {
			w, h := next.measure(width)
			moved = moved || size != [2]int{w, h}
		}
	}
	s.raw, s.text, s.natural, s.sizes, s.painted = raw, next.text, next.natural, next.sizes, false
	s.box.Measure = nil
	if s.text != (scene.Text{}) {
		s.box.Measure = s.measure
	}
	return moved
}

func (s *styledBox) measure(availableWidth int) (int, int) {
	size, ok := s.sizes[availableWidth]
	if !ok {
		size = s.natural
		if availableWidth < size[0] {
			size[0], size[1] = s.text.Size(availableWidth)
		}
		s.sizes[availableWidth] = size
	}
	return size[0], size[1]
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
