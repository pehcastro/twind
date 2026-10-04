package layout

import konst "github.com/twind-dev/twind/internal/konst/layout"

type heightMode uint8

const (
	measuring heightMode = iota
	autoHeight
	fixedHeight
)

type arena struct {
	ints   []int
	items  []flexItem
	tracks []track
	cells  []gridItem
	root   *Box
	screen container
	reuse  bool
}

func grab[T any](stack *[]T, n int) []T {
	s, top := *stack, len(*stack)
	if top+n > cap(s) {
		s = make([]T, top, 2*cap(s)+n)
	}
	*stack = s[:top+n]
	return s[top : top+n : top+n]
}

func grabZero[T any](stack *[]T, n int) []T {
	s := grab(stack, n)
	clear(s)
	return s
}

func Layout(root *Box, width int, height Length) {
	checkUnit(height.Unit)
	ready(root)
	if !visible(root) {
		hide(root)
		return
	}
	if root.arena == nil {
		root.arena = &arena{}
	}
	a := root.arena
	a.ints, a.items, a.tracks, a.cells = a.ints[:0], a.items[:0], a.tracks[:0], a.cells[:0]
	s := &root.Style
	w, ok := resolve(s.Width, width, true)
	if !ok {
		w = width
	}
	w = limit(s.MinWidth, s.MaxWidth, width, true).clamp(w)
	viewHeight, viewDefinite := resolve(height, 0, false)
	mode := autoHeight
	if _, sized := resolve(s.Height, viewHeight, viewDefinite); sized {
		mode = fixedHeight
	}
	h := a.heightOf(root, w, viewHeight, viewDefinite)
	viewport := Rect{0, 0, width, h}
	if viewDefinite {
		viewport.H = viewHeight
	}
	screen := container{viewport, viewport}
	a.root, a.screen, a.reuse = root, screen, a.screen == screen
	a.place(root, Rect{0, 0, w, h}, mode, &viewport, &screen)
}

type container struct{ box, clip Rect }

func (a *arena) place(b *Box, border Rect, mode heightMode, outer *Rect, absolute *container) bool {
	s, leaf := &b.Style, b.Measure != nil
	moved := !a.reuse || b.stale || b.BorderBox != border || b.Clip != *outer
	if !moved && b.spot.state == placed && (leaf || b.spot.mode == mode && b.spot.absolute.box == absolute.box && b.spot.absolute.clip == absolute.clip) {
		return false
	}
	padding := inset(border, s.Border)
	content := inset(padding, s.Padding)
	b.BorderBox, b.PaddingBox, b.ContentBox, b.Clip, b.spot.state = border, padding, content, *outer, placed
	if leaf {
		b.Moved, b.stale = b.Moved || moved, false
		return moved
	}
	b.spot.mode, b.spot.absolute = mode, *absolute
	clip := *outer
	if clips(s.Overflow) {
		clip = intersect(clip, padding)
	}
	m, definite := &b.memo, mode == fixedHeight
	if !m.framesKnown || m.framesW != content.W || m.framesH != content.H || m.framesMode != mode {
		a.arrange(b, content.W, content.H, mode)
	}
	if s.Overflow == OverflowScroll {
		moved = b.scroll() || moved
		content.X, content.Y, padding.X, padding.Y = content.X-b.ScrollX, content.Y-b.ScrollY, padding.X-b.ScrollX, padding.Y-b.ScrollY
	}
	if s.Position != PositionStatic {
		absolute = &container{padding, clip}
	}
	known := definite || s.Display == DisplayGrid
	for _, c := range children(b) {
		cs := &c.Style
		switch {
		case !visible(c):
			moved = hide(c) || moved
		case cs.Position == PositionAbsolute:
			moved = a.placeOut(c, content, absolute) || moved
		case cs.Position == PositionFixed:
			moved = a.placeOut(c, content, &a.screen) || moved
		default:
			childMode := autoHeight
			if _, sized := c.height(content.H, known); sized || known && (!isRow(s.Direction) || alignOf(s, cs) == AlignStretch) {
				childMode = fixedHeight
			}
			f := c.frame
			if cs.Position == PositionRelative {
				f.X += shift(cs.Inset.Left, cs.Inset.Right, content.W)
				f.Y += shift(cs.Inset.Top, cs.Inset.Bottom, content.H)
			}
			f.X, f.Y = f.X+content.X, f.Y+content.Y
			if cs.Position == PositionSticky {
				f = a.stuck(b, cs, f, content)
			}
			moved = a.place(c, f, childMode, &clip, absolute) || moved
		}
	}
	b.Moved, b.stale = b.Moved || moved, false
	return moved
}

