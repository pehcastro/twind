package layout

import (
	"fmt"
	"math"

	konst "github.com/pehcastro/twind/internal/konst/layout"
)

type TrackSize uint8

const (
	SizeAuto TrackSize = iota
	SizeCells
	SizePercent
	SizeFr
	SizeMinContent
	SizeMaxContent
)

type Breadth struct {
	Kind  TrackSize
	Value int
}

type Track struct{ Min, Max Breadth }

type Line struct{ Index, Span int }

type Placement struct{ Start, End Line }

type Flow uint8

const (
	FlowRow Flow = iota
	FlowColumn
	FlowRowDense
	FlowColumnDense
)

const (
	across = 0
	down   = 1
)

const infinite = math.MaxInt

type span struct{ start, end int }

type gridItem struct {
	box              *Box
	area             [2]span
	definite         [2]bool
	least, low, high int
	width            int
}

type track struct {
	min, max    Breadth
	base, limit int
	pos         int
	frozen      bool
}

func (b Breadth) contentBased() bool {
	switch b.Kind {
	case SizeAuto, SizeMinContent, SizeMaxContent:
		return true
	case SizeCells, SizePercent, SizeFr:
		return false
	}
	panic(fmt.Sprintf("layout: unknown track size %d", b.Kind))
}

func (a *arena) arrangeGrid(b *Box, innerW, innerH int, mode heightMode) int {
	ints, tracks, cells := len(a.ints), len(a.tracks), len(a.cells)
	s := &b.Style
	items, counts := a.placeItems(b)
	cols := a.initTracks(s.Columns, s.AutoColumns, counts[across], innerW, true)
	a.contributeWidths(items, cols)
	a.sizeTracks(cols, items, across, s.ColumnGap, innerW, true, maxContent)
	a.position(cols, s.ColumnGap, innerW, true, s.Justify)
	fixed := mode == fixedHeight
	rows := a.initTracks(s.Rows, s.AutoRows, counts[down], innerH, fixed)
	for i := range items {
		it := &items[i]
		c := it.box
		cs, m := &c.Style, c.Style.Margin
		areaW := extent(cols, it.area[across])
		avail := areaW - m.Left - m.Right
		if justifyOf(s, cs) == AlignStretch && cs.Width.Unit == Auto && cs.Aspect == (Ratio{}) {
			it.width = limit(cs.MinWidth, cs.MaxWidth, areaW, true).clamp(avail)
		} else {
			it.width = a.fitWidth(c, avail)
		}
		it.low = a.heightOf(c, it.width, 0, false) + m.Top + m.Bottom
		it.high, it.least = it.low, it.low
		if cs.MinHeight.Unit != Auto || clips(cs.Overflow) || !automaticMinimum(rows, it.area[down]) {
			it.least = limit(cs.MinHeight, cs.MaxHeight, 0, false).min + m.Top + m.Bottom
		}
	}
	a.sizeTracks(rows, items, down, s.RowGap, innerH, fixed, maxContent)
	a.position(rows, s.RowGap, innerH, fixed, s.AlignContent)
	for _, it := range items {
		c := it.box
		cs, m := &c.Style, c.Style.Margin
		areaH := extent(rows, it.area[down])
		x, y := cols[it.area[across].start].pos, rows[it.area[down].start].pos
		al, availH := alignOf(s, cs), areaH-m.Top-m.Bottom
		h := limit(cs.MinHeight, cs.MaxHeight, areaH, true).clamp(availH)
		if _, sized := resolve(cs.Height, areaH, true); sized || al != AlignStretch || cs.Aspect != (Ratio{}) {
			h = a.heightOf(c, it.width, areaH, true)
		}
		availW := extent(cols, it.area[across]) - m.Left - m.Right
		c.frame = Rect{x + m.Left + offset(justifyOf(s, cs), availW-it.width), y + m.Top + whole(offset(al, availH-h), s.RowUnits), it.width, h}
		c.frame.Y += c.nudge(c.frame.Y)
	}
	a.ints, a.tracks, a.cells = a.ints[:ints], a.tracks[:tracks], a.cells[:cells]
	return total(rows, s.RowGap)
}

