package present

import (
	"fmt"
	"image"
	"math"
	"slices"

	paintkonst "github.com/pehcastro/twind/internal/konst/paint"
	konst "github.com/pehcastro/twind/internal/konst/scene"
	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/terminal"
)

func (s *Screen) scroll(f *scene.Frame, sc scene.Scroll) {
	clip, by := f.Layers[sc.Layer].Clip, sc.By
	area := image.Rect(clip.Min.X/s.Cell.X, clip.Min.Y/s.Cell.Y, clip.Max.X/s.Cell.X, clip.Max.Y/s.Cell.Y)
	surface := s.Graphics == terminal.GraphicsGDI
	if !s.Margins && !surface {
		area.Min.X, area.Max.X = 0, s.cols
	}
	lines := by.Y / s.Cell.Y
	if s.Graphics != terminal.GraphicsSixel && !surface || by.X != 0 || max(lines, -lines) >= area.Dy() {
		s.damage(clip)
		return
	}
	if !surface {
		s.region(area, lines)
	}
	s.touch(area.Inset(-1))
	cells := func(y int) (int, int) { return y*s.cols + area.Min.X, y*s.cols + area.Max.X }
	slide(area.Min.Y, area.Max.Y, lines, func(dst, src int) {
		from, to := cells(src)
		at, _ := cells(dst)
		copy(s.samples[at:], s.samples[from:to])
		copy(s.sampled[at:], s.sampled[from:to])
		if surface {
			return
		}
		copy(s.shown.Row(dst)[area.Min.X:area.Max.X], s.shown.Row(src)[area.Min.X:area.Max.X])
		if s.owners != nil {
			copy(s.owners[at:], s.owners[from:to])
		}
	}, func(dst int) {
		from, to := cells(dst)
		clear(s.sampled[from:to])
		s.damage(s.pixels(image.Rect(area.Min.X, dst, area.Max.X, dst+1)))
		if !surface {
			s.shown.Fill(buffer.Rect{X: area.Min.X, Y: dst, W: area.Dx(), H: 1}, buffer.Cell{Grapheme: "\x00"})
			s.disown(area.Min.X, dst, area.Dx())
		}
	})
	if s.owners != nil {
		s.lose(area, lines)
	}
	px := s.pixels(area)
	span := paintkonst.TileColumns * s.Cell.X
	for left := px.Min.X / span * span; left < px.Max.X; left += span {
		c := s.column(left)
		lo, hi := max(px.Min.X, left)-left, min(px.Max.X, left+span)-left
		clear(s.splices)
		if lo == 0 && hi == c.width {
			s.blank = append(s.blank[:0], run{End: int32(c.width)})
			c.shift(px.Min.Y, px.Max.Y, by.Y, c.intern(s.blank, s.digest(&s.key, s.blank)))
			continue
		}
		slide(px.Min.Y, px.Max.Y, by.Y, func(dst, src int) { s.splice(c, dst, c.lineOf[src], lo, hi) }, func(dst int) { s.splice(c, dst, -1, lo, hi) })
	}
	moved := s.moved
	if surface {
		moved = s.shifted
	}
	for t, tile := range s.tiles {
		moved[t] = moved[t] || tile.Overlaps(area)
	}
	moving := make([]bool, len(f.Layers))
	for i := range f.Layers {
		l := &f.Layers[i]
		if moving[i] = i == sc.Layer || l.Parent >= 0 && moving[l.Parent]; moving[i] {
			continue
		}
		for j := range l.Boxes {
			s.unshifted(l, &l.Boxes[j], by.Y, px)
		}
	}
	for i, r := range s.changed {
		if l := s.changedIn[i]; l < 0 || !moving[l] {
			s.damage(r.Intersect(px).Add(image.Pt(0, by.Y)).Intersect(px))
		}
	}
}

func (s *Screen) region(area image.Rectangle, lines int) {
	s.out.WriteString(termkonst.Reset)
	if s.Margins {
		s.out.WriteString(termkonst.MarginsOn)
	}
	fmt.Fprintf(&s.out, "%s%d;%d%s", termkonst.CSI, area.Min.Y+1, area.Max.Y, termkonst.RegionRows)
	if s.Margins {
		fmt.Fprintf(&s.out, "%s%d;%d%s", termkonst.CSI, area.Min.X+1, area.Max.X, termkonst.RegionColumns)
	}
	way, count := termkonst.ScrollDown, lines
	if lines < 0 {
		way, count = termkonst.ScrollUp, -lines
	}
	fmt.Fprintf(&s.out, "%s%d%s", termkonst.CSI, count, way)
	if s.Margins {
		s.out.WriteString(termkonst.MarginsOff)
	}
	s.out.WriteString(termkonst.RegionReset)
	s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
}

func (s *Screen) lose(area image.Rectangle, lines int) {
	top, bottom := int32(area.Min.Y), int32(area.Max.Y-1)
	by := int32(lines)
	for id, row := range s.markers {
		switch {
		case row < top || row > bottom:
		case by < 0 && row < top-by, by > 0 && row > bottom-by:
			s.markers[id] = -1
		default:
			s.markers[id] = row + by
		}
	}
	for i, id := range s.owners {
		if id != 0 && s.markers[id] < 0 {
			s.owners[i] = 0
			s.shown.Row(i / s.cols)[i%s.cols] = buffer.Cell{Grapheme: "\x00"}
			s.touch(image.Rect(i%s.cols, i/s.cols, i%s.cols+1, i/s.cols+1))
		}
	}
}

