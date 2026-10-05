package present

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"image"
	"math"
	"slices"
	"strconv"
	"sync"

	"github.com/pehcastro/twind/internal/graphics"
	colorkonst "github.com/pehcastro/twind/internal/konst/color"
	graphicskonst "github.com/pehcastro/twind/internal/konst/graphics"
	konst "github.com/pehcastro/twind/internal/konst/paint"
	scenekonst "github.com/pehcastro/twind/internal/konst/scene"
	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

type column struct {
	lock    sync.Mutex
	width   int
	lineOf  []int32
	baseOf  []int32
	store   [][]run
	spare   []run
	hash    []uint64
	holders []int32
	free    []int32
	memo    [1 << graphicskonst.LineMemoBits]int32
}

func (c *column) line(y int) []run { return c.store[c.lineOf[y]] }

func (c *column) newLine(n int) int32 {
	k := int32(len(c.store))
	if f := len(c.free); f > 0 {
		k, c.free = c.free[f-1], c.free[:f-1]
	} else {
		if more := max(len(c.store), graphicskonst.LineSlots); len(c.store) == cap(c.store) {
			c.store, c.hash, c.holders = slices.Grow(c.store, more), slices.Grow(c.hash, more), slices.Grow(c.holders, more)
		}
		c.store, c.hash, c.holders = append(c.store, nil), append(c.hash, 0), append(c.holders, 0)
	}
	if cap(c.store[k]) < n {
		if len(c.spare) < n {
			c.spare = make([]run, max(n, graphicskonst.RunChunk))
		}
		c.store[k], c.spare = c.spare[:n:n], c.spare[n:]
	}
	c.store[k] = c.store[k][:n]
	return k
}

func (c *column) link(of []int32, y int, k int32) {
	if k >= 0 {
		c.holders[k]++
	}
	if old := of[y]; old >= 0 {
		if c.holders[old]--; c.holders[old] == 0 {
			c.free = append(c.free, old)
		}
	}
	of[y] = k
}

func (c *column) shift(top, bottom, by int, blank int32) {
	n := max(by, -by)
	gone, open := bottom-n, top
	if by < 0 {
		gone, open = top, bottom-n
	}
	c.holders[blank] += int32(n)
	for _, k := range c.lineOf[gone : gone+n] {
		if k < 0 {
			continue
		}
		if c.holders[k]--; c.holders[k] == 0 {
			c.free = append(c.free, k)
		}
	}
	if by > 0 {
		copy(c.lineOf[top+by:bottom], c.lineOf[top:bottom-by])
	} else {
		copy(c.lineOf[top:bottom+by], c.lineOf[top-by:bottom])
	}
	for y := open; y < open+n; y++ {
		c.lineOf[y] = blank
	}
}

func (c *column) intern(runs []run, h uint64) int32 {
	slot := &c.memo[h>>(64-graphicskonst.LineMemoBits)]
	if k := *slot - 1; k >= 0 && c.holders[k] > 0 && c.hash[k] == h && slices.Equal(c.store[k], runs) {
		return k
	}
	k := c.newLine(len(runs))
	copy(c.store[k], runs)
	c.hash[k], *slot = h, k+1
	return k
}

func (s *Screen) digest(scratch *[]byte, runs []run) uint64 {
	key := (*scratch)[:0]
	for _, r := range runs {
		key = binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(key, uint32(r.End)), r.Pixel)
	}
	if cap(key) != cap(*scratch) {
		*scratch = key
	}
	return maphash.Bytes(s.seed, key)
}

func (s *Screen) column(x int) *column { return &s.columns[x/(konst.TileColumns*s.Cell.X)] }

func (s *Screen) lines(t int) (image.Rectangle, *column) {
	r := s.pixels(s.tiles[t])
	return r, s.column(r.Min.X)
}

func (s *Screen) tileLines(t int, dst [][]run) [][]run {
	r, c := s.lines(t)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst = append(dst, c.line(y))
	}
	return dst
}