func (a *arena) gridWidth(b *Box, mode sizing) int {
	ints, tracks, cells := len(a.ints), len(a.tracks), len(a.cells)
	s := &b.Style
	items, counts := a.placeItems(b)
	cols := a.initTracks(s.Columns, s.AutoColumns, counts[across], 0, false)
	a.contributeWidths(items, cols)
	a.sizeTracks(cols, items, across, s.ColumnGap, 0, false, mode)
	w := total(cols, s.ColumnGap)
	a.ints, a.tracks, a.cells = a.ints[:ints], a.tracks[:tracks], a.cells[:cells]
	return w
}

func total(ts []track, gap int) int {
	sum := gap * max(len(ts)-1, 0)
	for _, t := range ts {
		sum += t.base
	}
	return sum
}

func extent(ts []track, sp span) int {
	last := ts[sp.end-1]
	return last.pos + last.base - ts[sp.start].pos
}

func justifyOf(parent, child *Style) Align {
	j := child.JustifySelf
	if j == AlignAuto {
		j = parent.JustifyItems
	}
	if j == AlignAuto {
		return AlignStretch
	}
	return j
}

func automaticMinimum(ts []track, sp span) bool {
	auto, flexible := false, false
	for _, t := range ts[sp.start:sp.end] {
		auto = auto || t.min.Kind == SizeAuto
		flexible = flexible || t.max.Kind == SizeFr
	}
	return auto && (sp.end-sp.start == 1 || !flexible)
}

func (a *arena) contributeWidths(items []gridItem, cols []track) {
	for i := range items {
		it := &items[i]
		c := it.box
		cs := &c.Style
		m := cs.Margin.Left + cs.Margin.Right
		bounds := limit(cs.MinWidth, cs.MaxWidth, 0, false)
		low, high := a.intrinsic(c, minContent), a.intrinsic(c, maxContent)
		w, ok := resolve(cs.Width, 0, false)
		if !ok {
			w, ok = aspectWidth(cs, 0, false)
		}
		least := bounds.min
		if cs.MinWidth.Unit == Auto && !clips(cs.Overflow) && automaticMinimum(cols, it.area[across]) {
			least = min(low, bounds.max)
			if ok {
				least = min(least, w)
			}
		}
		if ok {
			low, high = w, w
		}
		it.least, it.low, it.high = least+m, bounds.clamp(low)+m, bounds.clamp(high)+m
	}
}

func lines(p Placement, explicit int) (span, bool) {
	at := func(l Line) int {
		if l.Index > 0 {
			return l.Index - 1
		}
		return max(explicit+1+l.Index, 0)
	}
	start, end := p.Start, p.End
	switch {
	case start.Index != 0 && end.Index != 0:
		s, e := at(start), at(end)
		if e < s {
			s, e = e, s
		}
		return span{s, max(e, s+1)}, true
	case start.Index != 0:
		s := at(start)
		return span{s, s + max(end.Span, 1)}, true
	case end.Index != 0:
		e := at(end)
		s := max(e-max(start.Span, 1), 0)
		return span{s, max(e, s+1)}, true
	}
	return span{0, max(start.Span, 1)}, false
}

type occupancy struct {
	taken  []int
	stride int
}

func (o occupancy) free(major, minor span) bool {
	for r := major.start; r < major.end; r++ {
		for c := minor.start; c < minor.end; c++ {
			if o.taken[r*o.stride+c] != 0 {
				return false
			}
		}
	}
	return true
}

func (o occupancy) take(major, minor span) {
	for r := major.start; r < major.end; r++ {
		for c := minor.start; c < minor.end; c++ {
			o.taken[r*o.stride+c] = 1
		}
	}
}

