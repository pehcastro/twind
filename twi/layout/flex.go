package layout

import "math"

type heightMode uint8

const (
	measuring heightMode = iota
	autoHeight
	fixedHeight
)

func Layout(root *Box, width int, height Length) {
	if !visible(root) {
		hide(root)
		return
	}
	s := root.Style
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
	h := heightOf(root, w, viewHeight, viewDefinite)
	viewport := Rect{0, 0, width, h}
	if viewDefinite {
		viewport.H = viewHeight
	}
	screen := container{viewport, viewport}
	place(root, 0, 0, w, h, mode, viewport, screen, screen)
}

type container struct{ box, clip Rect }

func place(b *Box, x, y, w, h int, mode heightMode, clip Rect, absolute, fixed container) {
	s := &b.Style
	b.BorderBox = Rect{x, y, w, h}
	b.PaddingBox = inset(b.BorderBox, s.Border)
	b.ContentBox = inset(b.PaddingBox, s.Padding)
	b.Clip = clip
	if clips(s.Overflow) {
		clip = intersect(clip, b.PaddingBox)
	}
	if b.Measure != nil {
		return
	}
	content, definite, padding := b.ContentBox, mode == fixedHeight, b.PaddingBox
	frames, _ := arrange(b, content.W, content.H, mode)
	if s.Overflow == OverflowScroll {
		b.scroll(frames)
		content.X, content.Y, padding.X, padding.Y = content.X-b.ScrollX, content.Y-b.ScrollY, padding.X-b.ScrollX, padding.Y-b.ScrollY
	}
	if s.Position != PositionStatic {
		absolute = container{padding, clip}
	}
	for i, c := range b.Children {
		cs := &c.Style
		switch {
		case !visible(c):
			hide(c)
		case cs.Position == PositionAbsolute:
			placeOut(c, content, absolute, fixed)
		case cs.Position == PositionFixed:
			placeOut(c, content, fixed, fixed)
		default:
			childMode := autoHeight
			if _, sized := resolve(cs.Height, content.H, definite); sized || definite && (!isRow(s.Direction) || alignOf(s, cs) == AlignStretch) {
				childMode = fixedHeight
			}
			f := frames[i]
			if cs.Position == PositionRelative {
				f.X += shift(cs.Inset.Left, cs.Inset.Right, content.W)
				f.Y += shift(cs.Inset.Top, cs.Inset.Bottom, content.H)
			}
			place(c, content.X+f.X, content.Y+f.Y, f.W, f.H, childMode, clip, absolute, fixed)
		}
	}
}

func (b *Box) scroll(frames []Rect) {
	s, view := b.Style, b.PaddingBox
	right, bottom := 0, 0
	for i, c := range b.Children {
		if visible(c) && flowing(c.Style.Position) {
			f, m := frames[i], c.Style.Margin
			right, bottom = max(right, f.X+f.W+m.Right), max(bottom, f.Y+f.H+m.Bottom)
		}
	}
	b.ScrollWidth = max(view.W, s.Padding.Left+right+s.Padding.Right)
	b.ScrollHeight = max(view.H, s.Padding.Top+bottom+s.Padding.Bottom)
	b.ScrollX = max(min(b.ScrollX, b.ScrollWidth-view.W), 0)
	b.ScrollY = max(min(b.ScrollY, b.ScrollHeight-view.H), 0)
}

func shift(near, far Length, base int) int {
	if v, ok := resolve(near, base, true); ok {
		return v
	}
	v, _ := resolve(far, base, true)
	return -v
}