func (a *arena) stuck(b *Box, cs *Style, f, bound Rect) Rect {
	if s := &b.Style; s.Overflow == OverflowScroll {
		bound.W, bound.H = max(bound.W, b.ScrollWidth-s.Padding.Left-s.Padding.Right), max(bound.H, b.ScrollHeight-s.Padding.Top-s.Padding.Bottom)
	}
	port := a.screen.box
	for p := b; ; p = p.parent {
		if p.Style.Overflow == OverflowScroll {
			port = p.PaddingBox
			break
		}
		if p == a.root {
			break
		}
	}
	m := cs.Margin
	f.X = stick(f.X, f.W, cs.Inset.Left, cs.Inset.Right, port.X, port.W, bound.X+m.Left, bound.X+bound.W-m.Right)
	f.Y = stick(f.Y, f.H, cs.Inset.Top, cs.Inset.Bottom, port.Y, port.H, bound.Y+m.Top, bound.Y+bound.H-m.Bottom)
	return f
}

func stick(at, size int, near, far Length, view, span, low, high int) int {
	if v, ok := resolve(near, span, true); ok {
		at = max(at, min(view+v, high-size))
	}
	if v, ok := resolve(far, span, true); ok {
		at = min(at, max(view+span-v-size, low))
	}
	return at
}

func (b *Box) scroll() bool {
	s, view, was := &b.Style, b.PaddingBox, [4]int{b.ScrollX, b.ScrollY, b.ScrollWidth, b.ScrollHeight}
	right, bottom := 0, 0
	for _, c := range children(b) {
		if visible(c) && flowing(c.Style.Position) {
			f, m := c.frame, c.Style.Margin
			right, bottom = max(right, f.X+f.W+m.Right), max(bottom, f.Y+f.H+m.Bottom)
		}
	}
	b.ScrollWidth = max(view.W, s.Padding.Left+right+s.Padding.Right)
	b.ScrollHeight = max(view.H, s.Padding.Top+bottom+s.Padding.Bottom)
	b.ScrollX = max(min(b.ScrollX, b.ScrollWidth-view.W), 0)
	limit := b.ScrollHeight - view.H
	limit = -whole(-limit, s.RowUnits)
	b.ScrollY = max(min(b.ScrollY, limit), 0)
	return was != [4]int{b.ScrollX, b.ScrollY, b.ScrollWidth, b.ScrollHeight}
}

func shift(near, far Length, base int) int {
	if v, ok := resolve(near, base, true); ok {
		return v
	}
	v, _ := resolve(far, base, true)
	return -v
}

func (a *arena) placeOut(b *Box, static Rect, cb *container) bool {
	s, m, in, area := &b.Style, b.Style.Margin, b.Style.Inset, cb.box
	left, hasLeft := resolve(in.Left, area.W, true)
	right, hasRight := resolve(in.Right, area.W, true)
	top, hasTop := resolve(in.Top, area.H, true)
	bottom, hasBottom := resolve(in.Bottom, area.H, true)
	w, sized := resolve(s.Width, area.W, true)
	if avail := area.W - left - right - m.Left - m.Right; !sized && hasLeft && hasRight {
		w = avail
	} else if !sized {
		w = min(max(a.intrinsic(b, minContent), avail), a.intrinsic(b, maxContent))
	}
	w = limit(s.MinWidth, s.MaxWidth, area.W, true).clamp(w)
	mode := fixedHeight
	h, sized := resolve(s.Height, area.H, true)
	if !sized && hasTop && hasBottom {
		h = area.H - top - bottom - m.Top - m.Bottom
	} else if !sized {
		h, mode = a.contentHeight(b, w), autoHeight
	}
	h = limit(s.MinHeight, s.MaxHeight, area.H, true).clamp(h)
	x := edge(hasLeft, hasRight, area.X+left+m.Left, area.X+area.W-right-m.Right-w, static.X+m.Left)
	y := edge(hasTop, hasBottom, area.Y+top+m.Top, area.Y+area.H-bottom-m.Bottom-h, static.Y+m.Top)
	return a.place(b, Rect{x, y + b.nudge(y), w, h}, mode, &cb.clip, cb)
}

