package render

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"slices"
	"strings"

	skonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
)

type Node struct {
	Text     string
	Classes  []string
	State    *style.NodeState
	Children []Node
}

type Frame struct {
	Sheet    style.Sheet
	Width    int
	Height   layout.Length
	Sanitize func(raw string) scene.Text
	Look     paint.Look
	Cell     image.Point
}

type styledBox struct {
	box      *layout.Box
	classes  []string
	state    style.NodeState
	computed style.ComputedStyle
	truncate bool
	nowrap   bool
	reverse  bool
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
	root                       *styledBox
	width                      int
	height                     layout.Length
	cell                       image.Point
	band                       int
	restyle, relayout, crossed bool
	cascades                   int
}

func (t *Tree) Restyle() { t.restyle = true }

func Scene(root Node, f Frame) (scene.Node, error) { return new(Tree).Scene(root, f) }

func (t *Tree) Scene(root Node, f Frame) (scene.Node, error) {
	if f.Sanitize == nil {
		f.Sanitize = scene.Sanitize
	}
	if f.Cell.X <= 0 || f.Cell.Y <= 0 {
		f.Cell = image.Pt(skonst.NominalCellX, skonst.NominalCellY)
	}
	f.Sheet = f.Sheet.WithColumns(f.Width)
	band := f.Sheet.Band(f.Width)
	t.relayout = t.relayout || t.root == nil || f.Width != t.width || f.Height != t.height
	t.restyle = t.restyle || f.Cell != t.cell
	t.crossed = band != t.band
	styled, err := t.build(f, t.root, style.ComputedStyle{}, false, root)
	t.root, t.width, t.height, t.cell, t.band, t.restyle = styled, f.Width, f.Height, f.Cell, band, false
	if err != nil {
		return scene.Node{}, err
	}
	moved := t.relayout
	if moved {
		layout.Layout(styled.box, f.Width, f.Height)
	}
	t.relayout = false
	return styled.scene(moved), nil
}

func (t *Tree) ScrollBy(path []int, dx, dy int) bool {
	b := t.scroller(path)
	return b != nil && t.scrollTo(b, b.ScrollX+dx, b.ScrollY+dy)
}

func (t *Tree) ScrollTo(path []int, x, y int) bool {
	b := t.scroller(path)
	return b != nil && t.scrollTo(b, x, y)
}

func (t *Tree) ScrollIntoView(path []int) bool {
	s, scrollers := t.find(path)
	if s == nil {
		return false
	}
	target, moved := s.box.BorderBox, false
	for _, b := range slices.Backward(scrollers) {
		view, x, y := b.PaddingBox, b.ScrollX, b.ScrollY
		moved = t.scrollTo(b, x+reveal(target.X, target.W, view.X, view.W), y+reveal(target.Y, target.H, view.Y, view.H)) || moved
		target.X, target.Y = target.X-(b.ScrollX-x), target.Y-(b.ScrollY-y)
	}
	return moved
}

func reveal(at, size, view, span int) int {
	switch {
	case at < view:
		return at - view
	case at+size > view+span:
		return min(at+size-view-span, at-view)
	}
	return 0
}

func (t *Tree) scroller(path []int) *layout.Box {
	if s, _ := t.find(path); s != nil && s.box.Style.Overflow == layout.OverflowScroll {
		return s.box
	}
	return nil
}

func (t *Tree) find(path []int) (*styledBox, []*layout.Box) {
	s := t.root
	var scrollers []*layout.Box
	for _, i := range path {
		if s == nil || i < 0 || i >= len(s.children) {
			return nil, nil
		}
		if s.box.Style.Overflow == layout.OverflowScroll {
			scrollers = append(scrollers, s.box)
		}
		s = s.children[i]
	}
	return s, scrollers
}

func (t *Tree) scrollTo(b *layout.Box, x, y int) bool {
	x, y = max(min(x, b.ScrollWidth-b.PaddingBox.W), 0), max(min(y, b.ScrollHeight-b.PaddingBox.H), 0)
	if x == b.ScrollX && y == b.ScrollY {
		return false
	}
	b.ScrollX, b.ScrollY, t.relayout = x, y, true
	return true
}

