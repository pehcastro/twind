package render

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"image"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/pehcastro/twind/internal/buffer"
	lkonst "github.com/pehcastro/twind/internal/konst/layout"
	skonst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/paint"
	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/motion"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/text"
)

type Node struct {
	Key      string
	Text     string
	Element  style.Element
	Classes  []string
	State    *style.NodeState
	TopLayer int
	Children []Node
	Enter    *motion.Presence
	Exit     *motion.Presence
	Extra    *Extra
}

type Extra struct {
	Placement
	Canvas *Canvas
}

type Placement struct {
	Positioned   bool
	At, Min, Max image.Point
}

type Canvas struct {
	Key   uint64
	Paint func(dst *image.RGBA, cell image.Point)
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
	box      layout.Box
	computed *style.ComputedStyle
	node     *scene.Node
	state    style.NodeState
	classes  []string
	element  style.Element
	truncate bool
	nowrap   bool
	reverse  bool
	painted  bool
	animated bool
	marks    style.Markers
	near     []int
	hands    []int
	related  []int
	wrapping text.Wrapping
	placed   Placement
	top      int
	raw      string
	text     scene.Text
	natural  [2]int
	rows     int
	sizes    []sized
	children []*styledBox
	exiting  []*styledBox
	reclip   reclip
	key      motion.Key
	id       string
	born     time.Duration
	enter    *motion.Presence
	exit     *motion.Presence
	move     *moving
	canvas   *Canvas
	pixels   *raster.Pixels
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
	rows             int
	drawn            scene.Node
	blank            style.ComputedStyle
	seed             maphash.Seed
	styles           map[cascade]*styledBox
}

type cascade struct {
	inherited           style.Inheritance
	display             style.Display
	direction           style.Direction
	items, justify      style.Align
	dir                 text.Direction
	hash                uint64
	states              style.State
	places              style.Place
	enter, exit, placed bool
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
	if t.styles == nil {
		t.seed, t.styles = maphash.MakeSeed(), map[cascade]*styledBox{}
	}
	rows := 1
	if f.Graphics {
		rows = lkonst.HalfRows
	}
	if rows != t.rows && t.root != nil {
		rescale(t.root, rows)
		t.restyle = true
	}
	t.rows = rows
	styled, fresh := t.root, t.root == nil
	if fresh {
		styled = new(styledBox)
	}
	err := t.build(&f, styled, fresh, &t.blank, text.DirLTR, false, root, style.PlaceOf(0, 1), nil)
	clear(t.styles)
	t.root, t.cell, t.band, t.restyle = styled, f.Cell, band, false
	if err != nil {
		t.root = nil
		return scene.Node{}, err
	}
	height := f.Height
	if height.Unit == layout.Cells {
		height.Value *= rows
	}
	layout.Layout(&styled.box, f.Width, height)
	viewport, _ := t.outer(styled.box.Clip)
	t.scene(styled, reclip{viewport: viewport}, &t.drawn)
	styled.node = &t.drawn
	return t.drawn, nil
}