func expand(pix []uint8, rows [][]byte, lines [][]run) ([]uint8, [][]byte) {
	width, distinct := 4*int(lines[0][len(lines[0])-1].End), 0
	for i := range lines {
		if i == 0 || &lines[i][0] != &lines[i-1][0] {
			distinct++
		}
	}
	pix, rows = slices.Grow(pix[:0], distinct*width), rows[:0]
	for i, line := range lines {
		if i > 0 && &line[0] == &lines[i-1][0] {
			rows = append(rows, rows[i-1])
			continue
		}
		row := pix[len(pix) : len(pix)+width]
		pix = pix[:len(pix)+width]
		x := 0
		for _, r := range line {
			seg := row[4*x : 4*r.End]
			binary.LittleEndian.PutUint32(seg, r.Pixel)
			for n := 4; n < len(seg); n *= 2 {
				copy(seg[n:], seg[:n])
			}
			x = int(r.End)
		}
		rows = append(rows, row)
	}
	return pix, rows
}

func (s *Screen) placement(t int) graphics.Placement {
	cells := s.tiles[t]
	if c := s.covers[t]; c.X*c.Y > 1 {
		cells = cells.Union(s.tiles[t-(c.Y-1)*len(s.columns)+c.X-1])
	}
	return graphics.Placement{Col: cells.Min.X, Row: cells.Min.Y, Cols: cells.Dx(), Rows: cells.Dy()}
}

func (s *Screen) join() {
	across := len(s.columns)
	for t := range s.twins {
		s.twins[t], s.leads[t] = t, -1
	}
	for band := len(s.tiles)/across - 1; band >= 0; band-- {
		row, sent := s.send[band*across:(band+1)*across], -1
		for i, send := range row {
			if !send {
				continue
			}
			if sent >= 0 && i-sent-1 <= konst.BridgedTiles {
				for k := sent + 1; k < i; k++ {
					row[k] = true
				}
			}
			sent = i
		}
		for col := 0; col < across; {
			t, n := band*across+col, 1
			if !s.send[t] {
				col++
				continue
			}
			for col+n < across && s.send[t+n] {
				n++
			}
			lead := t
			if up := t + across; up < len(s.tiles) && s.leads[up] >= 0 {
				if f := s.leads[up]; f%across == col && s.covers[f].X == n && f/across-s.covers[f].Y == band {
					lead = f
					s.covers[f].Y++
				}
			}
			for i := t; i < t+n; i++ {
				s.leads[i], s.covers[i] = lead, image.Point{}
			}
			if lead == t {
				s.covers[t] = image.Pt(n, 1)
			}
			col += n
		}
	}
}

func (w *worker) joinedLines(s *Screen, t int) [][]run {
	cover := s.covers[t]
	top, _ := s.lines(t)
	bottom, _ := s.lines(t - (cover.Y-1)*len(s.columns))
	w.joined, w.cuts, w.lines = w.joined[:0], w.cuts[:0], w.lines[:0]
	for y := top.Min.Y; y < bottom.Max.Y; y++ {
		repeated := y > top.Min.Y
		for i := t; i < t+cover.X && repeated; i++ {
			_, c := s.lines(i)
			repeated = c.lineOf[y] == c.lineOf[y-1]
		}
		if repeated {
			w.cuts = append(w.cuts, -1)
			continue
		}
		start, shift := len(w.joined), int32(0)
		for i := t; i < t+cover.X; i++ {
			_, c := s.lines(i)
			for _, part := range c.line(y) {
				if n := len(w.joined); n > start && w.joined[n-1].Pixel == part.Pixel {
					w.joined[n-1].End = part.End + shift
					continue
				}
				w.joined = append(w.joined, run{End: part.End + shift, Pixel: part.Pixel})
			}
			shift += int32(c.width)
		}
		w.cuts = append(w.cuts, len(w.joined))
	}
	from := 0
	for _, cut := range w.cuts {
		if cut < 0 {
			w.lines = append(w.lines, w.lines[len(w.lines)-1])
			continue
		}
		w.lines, from = append(w.lines, w.joined[from:cut:cut]), cut
	}
	return w.lines
}