func edge(hasNear, hasFar bool, near, far, static int) int {
	if hasNear {
		return near
	}
	if hasFar {
		return far
	}
	return static
}

func hide(b *Box) bool {
	if !b.stale && b.spot.state == hidden {
		return false
	}
	moved := b.BorderBox != Rect{} || b.PaddingBox != Rect{} || b.ContentBox != Rect{} || b.Clip != Rect{}
	b.BorderBox, b.PaddingBox, b.ContentBox, b.Clip, b.spot.state = Rect{}, Rect{}, Rect{}, Rect{}, hidden
	for _, c := range children(b) {
		moved = hide(c) || moved
	}
	b.Moved, b.stale = b.Moved || moved, false
	return moved
}

func (a *arena) arrange(b *Box, innerW, innerH int, mode heightMode) int {
	m, used, settled := &b.memo, 0, true
	ints, flexItems := len(a.ints), len(a.items)
	switch {
	case b.Style.Display == DisplayGrid:
		used = a.arrangeGrid(b, innerW, innerH, mode)
	case isRow(b.Style.Direction):
		used = a.arrangeRow(b, innerW, innerH, mode)
	default:
		used, settled = a.arrangeColumn(b, innerW, innerH, mode)
	}
	a.ints, a.items = a.ints[:ints], a.items[:flexItems]
	m.framesW, m.framesH, m.framesMode, m.framesKnown = innerW, innerH, mode, settled || mode != measuring
	if mode == measuring {
		m.framesH, m.framesMode = used, autoHeight
	}
	return used
}

func (a *arena) arrangeColumn(b *Box, innerW, innerH int, mode heightMode) (int, bool) {
	s, fixed := &b.Style, mode == fixedHeight
	kids := children(b)
	items := grab(&a.items, len(kids))[:0]
	for _, c := range kids {
		cs := &c.Style
		if cs.Display == DisplayNone || !flowing(cs.Position) {
			continue
		}
		items = items[:len(items)+1]
		it, m, f := &items[len(items)-1], &cs.Margin, &c.frame
		it.box = c
		al, avail := alignOf(s, cs), innerW-m.Left-m.Right
		if al == AlignStretch && cs.Width.Unit == Auto {
			f.W = c.widthLimit(innerW, true).clamp(avail)
		} else {
			f.W = a.fitWidth(c, avail)
		}
		f.X = m.Left + offset(al, avail-f.W)
		it.set(c, cs.Height, cs.MinHeight, cs.MaxHeight, innerH, fixed)
		it.margins = m.Top + m.Bottom
		content, automatic := it.content, cs.MinHeight.Unit == Auto && !clips(cs.Overflow)
		if content || automatic {
			natural := a.contentHeight(c, f.W)
			if content {
				it.basis = natural
			}
			if h, sized := c.height(innerH, fixed); sized {
				natural = min(natural, h)
			}
			if automatic {
				it.min = min(natural, it.max)
			}
		}
	}
	spent, used, settled := hypothetical(items, s.RowGap)
	free, step := 0, max(s.RowUnits, 1)
	if mode != measuring {
		if !rigid(items, innerH-used) {
			used = spent + a.flexSizes(items, innerH-spent, step)
		}
		free = innerH - used
	}
	pos, extra := a.justify(s.Justify, free, len(items), step)
	for k := range items {
		it := &items[k]
		m, f := &it.box.Style.Margin, &it.box.frame
		pos += it.box.nudge(pos + m.Top)
		f.Y, f.H = pos+m.Top, it.size
		pos += m.Top + it.size + m.Bottom + s.RowGap
		if extra != nil {
			pos += extra[k]
		}
	}
	if len(items) > 0 {
		used = max(used, pos-s.RowGap)
	}
	return used, settled
}

func hypothetical(items []flexItem, gap int) (spent, used int, settled bool) {
	spent = gap * max(len(items)-1, 0)
	used, settled = spent, true
	for k := range items {
		it := &items[k]
		it.size = it.clamp(it.basis)
		spent += it.margins
		used += it.margins + it.size
		settled = settled && (it.shrink*it.basis <= 0 || it.basis <= it.size)
	}
	return spent, used, settled
}