func (t *Tree) cascade(f *Frame, s *styledBox, parent *style.ComputedStyle, dir text.Direction, n *Node, state *style.NodeState, related []int, placed Placement) (*style.ComputedStyle, layout.Style, error) {
	var h maphash.Hash
	h.SetSeed(t.seed)
	for _, c := range n.Classes {
		h.WriteString(c)
		h.WriteByte(0)
	}
	for _, a := range state.Attrs {
		h.WriteString(a.Name)
		h.WriteByte(0)
		h.WriteString(a.Value)
		h.WriteByte(0)
	}
	var word [8]byte
	put := func(v int) {
		binary.LittleEndian.PutUint64(word[:], uint64(v))
		h.Write(word[:])
	}
	for _, r := range related {
		put(r)
	}
	for _, v := range [...]int{placed.At.X, placed.At.Y, placed.Min.X, placed.Min.Y, placed.Max.X, placed.Max.Y} {
		put(v)
	}
	k := cascade{parent.Inheritance(), parent.Display, parent.Direction, parent.AlignItems, parent.JustifyItems, dir, h.Sum64(), state.States, state.Places, s.enter != nil, s.exit != nil, placed.Positioned}
	if m := t.styles[k]; m != nil && m.placed == placed && slices.Equal(m.classes, n.Classes) && slices.Equal(m.state.Attrs, state.Attrs) && slices.Equal(m.related, related) {
		return m.computed, m.box.Style, nil
	}
	t.styles[k] = s
	computed := new(style.ComputedStyle)
	*computed = f.Sheet.ComputeRelated(*parent, n.Classes, *state, related)
	if k := computed.Animation.Keyframes; k == style.KeyframesEnter && s.enter != nil || k == style.KeyframesExit && s.exit != nil {
		computed.Animation = style.Animation{}
	}
	if at := placed.At; placed.Positioned {
		if computed.Position != style.PositionFixed {
			computed.Position = style.PositionAbsolute
		}
		auto := style.Length{Unit: style.Auto}
		computed.Inset = style.Edges{Top: style.Length{Value: float64(at.Y)}, Left: style.Length{Value: float64(at.X)}, Right: auto, Bottom: auto}
	}
	limit := func(cells int, to *style.Length) {
		if cells > 0 {
			*to = style.Length{Value: float64(cells)}
		}
	}
	limit(placed.Min.X, &computed.MinWidth)
	limit(placed.Min.Y, &computed.MinHeight)
	limit(placed.Max.X, &computed.MaxWidth)
	limit(placed.Max.Y, &computed.MaxHeight)
	ls, err := boxStyle(parent, computed, dir, f.Cell, t.rows)
	return computed, ls, err
}

func (t *Tree) ScrollBy(path []int, dx, dy int) bool {
	b := t.scroller(path)
	return b != nil && t.scrollTo(b, b.ScrollX+dx, b.ScrollY+dy*t.rows)
}

func (t *Tree) ScrollTo(path []int, x, y int) bool {
	b := t.scroller(path)
	return b != nil && t.scrollTo(b, x, y*t.rows)
}

func (t *Tree) ScrollIntoView(path []int) bool { return t.intoView(path, false) }

func (t *Tree) ScrollToAnchor(path []int) bool { return t.intoView(path, true) }

func (t *Tree) intoView(path []int, anchor bool) bool {
	s, scrollers := t.find(path)
	if s == nil {
		return false
	}
	target, moved := s.box.BorderBox, false
	for i, b := range slices.Backward(scrollers) {
		view, x, y := b.PaddingBox, b.ScrollX, b.ScrollY
		dy := reveal(target.Y, target.H, view.Y, view.H)
		if anchor && i == len(scrollers)-1 {
			dy = down(target.Y-view.Y, t.rows) * t.rows
		}
		moved = t.scrollTo(b, x+reveal(target.X, target.W, view.X, view.W), y+dy) || moved
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
		return &s.box
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
			scrollers = append(scrollers, &s.box)
		}
		s = s.children[i]
	}
	return s, scrollers
}

