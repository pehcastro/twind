package render

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"

	lkonst "github.com/twind-dev/twind/internal/konst/layout"
	skonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/motion"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

type Node struct {
	Text     string
	Element  style.Element
	Classes  []string
	State    *style.NodeState
	TopLayer int
	Children []Node
	Enter    *motion.Presence
	Exit     *motion.Presence
	At       *image.Point
}

type Frame struct {
	Sheet         style.Sheet
	Width         int
	Height        layout.Length
	Sanitize      func(raw string) scene.Text
	Look          paint.Look
	Profile       color.Profile
	Cell          image.Point
	Widths        text.Widths
	Now           time.Duration
	ReducedMotion bool
	Graphics      bool
}

type styledBox struct {
	state    style.NodeState
	box      *layout.Box
	classes  []string
	computed style.ComputedStyle
	element  style.Element
	marks    style.Markers
	near     []int
	hands    []int
	related  []int
	truncate bool
	nowrap   bool
	wrapping text.Wrapping
	placed   bool
	at       image.Point
	reverse  bool
	top      int
	raw      string
	text     scene.Text
	natural  [2]int
	sizes    map[int][2]int
	children []*styledBox
	exiting  []*styledBox
	node     scene.Node
	painted  bool
	reclip   reclip
	key      motion.Key
	born     time.Duration
	enter    *motion.Presence
	exit     *motion.Presence
	animated bool
	shown    *style.ComputedStyle
	lift     motion.Offset
	pose     motion.Pose
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
	(&paint.Painter{Widths: f.Widths, Profile: f.Profile}).Paint(buf, &tree, f.Look)
	return buf, nil
}

type Tree struct {
	root             *styledBox
	cell             image.Point
	band             int
	restyle, crossed bool
	cascades         int
	ancestors        []*styledBox
	related          []int
	motion           motion.Styles
	overlay          motion.Animated
	keys             motion.Key
	free             []motion.Key
	now              time.Duration
	presenting       bool
	graphics         bool
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
	t.restyle = t.restyle || f.Cell != t.cell
	t.crossed = band != t.band
	t.ancestors = t.ancestors[:0]
	t.now, t.motion.Reduced, t.presenting, t.graphics = f.Now, f.ReducedMotion, false, f.Graphics
	styled, err := t.build(f, t.root, style.ComputedStyle{}, false, root, style.PlaceOf(0, 1), nil)
	t.root, t.cell, t.band, t.restyle = styled, f.Cell, band, false
	if err != nil {
		return scene.Node{}, err
	}
	layout.Layout(styled.box, f.Width, f.Height)
	return t.scene(styled, reclip{viewport: styled.box.Clip}), nil
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
	b.ScrollX, b.ScrollY = x, y
	b.Invalidate()
	return true
}

type reclip struct {
	on                       bool
	flow, absolute, viewport layout.Rect
}