func (a *arena) arrangeRow(b *Box, innerW, innerH int, mode heightMode) int {
	s, fixed, gap := &b.Style, mode == fixedHeight, b.Style.ColumnGap
	kids := children(b)
	items := grab(&a.items, len(kids))[:0]
	for _, c := range kids {
		cs := &c.Style
		if cs.Display == DisplayNone || !flowing(cs.Position) {
			continue
		}
		items = items[:len(items)+1]
		it := &items[len(items)-1]
		it.box = c
		it.set(c, cs.Width, cs.MinWidth, cs.MaxWidth, innerW, true)
		it.margins = cs.Margin.Left + cs.Margin.Right
		if w, ok := aspectWidth(cs, innerH, fixed); it.content && ok {
			it.basis, it.content = w, false
		}
		if it.content {
			it.basis = a.intrinsic(c, maxContent)
		} else {
			it.min = a.autoMin(c, it.bounds, innerW)
		}
	}
	first, end := 0, len(items)
	if s.Wrap != NoWrap {
		end = lineEnd(items, 0, gap, innerW)
	}
	multi, cross := end < len(items), 0
	for {
		group := items[first:end]
		spent, used, _ := hypothetical(group, gap)
		if used > innerW {
			for k := range group {
				if it := &group[k]; it.content && it.shrink > 0 {
					it.min = a.autoMin(it.box, it.bounds, innerW)
				}
			}
		}
		if used > innerW || !rigid(group, innerW-used) {
			used = spent + a.flexSizes(group, innerW-spent, 1)
		}
		pos, extra := a.justify(s.Justify, innerW-used, len(group), 1)
		line := innerH
		if multi {
			line = 0
		}
		for k := range group {
			it := &group[k]
			c := it.box
			m, f := &c.Style.Margin, &c.frame
			f.X, f.W = pos+m.Left, it.size
			f.H = a.heightOf(c, it.size, innerH, fixed)
			pos += m.Left + it.size + m.Right + gap
			if extra != nil {
				pos += extra[k]
			}
			if mode == measuring || multi {
				line = max(line, f.H+m.Top+m.Bottom)
			}
		}
		reach := line
		for k := range group {
			c := group[k].box
			cs, f := &c.Style, &c.frame
			al, avail := alignOf(s, cs), line-cs.Margin.Top-cs.Margin.Bottom
			if _, sized := c.height(innerH, fixed); !sized && al == AlignStretch {
				f.H = c.heightLimit(innerH, fixed).clamp(avail)
			}
			f.Y = cross + cs.Margin.Top + whole(offset(al, avail-f.H), s.RowUnits)
			f.Y += c.nudge(f.Y)
			reach = max(reach, f.Y+f.H+cs.Margin.Bottom-cross)
		}
		if mode == measuring || multi {
			line = reach
		}
		cross += line + s.RowGap
		if end == len(items) {
			break
		}
		first, end = end, lineEnd(items, end, gap, innerW)
	}
	cross = max(cross-s.RowGap, 0)
	if s.Wrap == WrapReverse {
		for k := range items {
			f := &items[k].box.frame
			f.Y = cross - f.Y - f.H
		}
	}
	return cross
}

func lineEnd(items []flexItem, first, gap, space int) int {
	run := -gap
	for k := first; k < len(items); k++ {
		size := items[k].clamp(items[k].basis) + items[k].margins
		if k > first && run+gap+size > space {
			return k
		}
		run += gap + size
	}
	return len(items)
}

func (a *arena) fitWidth(b *Box, avail int) int {
	s := &b.Style
	bounds := limit(s.MinWidth, s.MaxWidth, avail, true)
	if w, ok := resolve(s.Width, avail, true); ok {
		return bounds.clamp(w)
	}
	if w, ok := aspectWidth(s, 0, false); ok {
		return bounds.clamp(w)
	}
	return bounds.clamp(a.contentWidth(b, avail))
}

func (a *arena) autoMin(c *Box, r bounds, innerW int) int {
	cs := &c.Style
	if cs.MinWidth.Unit != Auto || clips(cs.Overflow) {
		return r.min
	}
	least := a.intrinsic(c, minContent)
	if w, sized := resolve(cs.Width, innerW, true); sized {
		least = min(least, w)
	}
	return min(least, r.max)
}