func (t *Tree) scrollTo(b *layout.Box, x, y int) bool {
	x, y = max(min(x, b.ScrollWidth-b.PaddingBox.W), 0), max(min(y, up(b.ScrollHeight-b.PaddingBox.H, t.rows)*t.rows), 0)
	if y < b.ScrollY {
		y = down(y, t.rows) * t.rows
	} else {
		y = up(y, t.rows) * t.rows
	}
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

func (t *Tree) scene(s *styledBox, r reclip, n *scene.Node) {
	if s.painted && !s.box.Moved && r == s.reclip {
		*n = *s.node
		return
	}
	s.reclip, s.box.Moved = r, false
	st, m := s.computed, s.current()
	if s.animated {
		st = m.shown
	}
	*n = scene.New(&s.box, *st, s.text)
	n.Direct(s.wrapping.Dir)
	n.Truncate, n.NoWrap, n.TopLayer = s.truncate, s.nowrap, s.top
	n.Bounds, n.Halves.Bounds = t.outer(n.Bounds)
	n.Padding, n.Halves.Padding = t.outer(n.Padding)
	n.Clip, n.Halves.Clip = t.outer(n.Clip)
	row := up(n.Content.Y, t.rows)
	n.Content.Y, n.Content.H = row, up(n.Content.Y+n.Content.H, t.rows)-row
	n.ScrollContent, _ = t.outer(n.ScrollContent)
	if s.top > 0 {
		r.on, r.flow, r.absolute = true, r.viewport, r.viewport
	}
	if r.on {
		n.Clip, n.Halves.Clip = r.flow, 0
		if n.HidesOverflow {
			r.flow = overlap(r.flow, n.Padding)
		}
		if n.Position != layout.PositionStatic {
			r.absolute = r.flow
		}
	}
	dx := int(math.Round(translated(st.TranslateX, n.Bounds.W) + translated(m.pose.TranslateX, n.Bounds.W) + m.lift.X))
	dy := int(math.Round(translated(st.TranslateY, n.Bounds.H) + translated(m.pose.TranslateY, n.Bounds.H) + m.lift.Y))
	sx, sy := st.ScaleX, st.ScaleY
	transformed := dx != 0 || dy != 0 || sx != 1 || sy != 1
	if transformed {
		r.on, r.flow = true, layout.Rect{X: -lkonst.Unbounded / 2, Y: -lkonst.Unbounded / 2, W: lkonst.Unbounded, H: lkonst.Unbounded}
		if n.HidesOverflow {
			r.flow = n.Padding
		}
		r.absolute = r.flow
	}
	if size := image.Pt(n.Content.W*t.cell.X, n.Content.H*t.cell.Y); s.canvas != nil && t.graphics && size.X > 0 && size.Y > 0 {
		if s.pixels == nil || s.pixels.Key != s.canvas.Key || s.pixels.Image.Rect.Size() != size {
			s.pixels = &raster.Pixels{Key: s.canvas.Key, Image: image.NewRGBA(image.Rectangle{Max: size})}
			s.canvas.Paint(s.pixels.Image, t.cell)
		}
		n.Pixels = s.pixels
	}
	if n.Pixels == nil {
		n.Children = make([]scene.Node, len(s.children)+len(s.exiting))
		for i := range n.Children {
			var c *styledBox
			if i < len(s.children) {
				c = s.children[i]
			} else {
				c = s.exiting[i-len(s.children)]
			}
			inner := r
			switch c.box.Style.Position {
			case layout.PositionStatic, layout.PositionRelative, layout.PositionSticky:
			case layout.PositionAbsolute:
				inner.flow = r.absolute
			case layout.PositionFixed:
				inner.flow, inner.absolute = r.viewport, r.viewport
			}
			if c.box.Style.Display == layout.DisplayNone {
				inner.on, inner.viewport = false, layout.Rect{}
			}
			t.scene(c, inner, &n.Children[i])
			c.node = &n.Children[i]
		}
	}
	n.Opacity *= m.lift.Opacity
	if transformed {
		transform(n, dx, dy, sx, sy)
	}
	n.Turn, n.Shrink = m.pose.Turn, 1-m.lift.Scale*m.pose.Scale
	n.Enclose()
	if k := s.computed.Animation.Keyframes; k != style.KeyframesNone && k != style.KeyframesExit {
		t.motion.Unseen(s.key, n.Visibility == style.Hidden || n.Bounds.W <= 0 || n.Bounds.H <= 0 || k == style.KeyframesSpin && !t.graphics)
	}
	s.painted = true
}

func rescale(s *styledBox, rows int) {
	s.box.ScrollY = s.box.ScrollY / s.rows * rows
	s.rows, s.sizes, s.painted = rows, s.sizes[:0], false
	s.box.Invalidate()
	for _, list := range [2][]*styledBox{s.children, s.exiting} {
		for _, c := range list {
			rescale(c, rows)
		}
	}
}

func (t *Tree) outer(r layout.Rect) (layout.Rect, scene.Half) {
	top, bottom := down(r.Y, t.rows), up(r.Y+r.H, t.rows)
	var h scene.Half
	if top*t.rows != r.Y {
		h |= scene.HalfTop
	}
	if bottom*t.rows != r.Y+r.H {
		h |= scene.HalfBottom
	}
	return layout.Rect{X: r.X, Y: top, W: r.W, H: bottom - top}, h
}

func down(v, n int) int {
	q := v / n
	if v%n < 0 {
		q--
	}
	return q
}

func up(v, n int) int { return -down(-v, n) }

func overlap(a, b layout.Rect) layout.Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return layout.Rect{X: x, Y: y, W: max(min(a.X+a.W, b.X+b.W)-x, 0), H: max(min(a.Y+a.H, b.Y+b.H)-y, 0)}
}