func (w *worker) encode(s *Screen, t int) {
	lo, key, single := len(w.out), twin{s.hashes[t], s.tiles[t].Size()}, s.covers[t] == image.Pt(1, 1) && !s.fresh
	if single {
		s.lock.Lock()
		body, ok := s.stored.image(key, s.frame)
		s.lock.Unlock()
		if ok {
			if len(body) > 0 {
				w.out = append(cursor(w.out, s.tiles[t].Min), body...)
			}
			s.pieces[t] = piece{w, lo, len(w.out)}
			return
		}
	}
	s.encoded.Add(1)
	switch s.Graphics {
	case terminal.GraphicsSixel:
		if w.sixel == nil {
			w.sixel = &graphics.Sixel{}
		}
		if c := s.covers[t]; c.X*c.Y > 1 {
			if w.out = w.sixel.Encode(w.out, w.joinedLines(s, t), s.placement(t)); !w.sixel.Quantised() {
				s.pieces[t] = piece{w, lo, len(w.out)}
				return
			}
			w.out = w.out[:lo]
			split := image.Pt(c.X, 1)
			if c.Y == 1 {
				split = image.Pt(1, 1)
			}
			for k := range c.Y {
				for i := 0; i < c.X; i += split.X {
					s.covers[t-k*len(s.columns)+i] = split
					w.encode(s, t-k*len(s.columns)+i)
				}
			}
			return
		}
		w.lines = s.tileLines(t, w.lines[:0])
		w.out = w.sixel.Encode(w.out, w.lines, s.placement(t))
	case terminal.GraphicsITerm2:
		if w.iterm == nil {
			w.iterm = &graphics.ITerm{}
		}
		w.lines = s.tileLines(t, w.lines[:0])
		w.pix, w.rows = expand(w.pix, w.rows, w.lines)
		w.out = w.iterm.Encode(w.out, w.rows, s.placement(t))
	case terminal.GraphicsKitty, terminal.GraphicsNone, terminal.GraphicsGDI:
		panic(fmt.Sprintf("present: a tile encoded for graphics %d", s.Graphics))
	default:
		panic(fmt.Sprintf("present: unknown graphics %d", s.Graphics))
	}
	if single {
		s.lock.Lock()
		s.stored.remember(key, w.out[lo+bytes.IndexByte(w.out[lo:], 'H')+1:], s.frame)
		s.lock.Unlock()
	}
	s.pieces[t] = piece{w, lo, len(w.out)}
}

func cursor(dst []byte, at image.Point) []byte {
	dst = strconv.AppendInt(append(dst, termkonst.CSI...), int64(at.Y+1), 10)
	return append(strconv.AppendInt(append(dst, ';'), int64(at.X+1), 10), 'H')
}

func (s *Screen) hash(t int) uint64 {
	r, c := s.lines(t)
	h := uint64(scenekonst.HashSeed) ^ uint64(s.page)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		h = (h ^ c.hash[c.lineOf[y]]) * scenekonst.HashPrime
	}
	return h
}

func (s *Screen) area(x, y int) (r image.Rectangle, c *column, at int) {
	r = s.pixels(image.Rect(x, y, x+1, y+1))
	return r, s.column(r.Min.X), r.Min.X % (konst.TileColumns * s.Cell.X)
}

func (s *Screen) sample(x, y int) color.Color {
	i := y*s.cols + x
	if !s.sampled[i] {
		s.sampled[i], s.samples[i] = true, s.mean(x, y, 0)
	}
	return s.samples[i]
}

func (s *Screen) behind(x, y int) color.Color {
	switch {
	case s.Graphics == terminal.GraphicsSixel && s.edgedCell(x, y):
		return s.mean(x, y, (s.Cell.Y-1)/2)
	case s.edged(x, y):
		return s.mean(x, y, s.inset())
	}
	return s.sample(x, y)
}

func (s *Screen) edged(x, y int) bool {
	of, top := s.columns[x/konst.TileColumns].lineOf, y*s.Cell.Y
	middle := of[top+s.Cell.Y/2]
	return of[top] != middle || of[top+s.Cell.Y-1] != middle
}

func (s *Screen) mean(x, y, margin int) color.Color {
	r, c, at := s.area(x, y)
	r.Min.Y, r.Max.Y = r.Min.Y+margin, r.Max.Y-margin
	s.bands, s.sampling = s.bands[:0], s.sampling[:0]
	for py := r.Min.Y; py < r.Max.Y; py++ {
		if py == r.Min.Y || c.lineOf[py] != c.lineOf[py-1] {
			s.sampling = append(s.sampling, c.line(py)...)
		}
		s.bands = banded(s.bands, [2]int32{int32(len(s.sampling) - len(c.line(py))), int32(len(s.sampling))})
	}
	s.sums = cellSums(s.sums, s.bands, s.sampling, c.width/s.Cell.X, s.Cell.X)
	return s.average(pixel(s.sampling[:s.bands[0].span[1]], at), s.sums[at/s.Cell.X], s.plain[s.tileAt(x, y)], r.Dy())
}