func (s *styledBox) scene(moved bool) scene.Node {
	if s.painted && !moved {
		return s.node
	}
	n := scene.New(s.box, s.computed, s.text)
	n.Truncate = s.truncate
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
	var state style.NodeState
	if n.State != nil {
		state = *n.State
	}
	restate := s.state.States != state.States || !slices.Equal(s.state.Attrs, state.Attrs)
	crossed := t.crossed && f.Sheet.Responsive(n.Classes)
	if prev == nil || t.restyle || parentChanged || restate || crossed || !slices.Equal(s.classes, n.Classes) {
		t.cascades++
		computed := f.Sheet.ComputeState(parent, n.Classes, state)
		ls, err := boxStyle(parent, computed, f.Cell)
		if err != nil {
			return nil, err
		}
		if prev == nil || ls != s.box.Style {
			s.box.Style, t.relayout = ls, true
		}
		if reverse := reversed(computed); reverse != s.reverse {
			s.reverse, t.relayout = reverse, true
		}
		if changed = prev == nil || t.restyle || !reflect.DeepEqual(computed, s.computed); changed {
			s.computed, s.painted = computed, false
		}
		if nowrap := unwrapped(computed.WhiteSpace); nowrap != s.nowrap {
			s.nowrap, t.relayout = nowrap, true
			clear(s.sizes)
		}
		ellipsis := computed.TextOverflow == style.TextOverflowEllipsis || len(n.Classes) == 0 && parent.TextOverflow == style.TextOverflowEllipsis
		if truncate := ellipsis && s.nowrap; truncate != s.truncate {
			s.truncate, s.painted = truncate, false
		}
		s.classes, s.state = n.Classes, state
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
		at := i
		if s.reverse {
			at = len(n.Children) - 1 - i
		}
		s.children[i], s.box.Children[at] = child, child.box
		s.painted = s.painted && child.painted
	}
	return s, nil
}

func (s *styledBox) retext(f Frame, raw string) (moved bool) {
	next := styledBox{nowrap: s.nowrap}
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
		if availableWidth < size[0] && !s.nowrap {
			wrapAt := availableWidth
			if wrapAt == 0 {
				wrapAt = s.text.MinContent()
			}
			size[0], size[1] = s.text.Size(wrapAt)
		}
		s.sizes[availableWidth] = size
	}
	return size[0], size[1]
}

func boxStyle(parent, s style.ComputedStyle, cell image.Point) (layout.Style, error) {
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
		case style.Auto, style.None, style.FitContent:
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
	switch {
	case scrolls(s.OverflowX) || scrolls(s.OverflowY):
		out.Overflow = layout.OverflowScroll
	case s.OverflowX != style.OverflowVisible || s.OverflowY != style.OverflowVisible:
		out.Overflow = layout.OverflowHidden
	}
	switch s.Wrap {
	case style.NoWrap:
	case style.Wrap:
		out.Wrap = layout.Wrap
	case style.WrapReverse:
		out.Wrap = layout.WrapReverse
	default:
		panic(fmt.Sprintf("render: unknown flex-wrap %d", s.Wrap))
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
	if s.AspectRatio > 0 {
		precision := cell.X * cell.Y
		out.Aspect = layout.Ratio{W: int(math.Round(s.AspectRatio * float64(cell.Y*precision))), H: cell.X * precision}
	}
	column := parent.Display != style.DisplayFlex || parent.Direction == style.Column || parent.Direction == style.ColumnReverse
	stretched := s.AlignSelf == style.AlignStretch || s.AlignSelf == style.AlignAuto && (parent.AlignItems == style.AlignAuto || parent.AlignItems == style.AlignStretch)
	if stretched && (column && s.Width.Unit == style.FitContent || !column && s.Height.Unit == style.FitContent) {
		out.AlignSelf = layout.AlignStart
	}
	switch {
	case s.Display == style.DisplayNone:
		out.Display = layout.DisplayNone
	case s.Display == style.DisplayBlock:
		out.Direction = layout.Column
	case s.Display != style.DisplayFlex:
		unsupported = append(unsupported, "display")
	case s.Direction == style.Column || s.Direction == style.ColumnReverse:
		out.Direction = layout.Column
	}
	if reversed(s) {
		if s.Wrap != style.NoWrap {
			unsupported = append(unsupported, "flex-wrap in a reversed direction")
		}
		switch out.Justify {
		case layout.JustifyStart:
			out.Justify = layout.JustifyEnd
		case layout.JustifyEnd:
			out.Justify = layout.JustifyStart
		}
	}
	if len(unsupported) > 0 {
		return layout.Style{}, fmt.Errorf("twi: layout does not support this %s yet", strings.Join(slices.Compact(unsupported), ", "))
	}
	return out, nil
}

func reversed(s style.ComputedStyle) bool {
	return s.Display == style.DisplayFlex && (s.Direction == style.RowReverse || s.Direction == style.ColumnReverse)
}

func scrolls(o style.Overflow) bool {
	switch o {
	case style.OverflowVisible, style.OverflowHidden:
		return false
	case style.OverflowScroll, style.OverflowAuto:
		return true
	}
	panic(fmt.Sprintf("render: unknown overflow %d", o))
}

func unwrapped(w style.WhiteSpace) bool {
	switch w {
	case style.WhiteSpaceNormal, style.WhiteSpacePreWrap:
		return false
	case style.WhiteSpaceNowrap, style.WhiteSpacePre:
		return true
	}
	panic(fmt.Sprintf("render: unknown white-space %d", w))
}