func placeOut(b *Box, static Rect, cb, fixed container) {
	s, m, in, area := b.Style, b.Style.Margin, b.Style.Inset, cb.box
	left, hasLeft := resolve(in.Left, area.W, true)
	right, hasRight := resolve(in.Right, area.W, true)
	top, hasTop := resolve(in.Top, area.H, true)
	bottom, hasBottom := resolve(in.Bottom, area.H, true)
	w, sized := resolve(s.Width, area.W, true)
	if avail := area.W - left - right - m.Left - m.Right; !sized && hasLeft && hasRight {
		w = avail
	} else if !sized {
		w = contentWidth(b, avail)
	}
	w = limit(s.MinWidth, s.MaxWidth, area.W, true).clamp(w)
	mode := fixedHeight
	h, sized := resolve(s.Height, area.H, true)
	if !sized && hasTop && hasBottom {
		h = area.H - top - bottom - m.Top - m.Bottom
	} else if !sized {
		h, mode = contentHeight(b, w), autoHeight
	}
	h = limit(s.MinHeight, s.MaxHeight, area.H, true).clamp(h)
	x := edge(hasLeft, hasRight, area.X+left+m.Left, area.X+area.W-right-m.Right-w, static.X+m.Left)
	y := edge(hasTop, hasBottom, area.Y+top+m.Top, area.Y+area.H-bottom-m.Bottom-h, static.Y+m.Top)
	place(b, x, y, w, h, mode, cb.clip, cb, fixed)
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

func hide(b *Box) {
	b.BorderBox, b.PaddingBox, b.ContentBox, b.Clip = Rect{}, Rect{}, Rect{}, Rect{}
	for _, c := range b.Children {
		hide(c)
	}
}

func arrange(b *Box, innerW, innerH int, mode heightMode) ([]Rect, int) {
	s := &b.Style
	row, fixed := isRow(s.Direction), mode == fixedHeight
	var shown []int
	for i, c := range b.Children {
		if visible(c) && flowing(c.Style.Position) {
			shown = append(shown, i)
		}
	}
	frames := make([]Rect, len(b.Children))
	gap, space, flexing := s.RowGap, innerH, mode != measuring
	if row {
		gap, space, flexing = s.ColumnGap, innerW, true
	}
	items := make([]flexItem, len(shown))
	for k, i := range shown {
		c := b.Children[i]
		cs, m := &c.Style, c.Style.Margin
		if row {
			items[k] = newItem(cs, cs.Width, cs.MinWidth, cs.MaxWidth, innerW, true)
			items[k].margins = m.Left + m.Right
			if w, ok := aspectWidth(cs, innerH, fixed); items[k].content && ok {
				items[k].basis, items[k].content = w, false
			}
			if items[k].content {
				items[k].basis = intrinsic(c, maxContent)
			} else {
				items[k].min = autoMin(c, items[k], innerW)
			}
		} else {
			a, avail := alignOf(s, cs), innerW-m.Left-m.Right
			if a == AlignStretch && cs.Width.Unit == Auto {
				frames[i].W = limit(cs.MinWidth, cs.MaxWidth, innerW, true).clamp(avail)
			} else {
				frames[i].W = fitWidth(c, avail)
			}
			frames[i].X = m.Left + offset(a, avail-frames[i].W)
			items[k] = newItem(cs, cs.Height, cs.MinHeight, cs.MaxHeight, innerH, fixed)
			items[k].margins = m.Top + m.Bottom
			content, automatic := items[k].content, cs.MinHeight.Unit == Auto && !clips(cs.Overflow)
			if content || automatic {
				natural := contentHeight(c, frames[i].W)
				if content {
					items[k].basis = natural
				}
				if h, sized := resolve(cs.Height, innerH, fixed); sized {
					natural = min(natural, h)
				}
				if automatic {
					items[k].min = min(natural, items[k].max)
				}
			}
		}
	}
	var breaks [2]int
	lines := breaks[:1]
	if row && s.Wrap != NoWrap {
		run := -gap
		for k, it := range items {
			size := it.clamp(it.basis) + it.margins
			if k > lines[len(lines)-1] && run+gap+size > space {
				lines, run = append(lines, k), -gap
			}
			run += gap + size
		}
	}
	lines = append(lines, len(shown))
	multi, cross := len(lines) > 2, 0
	for l := range len(lines) - 1 {
		first, end := lines[l], lines[l+1]
		group := items[first:end]
		spent, sizes := gap*max(len(group)-1, 0), make([]int, len(group))
		used := spent
		for k, it := range group {
			spent += it.margins
			sizes[k] = it.clamp(it.basis)
			used += it.margins + sizes[k]
		}
		if row && used > space {
			for k, it := range group {
				if it.content && it.shrink > 0 {
					group[k].min = autoMin(b.Children[shown[first+k]], it, innerW)
				}
			}
		}
		free := 0
		if flexing {
			sizes, used = flexSizes(group, space-spent), spent
			for _, size := range sizes {
				used += size
			}
			free = space - used
		}
		pos, extra := justify(s.Justify, free, len(sizes))
		line := innerH
		if multi {
			line = 0
		}
		for k, i := range shown[first:end] {
			m := b.Children[i].Style.Margin
			if !row {
				frames[i].Y, frames[i].H = pos+m.Top, sizes[k]
				pos += m.Top + sizes[k] + m.Bottom + gap + extra[k]
				continue
			}
			frames[i].X, frames[i].W = pos+m.Left, sizes[k]
			frames[i].H = heightOf(b.Children[i], sizes[k], innerH, fixed)
			pos += m.Left + sizes[k] + m.Right + gap + extra[k]
			if mode == measuring || multi {
				line = max(line, frames[i].H+m.Top+m.Bottom)
			}
		}
		if !row {
			return frames, used
		}
		for _, i := range shown[first:end] {
			cs, m := &b.Children[i].Style, b.Children[i].Style.Margin
			a, avail := alignOf(s, cs), line-m.Top-m.Bottom
			if _, sized := resolve(cs.Height, innerH, fixed); !sized && a == AlignStretch {
				frames[i].H = limit(cs.MinHeight, cs.MaxHeight, innerH, fixed).clamp(avail)
			}
			frames[i].Y = cross + m.Top + offset(a, avail-frames[i].H)
		}
		cross += line + s.RowGap
	}
	cross = max(cross-s.RowGap, 0)
	if s.Wrap == WrapReverse {
		for _, i := range shown {
			frames[i].Y = cross - frames[i].Y - frames[i].H
		}
	}
	return frames, cross
}

func fitWidth(b *Box, avail int) int {
	s := &b.Style
	bounds := limit(s.MinWidth, s.MaxWidth, avail, true)
	if w, ok := resolve(s.Width, avail, true); ok {
		return bounds.clamp(w)
	}
	if w, ok := aspectWidth(s, 0, false); ok {
		return bounds.clamp(w)
	}
	return bounds.clamp(contentWidth(b, avail))
}

func autoMin(c *Box, it flexItem, innerW int) int {
	cs := &c.Style
	if cs.MinWidth.Unit != Auto || clips(cs.Overflow) {
		return it.min
	}
	least := intrinsic(c, minContent)
	if w, sized := resolve(cs.Width, innerW, true); sized {
		least = min(least, w)
	}
	return min(least, it.max)
}

func aspectWidth(s *Style, base int, baseDefinite bool) (int, bool) {
	if s.Aspect == (Ratio{}) {
		return 0, false
	}
	h, ok := resolve(s.Height, base, baseDefinite)
	return scale(h, s.Aspect.W, s.Aspect.H), ok
}

func scale(v, num, den int) int {
	return (v*num + den/2) / den
}

type sizing uint8

const (
	minContent sizing = iota
	maxContent
)

func intrinsic(b *Box, mode sizing) int {
	s := &b.Style
	frameW, _ := frame(s)
	if b.Measure != nil {
		avail := 0
		if mode == maxContent {
			avail = math.MaxInt
		}
		w, _ := b.Measure(avail)
		return w + frameW
	}
	row := isRow(s.Direction)
	sums, content, count := row && (mode == maxContent || s.Wrap == NoWrap), 0, 0
	for _, c := range b.Children {
		cs := &c.Style
		if !visible(c) || !flowing(cs.Position) {
			continue
		}
		w, ok := resolve(cs.Width, 0, false)
		if !ok {
			w, ok = aspectWidth(cs, 0, false)
		}
		if !ok {
			childMode := mode
			if row && cs.Shrink == 0 {
				childMode = maxContent
			}
			w = intrinsic(c, childMode)
		}
		w = limit(cs.MinWidth, cs.MaxWidth, 0, false).clamp(w) + cs.Margin.Left + cs.Margin.Right
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
	return content + frameW
}

func contentWidth(b *Box, avail int) int {
	s := &b.Style
	frameW, _ := frame(s)
	inner := max(avail-frameW, 0)
	if b.Measure != nil {
		w, _ := b.Measure(inner)
		return min(w, inner) + frameW
	}
	row, content, count := isRow(s.Direction), 0, 0
	for _, c := range b.Children {
		if !visible(c) || !flowing(c.Style.Position) {
			continue
		}
		marginW := c.Style.Margin.Left + c.Style.Margin.Right
		w := fitWidth(c, inner-marginW) + marginW
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
	return min(content, inner) + frameW
}

func heightOf(b *Box, w, base int, baseDefinite bool) int {
	s := &b.Style
	h, ok := resolve(s.Height, base, baseDefinite)
	if !ok {
		h = contentHeight(b, w)
	}
	return limit(s.MinHeight, s.MaxHeight, base, baseDefinite).clamp(h)
}

func contentHeight(b *Box, w int) int {
	if a := b.Style.Aspect; a != (Ratio{}) {
		return scale(w, a.H, a.W)
	}
	frameW, frameH := frame(&b.Style)
	inner := max(w-frameW, 0)
	if b.Measure != nil {
		_, h := b.Measure(inner)
		return h + frameH
	}
	_, h := arrange(b, inner, 0, measuring)
	return h + frameH
}