func (w *worker) measure(s *Screen, t, twin int) {
	cells, shift := s.tiles[t], 0
	if twin >= 0 {
		shift = (s.tiles[twin].Min.Y-cells.Min.Y)*s.cols + s.tiles[twin].Min.X - cells.Min.X
	}
	painted, grounded := s.underText() && s.painted.Load(), s.underText() && !s.plain[t] && s.hashes[t] != s.sent[t] || s.Graphics == terminal.GraphicsGDI
	if grounded && w.entry != nil && w.entry.sampled(s, cells) {
		return
	}
	bands, sums := w.bands, w.sums
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		clear(s.sampled[y*s.cols+cells.Min.X : y*s.cols+cells.Max.X])
		if !painted && !grounded {
			continue
		}
		spans := w.spans[(y-cells.Min.Y)*s.Cell.Y:][:s.Cell.Y]
		var last [4]int
		colour, summed := s.average(0, last, false, s.Cell.Y), false
		for x := cells.Min.X; x < cells.Max.X; x++ {
			i := y*s.cols + x
			switch {
			case !grounded && blank(s.text.At(x, y)):
				continue
			case twin >= 0 && s.sampled[i+shift]:
				s.sampled[i], s.samples[i] = true, s.samples[i+shift]
				continue
			case w.lazy:
				w.unpack(s, t)
			}
			switch {
			case s.plain[t]:
				s.sampled[i], s.samples[i] = true, s.average(pixel(w.store[spans[0][0]:spans[0][1]], (x-cells.Min.X)*s.Cell.X), last, true, s.Cell.Y)
				continue
			case !summed:
				bands = bands[:0]
				for _, sp := range spans {
					bands = banded(bands, sp)
				}
				sums, summed = cellSums(sums, bands, w.store, cells.Dx(), s.Cell.X), true
			}
			if sum := sums[x-cells.Min.X]; sum != last {
				last, colour = sum, s.average(0, sum, false, s.Cell.Y)
			}
			s.sampled[i], s.samples[i] = true, colour
		}
	}
	w.bands, w.sums = bands, sums
	if grounded && w.entry != nil {
		w.entry.sample(s, cells)
	}
}

func (w *worker) unpack(s *Screen, t int) {
	r, c := s.lines(t)
	store, of := w.store[:0], c.lineOf[r.Min.Y:r.Max.Y]
	c.lock.Lock()
	for i, k := range of {
		if i > 0 && k == of[i-1] {
			w.spans[i] = w.spans[i-1]
			continue
		}
		w.spans[i] = [2]int32{int32(len(store)), int32(len(store) + len(c.store[k]))}
		store = append(store, c.store[k]...)
	}
	c.lock.Unlock()
	w.keep(store)
	w.lazy = false
}

func cellSums(dst [][4]int, bands []band, store []run, cells, width int) [][4]int {
	dst = slices.Grow(dst[:0], cells)[:cells]
	clear(dst)
	for _, b := range bands {
		runs := store[b.span[0]:b.span[1]]
		for x, i, cell := 0, 0, 0; cell < cells; {
			r, edge := runs[i], (cell+1)*width
			end := min(int(r.End), edge)
			n, sum := b.n*(end-x), &dst[cell]
			sum[0] += n * int(r.Pixel&math.MaxUint8)
			sum[1] += n * int(r.Pixel>>8&math.MaxUint8)
			sum[2] += n * int(r.Pixel>>16&math.MaxUint8)
			sum[3] += n * int(r.Pixel>>24)
			if x = end; x == int(r.End) {
				i++
			}
			if x == edge {
				cell++
			}
		}
	}
	return dst
}

type band struct {
	span [2]int32
	n    int
}

func banded(bands []band, span [2]int32) []band {
	if n := len(bands); n > 0 && bands[n-1].span == span {
		bands[n-1].n++
		return bands
	}
	return append(bands, band{span, 1})
}

func pixel(runs []run, x int) uint32 { return runs[find(runs, x)].Pixel }