func Dir(d text.Direction) style.Attr {
	switch d {
	case text.DirLTR:
		return style.Attr{Name: "dir", Value: "ltr"}
	case text.DirRTL:
		return style.Attr{Name: "dir", Value: "rtl"}
	}
	panic(fmt.Sprintf("render: no dir attribute for direction %d", d))
}

func (t *Tree) build(f *Frame, s *styledBox, fresh bool, parent *style.ComputedStyle, dir text.Direction, parentChanged bool, n Node, place style.Place, siblings []*styledBox) error {
	if fresh {
		s.key, s.born, s.rows = t.key(), t.now, t.rows
	}
	s.enter, s.exit = n.Enter, n.Exit
	changed, animating := false, false
	var state style.NodeState
	if n.State != nil {
		state = *n.State
	}
	state.Places = place
	for _, a := range state.Attrs {
		switch a {
		case Dir(text.DirLTR):
			dir = text.DirLTR
		case Dir(text.DirRTL):
			dir = text.DirRTL
		}
	}
	reclassed := fresh || !slices.Equal(s.classes, n.Classes)
	if reclassed {
		s.marks, s.near = f.Sheet.MarksNear(n.Classes, s.near[:0])
	}
	s.classes = n.Classes
	var placed Placement
	var painter *Canvas
	if n.Extra != nil {
		placed, painter = n.Extra.Placement, n.Extra.Canvas
	}
	related := t.relate(f.Sheet, s, n, state, siblings)
	restate := s.state.States != state.States || s.state.Places != state.Places || !slices.Equal(s.state.Attrs, state.Attrs) || n.Element != s.element || !slices.Equal(related, s.related) || placed != s.placed || dir != s.wrapping.Dir
	if reclassed || t.restyle || parentChanged || restate || t.crossed && f.Sheet.Responsive(n.Classes) {
		t.cascades++
		computed, ls, err := t.cascade(f, s, parent, dir, &n, &state, related, placed)
		if err != nil {
			return err
		}
		if fresh || !ls.Equal(&s.box.Style) {
			s.box.Style = ls
			s.box.Invalidate()
		}
		s.related = append(s.related[:0], related...)
		s.hands = f.Sheet.Hands(n.Classes, state, s.hands[:0])
		if reverse := reversed(computed) != mirrored(computed, dir); reverse != s.reverse {
			s.reverse = reverse
			s.box.Invalidate()
		}
		if animating = moves(computed) || s.animated || t.motion.Holds(s.key); animating {
			t.animate(s, s.computed, computed)
		}
		if changed = fresh || t.restyle || !computed.Equal(s.computed) || dir != s.wrapping.Dir; changed {
			s.painted = false
		}
		s.computed = computed
		nowrap := unwrapped(computed.WhiteSpace)
		wrapping := scene.Wrapping(computed)
		wrapping.Widths, wrapping.Dir = s.wrapping.Widths, dir
		if nowrap != s.nowrap || wrapping != s.wrapping {
			s.nowrap, s.wrapping = nowrap, wrapping
			s.sizes = s.sizes[:0]
			s.box.Invalidate()
		}
		ellipsis := computed.TextOverflow == style.TextOverflowEllipsis || len(n.Classes) == 0 && parent.TextOverflow == style.TextOverflowEllipsis
		if truncate := ellipsis && s.nowrap; truncate != s.truncate {
			s.truncate, s.painted = truncate, false
		}
		s.state, s.element, s.placed = state, n.Element, placed
	}
	if !animating && (s.animated || t.motion.Holds(s.key)) {
		t.animate(s, s.computed, s.computed)
	}
	lift := motion.Still()
	if age := t.now - s.born; s.enter != nil && age < s.enter.Duration && !t.motion.Reduced {
		lift, t.presenting = s.enter.Enter(age), true
	}
	if lift != s.current().lift {
		s.moves().lift, s.painted = lift, false
	}
	if n.TopLayer != s.top {
		s.top, s.painted = n.TopLayer, false
	}
	if (painter == nil) != (s.canvas == nil) || painter != nil && painter.Key != s.canvas.Key {
		s.painted = false
	}
	s.canvas = painter
	if (n.Text != s.raw || n.Text != "" && f.Widths != s.wrapping.Widths) && s.retext(f, n.Text) {
		s.box.Invalidate()
	}
	old := s.children
	kept := len(old) == len(n.Children)
	for i := 0; kept && i < len(old); i++ {
		kept = old[i].id == n.Children[i].Key
	}
	if !kept {
		s.children, s.box.Children = make([]*styledBox, len(n.Children)), make([]*layout.Box, len(n.Children))
		taken, born := make([]bool, len(old)), 0
		for i, c := range n.Children {
			at := i
			if c.Key != "" {
				at = slices.IndexFunc(old, func(o *styledBox) bool { return o.id == c.Key })
			}
			if at >= 0 && at < len(old) && !taken[at] && old[at].id == c.Key {
				s.children[i], taken[at] = old[at], true
			} else {
				born++
			}
		}
		slab := make([]styledBox, born)
		for i, c := range s.children {
			if c == nil {
				slab[0].id = n.Children[i].Key
				s.children[i], slab = &slab[0], slab[1:]
			}
		}
		for i, gone := range old {
			if !taken[i] {
				gone.born = t.now
				s.exiting = append(s.exiting, gone)
			}
		}
		s.painted = false
		s.box.Invalidate()
	}
	if len(s.exiting) > 0 {
		t.exits(s)
	}
	t.ancestors = append(t.ancestors, s)
	last, index := lastElement(n.Children), 0
	for i, c := range n.Children {
		var place style.Place
		if !c.text() {
			place, index = placeOf(index, i == last), index+1
		}
		child := s.children[i]
		if err := t.build(f, child, child.computed == nil, s.computed, dir, changed, c, place, s.children[:i]); err != nil {
			return err
		}
		at := i
		if s.reverse {
			at = len(n.Children) - 1 - i
		}
		s.box.Children[at] = &child.box
		s.painted = s.painted && (child.painted || s.canvas != nil && t.graphics)
	}
	s.box.Children = s.box.Children[:len(n.Children)]
	for _, e := range s.exiting {
		s.box.Children = append(s.box.Children, &e.box)
	}
	t.ancestors = t.ancestors[:len(t.ancestors)-1]
	return nil
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

func (s *styledBox) retext(f *Frame, raw string) (moved bool) {
	var next scene.Text
	if raw != "" {
		next = f.Sanitize(raw)
	}
	moved = (s.text == scene.Text{}) != (next == scene.Text{})
	stale := s.sizes
	s.raw, s.text, s.natural, s.sizes, s.wrapping.Widths, s.painted = raw, next, [2]int{}, s.sizes[:0], f.Widths, false
	if next == (scene.Text{}) {
		s.box.Measure = nil
		return moved
	}
	s.natural[0], s.natural[1] = next.Size(s.wrapping, math.MaxInt)
	for _, old := range stale {
		w, h := s.measure(old.width)
		moved = moved || old.size != [2]int{w, h}
	}
	if s.box.Measure == nil {
		s.box.Measure = s.measure
	}
	return moved
}

type sized struct {
	width int
	size  [2]int
}

func (s *styledBox) measure(availableWidth int) (int, int) {
	for _, m := range s.sizes {
		if m.width == availableWidth {
			return m.size[0], m.size[1]
		}
	}
	size := s.natural
	if availableWidth < size[0] && !s.nowrap {
		wrapAt := availableWidth
		if wrapAt <= 0 {
			wrapAt = s.text.MinContent(s.wrapping)
		}
		size[0], size[1] = s.text.Size(s.wrapping, wrapAt)
	}
	size[1] *= s.rows
	s.sizes = append(s.sizes, sized{availableWidth, size})
	return size[0], size[1]
}

func boxStyle(parent, s *style.ComputedStyle, dir text.Direction, cell image.Point, rows int) (layout.Style, error) {
	var unsupported []string
	cells := func(name string, l style.Length, unit int) int {
		if l.Unit != style.Cells {
			unsupported = append(unsupported, name)
		}
		return int(math.Round(l.Value * float64(unit)))
	}
	edges := func(name string, e style.Edges) layout.Edges {
		return layout.Edges{Top: cells(name, e.Top, rows), Right: cells(name, e.Right, 1), Bottom: cells(name, e.Bottom, rows), Left: cells(name, e.Left, 1)}
	}
	length := func(l style.Length, unit int) layout.Length {
		switch l.Unit {
		case style.Cells:
			return layout.Length{Unit: layout.Cells, Value: int(math.Round(l.Value * float64(unit)))}
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
	column := parent.Display != style.DisplayFlex || parent.Direction == style.Column || parent.Direction == style.ColumnReverse
	along := 1
	if column {
		along = rows
	}
	out := layout.Style{
		Justify:      justify(s.Justify),
		AlignContent: justify(s.AlignContent),
		AlignItems:   align("align-items", s.AlignItems),
		AlignSelf:    align("align-self", s.AlignSelf),
		JustifyItems: align("justify-items", s.JustifyItems),
		JustifySelf:  align("justify-self", s.JustifySelf),
		Columns:      tracks(s.GridColumns, 1),
		Rows:         tracks(s.GridRows, rows),
		AutoColumns:  tracks(s.GridAutoColumns, 1),
		AutoRows:     tracks(s.GridAutoRows, rows),
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
		Basis:     length(s.Basis, along),
		Width:     length(s.Width, 1),
		Height:    length(s.Height, rows),
		MinWidth:  length(s.MinWidth, 1),
		MinHeight: length(s.MinHeight, rows),
		MaxWidth:  length(s.MaxWidth, 1),
		MaxHeight: length(s.MaxHeight, rows),
		RowGap:    cells("row-gap", s.RowGap, rows),
		ColumnGap: cells("column-gap", s.ColumnGap, 1),
		Padding:   edges("padding", s.Padding),
		Margin:    edges("margin", s.Margin),
		Border:    edges("border-width", s.BorderWidth),
		Inset:     layout.Insets{Top: length(s.Inset.Top, rows), Right: length(s.Inset.Right, 1), Bottom: length(s.Inset.Bottom, rows), Left: length(s.Inset.Left, 1)},
		ZIndex:    s.ZIndex,
		RowUnits:  rows,
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
		out.Position = layout.PositionSticky
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
	if reversed(s) && s.Wrap != style.NoWrap {
		unsupported = append(unsupported, "flex-wrap in a reversed direction")
	}
	if reversed(s) != mirrored(s, dir) {
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

func tracks(ts []style.Track, unit int) []layout.Track {
	if ts == nil {
		return nil
	}
	breadth := func(b style.Breadth) layout.Breadth {
		switch b.Kind {
		case style.SizeAuto:
			return layout.Breadth{}
		case style.SizeCells:
			return layout.Breadth{Kind: layout.SizeCells, Value: int(math.Round(b.Value * float64(unit)))}
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

func reversed(s *style.ComputedStyle) bool {
	return s.Display == style.DisplayFlex && (s.Direction == style.RowReverse || s.Direction == style.ColumnReverse)
}

func mirrored(s *style.ComputedStyle, dir text.Direction) bool {
	return dir == text.DirRTL && s.Display == style.DisplayFlex && s.Wrap == style.NoWrap && (s.Direction == style.Row || s.Direction == style.RowReverse)
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