func aspectWidth(s *Style, base int, baseDefinite bool) (int, bool) {
	if s.Aspect == (Ratio{}) {
		return 0, false
	}
	h, ok := resolve(s.Height, base, baseDefinite)
	return scale(h, s.Aspect.W, s.Aspect.H*max(s.RowUnits, 1)), ok
}

func scale(v, num, den int) int {
	return (v*num + den/2) / den
}

type sizing uint8

const (
	minContent sizing = iota
	maxContent
)

func (a *arena) intrinsic(b *Box, mode sizing) int {
	m := &b.memo
	if m.intrinsicKnown[mode] {
		return m.intrinsic[mode]
	}
	s, frameW := &b.Style, b.frameW
	content := 0
	switch {
	case b.Measure != nil && mode == maxContent:
		content, m.natural = b.Measure(konst.Unbounded)
	case b.Measure != nil:
		content, _ = b.Measure(0)
	case s.Display == DisplayGrid:
		content = a.gridWidth(b, mode)
	default:
		row := isRow(s.Direction)
		sums, count := row && (mode == maxContent || s.Wrap == NoWrap), 0
		for _, c := range children(b) {
			cs := &c.Style
			if !visible(c) || !flowing(cs.Position) {
				continue
			}
			w, ok := 0, false
			if !c.plain {
				if w, ok = resolve(cs.Width, 0, false); !ok {
					w, ok = aspectWidth(cs, 0, false)
				}
			}
			if !ok {
				childMode := mode
				if row && cs.Shrink == 0 {
					childMode = maxContent
				}
				w = a.intrinsic(c, childMode)
			}
			w = c.widthLimit(0, false).clamp(w) + cs.Margin.Left + cs.Margin.Right
			if sums {
				content += w
			} else {
				content = max(content, w)
			}
			count++
		}
		if sums {
			content += s.ColumnGap * max(count-1, 0)
		}
	}
	m.intrinsic[mode], m.intrinsicKnown[mode] = content+frameW, true
	return m.intrinsic[mode]
}

func (a *arena) contentWidth(b *Box, avail int) int {
	m := &b.memo
	if m.widthKnown && m.widthAvail == avail {
		return m.width
	}
	s, frameW := &b.Style, b.frameW
	inner := max(avail-frameW, 0)
	content := 0
	switch {
	case b.Measure != nil && b.unwrapped(inner):
		content = m.intrinsic[maxContent] - frameW
	case b.Measure != nil:
		content, _ = b.Measure(inner)
	case s.Display == DisplayGrid:
		content = a.intrinsic(b, maxContent) - frameW
	default:
		row, count := isRow(s.Direction), 0
		for _, c := range children(b) {
			if !visible(c) || !flowing(c.Style.Position) {
				continue
			}
			marginW := c.Style.Margin.Left + c.Style.Margin.Right
			w := a.fitWidth(c, inner-marginW) + marginW
			if row {
				content += w
			} else {
				content = max(content, w)
			}
			count++
		}
		if row {
			content += s.ColumnGap * max(count-1, 0)
		}
	}
	m.widthAvail, m.width, m.widthKnown = avail, min(content, inner)+frameW, true
	return m.width
}

func (b *Box) unwrapped(inner int) bool {
	return b.memo.intrinsicKnown[maxContent] && inner >= b.memo.intrinsic[maxContent]-b.frameW
}

func (a *arena) heightOf(b *Box, w, base int, baseDefinite bool) int {
	h, ok := b.height(base, baseDefinite)
	if !ok {
		h = a.contentHeight(b, w)
	}
	return b.heightLimit(base, baseDefinite).clamp(h)
}

func (a *arena) contentHeight(b *Box, w int) int {
	if r := b.Style.Aspect; r != (Ratio{}) {
		return scale(w, r.H, r.W) * max(b.Style.RowUnits, 1)
	}
	m := &b.memo
	if m.heightKnown && m.heightWidth == w {
		return m.height
	}
	inner := max(w-b.frameW, 0)
	var h int
	switch {
	case b.Measure != nil && b.unwrapped(inner):
		h = m.natural
	case b.Measure != nil:
		_, h = b.Measure(inner)
	default:
		h = a.arrange(b, inner, 0, measuring)
	}
	m.heightWidth, m.height, m.heightKnown = w, h+b.frameH, true
	return m.height
}