func (s *Screen) average(first uint32, sum [4]int, plain bool, rows int) color.Color {
	n := s.Cell.X * rows
	switch {
	case plain && first>>24 > 0:
		return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: uint8(first), G: uint8(first >> 8), B: uint8(first >> 16), A: math.MaxUint8}}
	case plain || sum[3] == 0 || s.page == 0 && (sum[3]+n/2)/n == 0:
		return s.pageBg
	}
	coverage := uint8(max(1, sum[3]/n))
	if rest := n*math.MaxUint8 - sum[3]; s.page != 0 {
		for i := range 3 {
			sum[i] += int(s.page>>(8*i)&math.MaxUint8) * rest / math.MaxUint8
		}
		sum[3] += rest
	}
	unmul := func(v int) uint8 { return uint8((v*math.MaxUint8 + sum[3]/2) / sum[3]) }
	m := color.RGBA{R: unmul(sum[0]), G: unmul(sum[1]), B: unmul(sum[2]), A: coverage}
	if s.Graphics == terminal.GraphicsSixel {
		m.R, m.G, m.B = register(m.R), register(m.G), register(m.B)
	}
	return color.Color{Kind: color.Literal, RGBA: m}
}

func (s *Screen) inset() int {
	return int(math.Round(konst.OneRowInsetCell * float64(s.Cell.Y)))
}

func (s *Screen) flat(x, y, margin int) bool {
	r, c, at := s.area(x, y)
	first, seen := pixel(c.line(r.Min.Y+margin), at), int32(-1)
	for py := r.Min.Y + margin; py < r.Max.Y-margin; py++ {
		if c.lineOf[py] == seen {
			continue
		}
		seen = c.lineOf[py]
		runs := c.store[seen]
		if i := find(runs, at); runs[i].Pixel != first || int(runs[i].End) < at+s.Cell.X {
			return false
		}
	}
	return true
}

func (s *Screen) plainTile(lines [][]run) bool {
	for y := 0; y < len(lines); y += s.Cell.Y {
		top := lines[y]
		for at := 0; at < int(top[len(top)-1].End); at += s.Cell.X {
			i := find(top, at)
			if a := top[i].Pixel >> 24; a != 0 && a != math.MaxUint8 || int(top[i].End) < at+s.Cell.X {
				return false
			}
		}
		for _, line := range lines[y+1 : y+s.Cell.Y] {
			if &line[0] != &top[0] && !slices.Equal(line, top) {
				return false
			}
		}
	}
	return true
}

func quantise(runs []run) []run {
	out := runs[:0]
	for _, r := range runs {
		if r.Pixel>>24 == math.MaxUint8 {
			rgb := palette(color.RGBA{R: uint8(r.Pixel), G: uint8(r.Pixel >> 8), B: uint8(r.Pixel >> 16)}.ANSI256())
			r.Pixel = uint32(rgb[0]) | uint32(rgb[1])<<8 | uint32(rgb[2])<<16 | math.MaxUint8<<24
		}
		out = add(out, r.End, r.Pixel)
	}
	return out
}

func unpaged(runs []run, page uint32) []run {
	out := runs[:0]
	for _, r := range runs {
		if r.Pixel == page {
			r.Pixel = 0
		}
		out = add(out, r.End, r.Pixel)
	}
	return out
}

func palette(index uint8) [3]uint8 {
	if index >= colorkonst.GreyBase {
		v := uint8(colorkonst.GreyFirstLevel + int(index-colorkonst.GreyBase)*colorkonst.GreyLevelStep)
		return [3]uint8{v, v, v}
	}
	cube := int(index - colorkonst.CubeBase)
	var rgb [3]uint8
	for c := 2; c >= 0; c-- {
		if n := cube % colorkonst.CubeSide; n > 0 {
			rgb[c] = uint8(colorkonst.CubeFirstLevel + (n-1)*colorkonst.CubeLevelStep)
		}
		cube /= colorkonst.CubeSide
	}
	return rgb
}

func register(v uint8) uint8 {
	percent := (int(v)*graphicskonst.SixelPercent + math.MaxUint8/2) / math.MaxUint8
	return uint8((percent*math.MaxUint8 + graphicskonst.SixelPercent/2) / graphicskonst.SixelPercent)
}