func (s *Screen) compact() {
	renumbered, markers := map[int32]int32{}, []int32{-1}
	for i, id := range s.owners {
		if id == 0 {
			continue
		}
		n, seen := renumbered[id]
		if !seen {
			n, renumbered[id], markers = int32(len(markers)), int32(len(markers)), append(markers, s.markers[id])
		}
		s.owners[i] = n
	}
	s.markers = markers
}

func (s *Screen) unshifted(l *scene.Layer, b *scene.Box, dy int, area image.Rectangle) {
	v := b.Visual.Add(l.Origin).Intersect(l.Clip)
	c := s.look(b)
	s.rasterise(0, nil, nil)
	band := c.uniform
	top, bottom := max(v.Min.Y, b.Visual.Min.Y+l.Origin.Y+band[0]), min(v.Max.Y, b.Visual.Min.Y+l.Origin.Y+band[1])
	outer := image.Rect(v.Min.X, min(v.Min.Y, v.Min.Y+dy), v.Max.X, max(v.Max.Y, v.Max.Y+dy))
	inner := image.Rect(v.Min.X, max(top, top+dy), v.Max.X, min(bottom, bottom+dy))
	if inner.Empty() {
		s.damage(outer.Intersect(area))
		return
	}
	s.damage(image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, inner.Min.Y).Intersect(area))
	s.damage(image.Rect(outer.Min.X, inner.Max.Y, outer.Max.X, outer.Max.Y).Intersect(area))
}

func (s *Screen) splice(c *column, y int, from int32, lo, hi int) {
	pair := [2]int32{c.lineOf[y], from}
	k, ok := s.splices[pair]
	if !ok {
		s.blank = append(s.blank[:0], run{End: int32(c.width)})
		src := s.blank
		if from >= 0 {
			src = c.store[from]
		}
		s.spliced = compose(s.spliced[:0], c.line(y), int32(lo), int32(hi), src, 0, copyBlend, 0)
		k = pair[0]
		if !slices.Equal(s.spliced, c.line(y)) {
			k = c.intern(s.spliced, s.digest(&s.key, s.spliced))
		}
	}
	s.splices[pair] = k
	c.link(c.lineOf, y, k)
}

func slide(lo, hi, by int, move func(dst, src int), blank func(dst int)) {
	for i := range hi - lo {
		dst := lo + i
		if by > 0 {
			dst = hi - 1 - i
		}
		if src := dst - by; src >= lo && src < hi {
			move(dst, src)
		} else {
			blank(dst)
		}
	}
}

func (s *Screen) scrollbars(root *scene.Node) {
	s.bars = s.bars[:0]
	if !s.painter.Scrolls() {
		return
	}
	s.walker.Walk(root, s.cover, func(n *scene.Node, inside func()) {
		inside()
		s.place(n)
	})
	for _, b := range s.bars {
		s.overlay(b.at.X, b.at.Y, b.cell)
	}
}

func (s *Screen) cover(n *scene.Node) {
	if len(s.bars) == 0 || n.Background.Kind != color.Literal || n.Background.RGBA.A != math.MaxUint8 {
		return
	}
	b, c := n.Bounds, n.Clip
	r := image.Rect(b.X, b.Y, b.X+b.W, b.Y+b.H).Intersect(image.Rect(c.X, c.Y, c.X+c.W, c.Y+c.H))
	s.bars = slices.DeleteFunc(s.bars, func(under bar) bool { return under.at.In(r) })
}

func (s *Screen) place(n *scene.Node) {
	from, to, ok := n.Thumb(konst.ThumbEighths)
	if !ok {
		return
	}
	x, clip := n.Padding.X+n.Padding.W-1, n.Clip
	for y := n.Padding.Y + from/konst.ThumbEighths; (y-n.Padding.Y)*konst.ThumbEighths < to; y++ {
		if x < clip.X || x >= clip.X+clip.W || y < clip.Y || y >= clip.Y+clip.H {
			continue
		}
		top := max(from-(y-n.Padding.Y)*konst.ThumbEighths, 0)
		bottom := min(to-(y-n.Padding.Y)*konst.ThumbEighths, konst.ThumbEighths)
		under := s.want.At(x, y).Bg
		c := buffer.Cell{Grapheme: block(konst.ThumbEighths - top), Fg: thumb(n.Foreground, under), Bg: under}
		if bottom < konst.ThumbEighths {
			c.Grapheme, c.Attr = block(konst.ThumbEighths-bottom), buffer.Inverse
		}
		s.bars = append(s.bars, bar{image.Pt(x, y), c})
	}
}

func block(eighths int) string { return string(konst.LowerEighth + rune(eighths-1)) }

func thumb(fg, bg color.Color) color.Color {
	if fg.Kind != color.Literal || bg.Kind != color.Literal {
		return fg
	}
	mix := func(f, b uint8) uint8 { return uint8(float64(b) + (float64(f)-float64(b))*konst.ThumbAlpha) }
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: mix(fg.RGBA.R, bg.RGBA.R), G: mix(fg.RGBA.G, bg.RGBA.G), B: mix(fg.RGBA.B, bg.RGBA.B), A: bg.RGBA.A}}
}