func (a *arena) placeItems(b *Box) ([]gridItem, [2]int) {
	s := &b.Style
	major, minor, dense := down, across, false
	switch s.Flow {
	case FlowRow:
	case FlowRowDense:
		dense = true
	case FlowColumn:
		major, minor = across, down
	case FlowColumnDense:
		major, minor, dense = across, down, true
	default:
		panic(fmt.Sprintf("layout: unknown grid flow %d", s.Flow))
	}
	explicit := [2]int{len(s.Columns), len(s.Rows)}
	kids := children(b)
	items := grab(&a.cells, len(kids))[:0]
	width, bound, pushed, stride := explicit[minor], explicit[major], 0, 0
	for _, c := range kids {
		if !visible(c) || !flowing(c.Style.Position) {
			continue
		}
		it := gridItem{box: c}
		it.area[across], it.definite[across] = lines(c.Style.Column, explicit[across])
		it.area[down], it.definite[down] = lines(c.Style.Row, explicit[down])
		width = max(width, it.area[minor].end)
		if it.definite[major] {
			bound = max(bound, it.area[major].end)
		} else {
			pushed += it.area[major].end - it.area[major].start
		}
		if it.definite[major] && !it.definite[minor] {
			stride += it.area[minor].end
		}
		items = append(items, it)
	}
	bound += pushed
	occupied := occupancy{grabZero(&a.ints, (width+stride)*bound), width + stride}
	for _, it := range items {
		if it.definite[major] && it.definite[minor] {
			occupied.take(it.area[major], it.area[minor])
		}
	}
	cursors := grabZero(&a.ints, bound)
	for i := range items {
		it := &items[i]
		if !it.definite[major] || it.definite[minor] {
			continue
		}
		n, line, c := it.area[minor].end, it.area[major].start, 0
		if !dense {
			c = cursors[line]
		}
		for !occupied.free(it.area[major], span{c, c + n}) {
			c++
		}
		it.area[minor], cursors[line] = span{c, c + n}, c+n
		occupied.take(it.area[major], it.area[minor])
		width = max(width, c+n)
	}
	row, col, used := 0, 0, explicit[major]
	for i := range items {
		it := &items[i]
		if it.definite[major] {
			used = max(used, it.area[major].end)
			continue
		}
		if dense {
			row, col = 0, 0
		}
		rows, cols := it.area[major].end, it.area[minor].end-it.area[minor].start
		if it.definite[minor] {
			if it.area[minor].start < col {
				row++
			}
			col = it.area[minor].start
			for !occupied.free(span{row, row + rows}, it.area[minor]) {
				row++
			}
		} else {
			for col+cols > width || !occupied.free(span{row, row + rows}, span{col, col + cols}) {
				col++
				if col+cols > width {
					row, col = row+1, 0
				}
			}
			it.area[minor] = span{col, col + cols}
		}
		it.area[major] = span{row, row + rows}
		occupied.take(it.area[major], it.area[minor])
		used = max(used, row+rows)
	}
	var counts [2]int
	counts[major], counts[minor] = used, width
	return items, counts
}

func (a *arena) initTracks(template, auto []Track, count, size int, definite bool) []track {
	ts := grab(&a.tracks, count)
	for i := range ts {
		var t Track
		switch {
		case i < len(template):
			t = template[i]
		case len(auto) > 0:
			t = auto[(i-len(template))%len(auto)]
		}
		lo, hi := fix(t.Min, size, definite), fix(t.Max, size, definite)
		ts[i] = track{min: lo, max: hi, limit: infinite}
		if lo.Kind == SizeCells {
			ts[i].base = lo.Value
		}
		if hi.Kind == SizeCells {
			ts[i].limit = max(hi.Value, ts[i].base)
		}
	}
	return ts
}

func fix(b Breadth, size int, definite bool) Breadth {
	switch b.Kind {
	case SizePercent:
		if definite {
			return Breadth{SizeCells, b.Value * size / konst.PercentWhole}
		}
		return Breadth{}
	case SizeAuto, SizeCells, SizeFr, SizeMinContent, SizeMaxContent:
		return b
	}
	panic(fmt.Sprintf("layout: unknown track size %d", b.Kind))
}