func (t *Tree) scene(s *styledBox, r reclip) scene.Node {
	if s.painted && !s.box.Moved && r == s.reclip {
		return s.node
	}
	s.reclip, s.box.Moved = r, false
	st := &s.computed
	if s.animated {
		st = s.shown
	}
	n := scene.New(s.box, *st, s.text)
	n.Truncate, n.NoWrap, n.TopLayer = s.truncate, s.nowrap, s.top
	if s.top > 0 {
		r.on, r.flow, r.absolute = true, r.viewport, r.viewport
	}
	if r.on {
		n.Clip = r.flow
		if n.HidesOverflow {
			r.flow = overlap(r.flow, n.Padding)
		}
		if n.Position != layout.PositionStatic {
			r.absolute = r.flow
		}
	}
	n.Children = make([]scene.Node, 0, len(s.children)+len(s.exiting))
	for _, list := range [2][]*styledBox{s.children, s.exiting} {
		for _, c := range list {
			inner := r
			switch c.box.Style.Position {
			case layout.PositionStatic, layout.PositionRelative:
			case layout.PositionAbsolute:
				inner.flow = r.absolute
			case layout.PositionFixed:
				inner.flow, inner.absolute = r.viewport, r.viewport
			}
			if c.box.Style.Display == layout.DisplayNone {
				inner.on, inner.viewport = false, layout.Rect{}
			}
			n.Children = append(n.Children, t.scene(c, inner))
		}
	}
	n.Opacity *= s.lift.Opacity
	dx := int(math.Round(translated(st.TranslateX, n.Bounds.W) + translated(s.pose.TranslateX, n.Bounds.W) + s.lift.X))
	dy := int(math.Round(translated(st.TranslateY, n.Bounds.H) + translated(s.pose.TranslateY, n.Bounds.H) + s.lift.Y))
	sx, sy := st.ScaleX*s.lift.Scale*s.pose.Scale, st.ScaleY*s.lift.Scale*s.pose.Scale
	if dx != 0 || dy != 0 || sx != 1 || sy != 1 {
		transform(&n, dx, dy, sx, sy)
	}
	n.Turn = s.pose.Turn
	if k := s.computed.Animation.Keyframes; k != style.KeyframesNone && k != style.KeyframesExit {
		t.motion.Unseen(s.key, n.Visibility == style.Hidden || n.Bounds.W <= 0 || n.Bounds.H <= 0 || k == style.KeyframesSpin && !t.graphics)
	}
	s.node, s.painted = n, true
	return n
}

func overlap(a, b layout.Rect) layout.Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return layout.Rect{X: x, Y: y, W: max(min(a.X+a.W, b.X+b.W)-x, 0), H: max(min(a.Y+a.H, b.Y+b.H)-y, 0)}
}

