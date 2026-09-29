package layout

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
	s := b.Style
	b.BorderBox = Rect{x, y, w, h}
	b.PaddingBox = inset(b.BorderBox, s.Border)
	b.ContentBox = inset(b.PaddingBox, s.Padding)
	b.Clip = clip
	if clips(s.Overflow) {
		clip = intersect(clip, b.PaddingBox)
	}
	if s.Position != PositionStatic {
		absolute = container{b.PaddingBox, clip}
	}
	if b.Measure != nil {
		return
	}
	content, definite := b.ContentBox, mode == fixedHeight
	frames, _ := arrange(b, content.W, content.H, mode)
	for i, c := range b.Children {
		cs := c.Style
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
	s := b.Style
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
	used := gap * max(len(shown)-1, 0)
	items := make([]flexItem, len(shown))
	sizes := make([]int, len(shown))
	for k, i := range shown {
		c := b.Children[i]
		cs, m := c.Style, c.Style.Margin
		var content bool
		if row {
			used += m.Left + m.Right
			items[k], content = newItem(cs, cs.Width, cs.MinWidth, cs.MaxWidth, innerW, true)
			if content {
				items[k].basis = contentWidth(c, innerW-m.Left-m.Right)
			}
		} else {
			used += m.Top + m.Bottom
			a, avail := alignOf(s, cs), innerW-m.Left-m.Right
			if a == AlignStretch && cs.Width.Unit == Auto {
				frames[i].W = limit(cs.MinWidth, cs.MaxWidth, innerW, true).clamp(avail)
			} else {
				frames[i].W = fitWidth(c, avail)
			}
			frames[i].X = m.Left + offset(a, avail-frames[i].W)
			items[k], content = newItem(cs, cs.Height, cs.MinHeight, cs.MaxHeight, innerH, fixed)
			if content {
				items[k].basis = contentHeight(c, frames[i].W)
			}
		}
		sizes[k] = items[k].clamp(items[k].basis)
	}
	if flexing {
		sizes = flexSizes(items, space-used)
	}
	for _, size := range sizes {
		used += size
	}
	free := 0
	if flexing {
		free = space - used
	}
	pos, extra := justify(s.Justify, free, len(shown))
	line := innerH
	for k, i := range shown {
		m := b.Children[i].Style.Margin
		if !row {
			frames[i].Y, frames[i].H = pos+m.Top, sizes[k]
			pos += m.Top + sizes[k] + m.Bottom + gap + extra[k]
			continue
		}
		frames[i].X, frames[i].W = pos+m.Left, sizes[k]
		frames[i].H = heightOf(b.Children[i], sizes[k], innerH, fixed)
		pos += m.Left + sizes[k] + m.Right + gap + extra[k]
		if mode == measuring {
			line = max(line, frames[i].H+m.Top+m.Bottom)
		}
	}
	if !row {
		return frames, used
	}
	for _, i := range shown {
		cs, m := b.Children[i].Style, b.Children[i].Style.Margin
		a, avail := alignOf(s, cs), line-m.Top-m.Bottom
		if _, sized := resolve(cs.Height, innerH, fixed); !sized && a == AlignStretch {
			frames[i].H = limit(cs.MinHeight, cs.MaxHeight, innerH, fixed).clamp(avail)
		}
		frames[i].Y = m.Top + offset(a, avail-frames[i].H)
	}
	return frames, line
}

func fitWidth(b *Box, avail int) int {
	s := b.Style
	bounds := limit(s.MinWidth, s.MaxWidth, avail, true)
	if w, ok := resolve(s.Width, avail, true); ok {
		return bounds.clamp(w)
	}
	return bounds.clamp(contentWidth(b, avail))
}

func contentWidth(b *Box, avail int) int {
	s := b.Style
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
	s := b.Style
	h, ok := resolve(s.Height, base, baseDefinite)
	if !ok {
		h = contentHeight(b, w)
	}
	return limit(s.MinHeight, s.MaxHeight, base, baseDefinite).clamp(h)
}

func contentHeight(b *Box, w int) int {
	frameW, frameH := frame(b.Style)
	inner := max(w-frameW, 0)
	if b.Measure != nil {
		_, h := b.Measure(inner)
		return h + frameH
	}
	_, h := arrange(b, inner, 0, measuring)
	return h + frameH
}