func (t *Tree) build(f Frame, prev *styledBox, parent style.ComputedStyle, parentChanged bool, n Node, place style.Place, siblings []*styledBox) (*styledBox, error) {
	s := prev
	if s == nil {
		s = &styledBox{box: &layout.Box{}, key: t.key(), born: t.now, lift: motion.Still(), pose: motion.Pose{Scale: 1}}
	}
	s.enter, s.exit = n.Enter, n.Exit
	changed, animating := false, false
	var state style.NodeState
	if n.State != nil {
		state = *n.State
	}
	state.Places = place
	reclassed := prev == nil || !slices.Equal(s.classes, n.Classes)
	if reclassed {
		s.marks, s.near = f.Sheet.Marks(n.Classes), f.Sheet.Near(n.Classes, s.near[:0])
	}
	placed, at := n.At != nil, image.Point{}
	if placed {
		at = *n.At
	}
	related := t.relate(f.Sheet, s, n, state, siblings)
	restate := s.state.States != state.States || s.state.Places != state.Places || !slices.Equal(s.state.Attrs, state.Attrs) || n.Element != s.element || !slices.Equal(related, s.related) || placed != s.placed || at != s.at
	crossed := t.crossed && f.Sheet.Responsive(n.Classes)
	if reclassed || t.restyle || parentChanged || restate || crossed {
		t.cascades++
		computed := f.Sheet.ComputeRelated(parent, n.Classes, state, related)
		if k := computed.Animation.Keyframes; k == style.KeyframesEnter && n.Enter != nil || k == style.KeyframesExit && n.Exit != nil {
			computed.Animation = style.Animation{}
		}
		if placed {
			if computed.Position != style.PositionFixed {
				computed.Position = style.PositionAbsolute
			}
			auto := style.Length{Unit: style.Auto}
			computed.Inset = style.Edges{Top: style.Length{Value: float64(at.Y)}, Left: style.Length{Value: float64(at.X)}, Right: auto, Bottom: auto}
		}
		s.related = append(s.related[:0], related...)
		s.hands = f.Sheet.Hands(n.Classes, state, s.hands[:0])
		ls, err := boxStyle(parent, computed, f.Cell)
		if err != nil {
			return nil, err
		}
		if prev == nil || !reflect.DeepEqual(ls, s.box.Style) {
			s.box.Style = ls
			s.box.Invalidate()
		}
		if reverse := reversed(computed); reverse != s.reverse {
			s.reverse = reverse
			s.box.Invalidate()
		}
		if animating = moves(&computed) || s.animated || t.motion.Holds(s.key); animating {
			before := &s.computed
			if prev == nil {
				before = nil
			}
			t.animate(s, before, &computed)
		}
		if changed = prev == nil || t.restyle || !reflect.DeepEqual(computed, s.computed); changed {
			s.computed, s.painted = computed, false
		}
		nowrap := unwrapped(computed.WhiteSpace)
		wrapping := scene.Wrapping(&computed)
		wrapping.Widths = s.wrapping.Widths
		if nowrap != s.nowrap || wrapping != s.wrapping {
			s.nowrap, s.wrapping = nowrap, wrapping
			clear(s.sizes)
			s.box.Invalidate()
		}
		ellipsis := computed.TextOverflow == style.TextOverflowEllipsis || len(n.Classes) == 0 && parent.TextOverflow == style.TextOverflowEllipsis
		if truncate := ellipsis && s.nowrap; truncate != s.truncate {
			s.truncate, s.painted = truncate, false
		}
		s.classes, s.state, s.element, s.placed, s.at = n.Classes, state, n.Element, placed, at
	}
	if !animating && (s.animated || t.motion.Holds(s.key)) {
		t.animate(s, &s.computed, &s.computed)
	}
	lift := motion.Still()
	if age := t.now - s.born; s.enter != nil && age < s.enter.Duration && !t.motion.Reduced {
		lift, t.presenting = s.enter.Enter(age), true
	}
	if lift != s.lift {
		s.lift, s.painted = lift, false
	}
	if n.TopLayer != s.top {
		s.top, s.painted = n.TopLayer, false
	}
	if (n.Text != s.raw || n.Text != "" && f.Widths != s.wrapping.Widths) && s.retext(f, n.Text) {
		s.box.Invalidate()
	}
	old := s.children
	if len(old) != len(n.Children) {
		for _, gone := range old[min(len(old), len(n.Children)):] {
			gone.born = t.now
			s.exiting = append(s.exiting, gone)
		}
		s.children, s.box.Children = make([]*styledBox, len(n.Children)), make([]*layout.Box, len(n.Children))
		s.painted = false
		s.box.Invalidate()
	}
	if len(s.exiting) > 0 {
		t.exits(s)
	}
	t.ancestors = append(t.ancestors, s)
	last, index := lastElement(n.Children), 0
	for i, c := range n.Children {
		var p *styledBox
		if i < len(old) {
			p = old[i]
		}
		var place style.Place
		if !c.text() {
			place, index = placeOf(index, i == last), index+1
		}
		child, err := t.build(f, p, s.computed, changed, c, place, s.children[:i])
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
	s.box.Children = s.box.Children[:len(n.Children)]
	for _, e := range s.exiting {
		s.box.Children = append(s.box.Children, e.box)
	}
	t.ancestors = t.ancestors[:len(t.ancestors)-1]
	return s, nil
}

func (n Node) text() bool {
	return n.Text != "" && n.Element == style.ElementAny && n.Classes == nil && n.State == nil && n.Children == nil
}

func (t *Tree) relate(sheet style.Sheet, s *styledBox, n Node, state style.NodeState, siblings []*styledBox) []int {
	related := t.related[:0]
	if n.text() {
		return related
	}
	for _, i := range s.near {
		if near := &sheet.Rule(i).Near; t.near(sheet, near, n, siblings) {
			related = append(related, i)
		}
	}
	for depth, a := range t.ancestors {
		for _, i := range a.hands {
			target := &sheet.Rule(i).Target
			if (target.Relation == style.RelationDescendant || depth == len(t.ancestors)-1) && target.Accepts(n.Element, s.marks, state) {
				related = append(related, i)
			}
		}
	}
	t.related = related
	return related
}

func (t *Tree) near(sheet style.Sheet, m *style.Match, n Node, siblings []*styledBox) bool {
	switch m.Relation {
	case style.RelationAncestor:
		return slices.ContainsFunc(t.ancestors, func(a *styledBox) bool { return m.Accepts(a.element, a.marks, a.state) })
	case style.RelationPrevious:
		return slices.ContainsFunc(siblings, func(b *styledBox) bool { return m.Accepts(b.element, b.marks, b.state) })
	case style.RelationChild, style.RelationDescendant:
		return has(sheet, m, n.Children)
	}
	panic(fmt.Sprintf("render: unknown relation %d", m.Relation))
}

func lastElement(children []Node) int {
	i := len(children) - 1
	for i >= 0 && children[i].text() {
		i--
	}
	return i
}

func placeOf(index int, last bool) style.Place {
	if last {
		return style.PlaceOf(index, index+1)
	}
	return style.PlaceOf(index, index+2)
}

func has(sheet style.Sheet, m *style.Match, children []Node) bool {
	last, index := lastElement(children), 0
	for i, c := range children {
		if c.text() {
			continue
		}
		var state style.NodeState
		if c.State != nil {
			state = *c.State
		}
		state.Places, index = placeOf(index, i == last), index+1
		var marks style.Markers
		if m.Class != "" {
			marks = sheet.Marks(c.Classes)
		}
		if m.Accepts(c.Element, marks, state) || m.Relation == style.RelationDescendant && has(sheet, m, c.Children) {
			return true
		}
	}
	return false
}

func (s *styledBox) retext(f Frame, raw string) (moved bool) {
	next := styledBox{nowrap: s.nowrap, wrapping: s.wrapping}
	next.wrapping.Widths = f.Widths
	if raw != "" {
		next.text = f.Sanitize(raw)
	}
	moved = (s.text == scene.Text{}) != (next.text == scene.Text{})
	if next.text != (scene.Text{}) {
		next.natural[0], next.natural[1] = next.text.Size(next.wrapping, math.MaxInt)
		next.sizes = map[int][2]int{}
		for width, size := range s.sizes {
			w, h := next.measure(width)
			moved = moved || size != [2]int{w, h}
		}
	}
	s.raw, s.text, s.natural, s.sizes, s.wrapping, s.painted = raw, next.text, next.natural, next.sizes, next.wrapping, false
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
			if wrapAt <= 0 {
				wrapAt = s.text.MinContent(s.wrapping)
			}
			size[0], size[1] = s.text.Size(s.wrapping, wrapAt)
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
	justify := func(j style.Justify) layout.Justify {
		return [...]layout.Justify{
			style.JustifyStart:   layout.JustifyStart,
			style.JustifyEnd:     layout.JustifyEnd,
			style.JustifyCenter:  layout.JustifyCenter,
			style.JustifyBetween: layout.JustifyBetween,
			style.JustifyAround:  layout.JustifyAround,
			style.JustifyEvenly:  layout.JustifyEvenly,
			style.JustifyStretch: layout.JustifyStretch,
		}[j]
	}
	placement := func(p style.GridPlacement) layout.Placement {
		return layout.Placement{Start: layout.Line{Index: p.Start.Line, Span: p.Start.Span}, End: layout.Line{Index: p.End.Line, Span: p.End.Span}}
	}
	out := layout.Style{
		Justify:      justify(s.Justify),
		AlignContent: justify(s.AlignContent),
		AlignItems:   align("align-items", s.AlignItems),
		AlignSelf:    align("align-self", s.AlignSelf),
		JustifyItems: align("justify-items", s.JustifyItems),
		JustifySelf:  align("justify-self", s.JustifySelf),
		Columns:      tracks(s.GridColumns),
		Rows:         tracks(s.GridRows),
		AutoColumns:  tracks(s.GridAutoColumns),
		AutoRows:     tracks(s.GridAutoRows),
		Column:       placement(s.GridColumn),
		Row:          placement(s.GridRow),
		Flow: [...]layout.Flow{
			style.FlowRow:         layout.FlowRow,
			style.FlowColumn:      layout.FlowColumn,
			style.FlowRowDense:    layout.FlowRowDense,
			style.FlowColumnDense: layout.FlowColumnDense,
		}[s.GridFlow],
		Grow:      int(math.Round(s.Grow)),
		Shrink:    int(math.Round(s.Shrink)),
		Basis:     length(s.Basis),
		Width:     length(s.Width),
		Height:    length(s.Height),
		MinWidth:  length(s.MinWidth),
		MinHeight: length(s.MinHeight),
		MaxWidth:  length(s.MaxWidth),
		MaxHeight: length(s.MaxHeight),
		RowGap:    cells("row-gap", s.RowGap),
		ColumnGap: cells("column-gap", s.ColumnGap),
		Padding:   edges("padding", s.Padding),
		Margin:    edges("margin", s.Margin),
		Border:    edges("border-width", s.BorderWidth),
		Inset:     layout.Insets{Top: length(s.Inset.Top), Right: length(s.Inset.Right), Bottom: length(s.Inset.Bottom), Left: length(s.Inset.Left)},
		ZIndex:    s.ZIndex,
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
	stretches := func(self, items style.Align) bool {
		return self == style.AlignStretch || self == style.AlignAuto && (items == style.AlignAuto || items == style.AlignStretch)
	}
	stretched := stretches(s.AlignSelf, parent.AlignItems)
	column := parent.Display != style.DisplayFlex || parent.Direction == style.Column || parent.Direction == style.ColumnReverse
	switch {
	case parent.Display == style.DisplayGrid:
		if stretches(s.JustifySelf, parent.JustifyItems) && s.Width.Unit == style.FitContent {
			out.JustifySelf = layout.AlignStart
		}
		if stretched && s.Height.Unit == style.FitContent {
			out.AlignSelf = layout.AlignStart
		}
	case stretched && (column && s.Width.Unit == style.FitContent || !column && s.Height.Unit == style.FitContent):
		out.AlignSelf = layout.AlignStart
	}
	switch {
	case s.Display == style.DisplayNone:
		out.Display = layout.DisplayNone
	case s.Display == style.DisplayGrid:
		out.Display = layout.DisplayGrid
	case s.Display == style.DisplayBlock:
		out.Direction = layout.Column
	case s.Display != style.DisplayFlex:
		unsupported = append(unsupported, "display")
	case s.Direction == style.Column || s.Direction == style.ColumnReverse:
		out.Direction = layout.Column
	}
	if s.Display == style.DisplayFlex && s.Wrap != style.NoWrap && s.AlignContent != style.JustifyStretch {
		unsupported = append(unsupported, "align-content in a wrapping flex container")
	}
	if reversed(s) {
		if s.Wrap != style.NoWrap {
			unsupported = append(unsupported, "flex-wrap in a reversed direction")
		}
		switch out.Justify {
		case layout.JustifyStart, layout.JustifyStretch:
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

func tracks(ts []style.Track) []layout.Track {
	if ts == nil {
		return nil
	}
	breadth := func(b style.Breadth) layout.Breadth {
		switch b.Kind {
		case style.SizeAuto:
			return layout.Breadth{}
		case style.SizeCells:
			return layout.Breadth{Kind: layout.SizeCells, Value: int(math.Round(b.Value))}
		case style.SizePercent:
			return layout.Breadth{Kind: layout.SizePercent, Value: int(math.Round(b.Value))}
		case style.SizeFr:
			return layout.Breadth{Kind: layout.SizeFr, Value: int(math.Round(b.Value * lkonst.FrUnit))}
		case style.SizeMinContent:
			return layout.Breadth{Kind: layout.SizeMinContent}
		case style.SizeMaxContent:
			return layout.Breadth{Kind: layout.SizeMaxContent}
		}
		panic(fmt.Sprintf("render: unknown track size %d", b.Kind))
	}
	out := make([]layout.Track, len(ts))
	for i, t := range ts {
		out[i] = layout.Track{Min: breadth(t.Min), Max: breadth(t.Max)}
	}
	return out
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
	case style.WhiteSpaceNormal, style.WhiteSpacePreWrap, style.WhiteSpacePreLine:
		return false
	case style.WhiteSpaceNowrap, style.WhiteSpacePre:
		return true
	}
	panic(fmt.Sprintf("render: unknown white-space %d", w))
}
