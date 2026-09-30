package present

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"image"
	"math"
	"slices"
	"sync"

	colorkonst "github.com/twind-dev/twind/internal/konst/color"
	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	scenekonst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/terminal"
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
	recipes []recipe
	arena   []byte
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
	return graphics.Placement{Col: cells.Min.X, Row: cells.Min.Y, Cols: cells.Dx(), Rows: cells.Dy()}
}

func (w *worker) encode(s *Screen, t int) {
	lo := len(w.out)
	switch s.Graphics {
	case terminal.GraphicsSixel:
		if w.sixel == nil {
			w.sixel = &graphics.Sixel{}
		}
		w.out = w.sixel.Encode(w.out, w.lines, s.placement(t))
	case terminal.GraphicsITerm2:
		if w.iterm == nil {
			w.iterm = &graphics.ITerm{}
		}
		w.pix, w.rows = expand(w.pix, w.rows, w.lines)
		w.out = w.iterm.Encode(w.out, w.rows, s.placement(t))
	case terminal.GraphicsKitty, terminal.GraphicsNone:
		panic(fmt.Sprintf("present: a tile encoded for graphics %d", s.Graphics))
	default:
		panic(fmt.Sprintf("present: unknown graphics %d", s.Graphics))
	}
	s.pieces[t] = piece{w, lo, len(w.out)}
}

func (s *Screen) hash(t int) uint64 {
	r, c := s.lines(t)
	h := uint64(scenekonst.HashSeed)
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
	if s.sampled[i] {
		return s.samples[i]
	}
	r, c, at := s.area(x, y)
	s.bands = s.bands[:0]
	for py := r.Min.Y; py < r.Max.Y; py++ {
		s.bands = banded(s.bands, c.line(py))
	}
	s.sums = cellSums(s.sums, s.bands, c.width/s.Cell.X, s.Cell.X)
	s.sampled[i], s.samples[i] = true, s.average(pixel(s.bands[0].runs, at), s.sums[at/s.Cell.X], s.plain[s.tileAt(x, y)])
	return s.samples[i]
}

func (w *worker) measure(s *Screen, t, twin int) {
	cells, shift := s.tiles[t], 0
	if twin >= 0 {
		shift = (s.tiles[twin].Min.Y - cells.Min.Y) * s.cols
	}
	painted, grounded := s.underText() && s.painted.Load(), s.underText() && !s.plain[t] && s.hashes[t] != s.sent[t]
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		clear(s.sampled[y*s.cols+cells.Min.X : y*s.cols+cells.Max.X])
		if !painted && !grounded {
			continue
		}
		w.bands = w.bands[:0]
		for _, sp := range w.spans[(y-cells.Min.Y)*s.Cell.Y:][:s.Cell.Y] {
			w.bands = banded(w.bands, w.store[sp[0]:sp[1]])
		}
		var last [4]int
		colour, summed := s.average(0, last, false), false
		for x := cells.Min.X; x < cells.Max.X; x++ {
			i := y*s.cols + x
			switch {
			case !grounded && blank(s.text.At(x, y)):
				continue
			case twin >= 0 && s.sampled[i+shift]:
				s.sampled[i], s.samples[i] = true, s.samples[i+shift]
				continue
			case s.plain[t]:
				s.sampled[i], s.samples[i] = true, s.average(pixel(w.bands[0].runs, (x-cells.Min.X)*s.Cell.X), last, true)
				continue
			case !summed:
				w.sums, summed = cellSums(w.sums, w.bands, cells.Dx(), s.Cell.X), true
			}
			if sum := w.sums[x-cells.Min.X]; sum != last {
				last, colour = sum, s.average(0, sum, false)
			}
			s.sampled[i], s.samples[i] = true, colour
		}
	}
}

func cellSums(dst [][4]int, bands []band, cells, width int) [][4]int {
	dst = slices.Grow(dst[:0], cells)[:cells]
	clear(dst)
	for _, b := range bands {
		for x, i := 0, 0; x < cells*width; {
			r := b.runs[i]
			end := min(int(r.End), (x/width+1)*width)
			sum := &dst[x/width]
			for ch := range sum {
				sum[ch] += b.n * (end - x) * int(r.Pixel>>(8*ch)&math.MaxUint8)
			}
			if x = end; x == int(r.End) {
				i++
			}
		}
	}
	return dst
}

type band struct {
	runs []run
	n    int
}

func banded(bands []band, runs []run) []band {
	if n := len(bands); n > 0 && &bands[n-1].runs[0] == &runs[0] {
		bands[n-1].n++
		return bands
	}
	return append(bands, band{runs, 1})
}

func pixel(runs []run, x int) uint32 { return runs[find(runs, x)].Pixel }

func (s *Screen) average(first uint32, sum [4]int, plain bool) color.Color {
	n := s.Cell.X * s.Cell.Y
	switch {
	case plain && first>>24 > 0:
		return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: uint8(first), G: uint8(first >> 8), B: uint8(first >> 16), A: math.MaxUint8}}
	case plain || (sum[3]+n/2)/n == 0:
		return color.Color{}
	}
	unmul := func(v int) uint8 { return uint8((v*math.MaxUint8 + sum[3]/2) / sum[3]) }
	m := color.RGBA{R: unmul(sum[0]), G: unmul(sum[1]), B: unmul(sum[2]), A: uint8(max(1, sum[3]/n))}
	if s.Graphics == terminal.GraphicsSixel {
		m.R, m.G, m.B = register(m.R), register(m.G), register(m.B)
	}
	return color.Color{Kind: color.Literal, RGBA: m}
}

func (s *Screen) flat(x, y int) bool {
	r, c, at := s.area(x, y)
	first := pixel(c.line(r.Min.Y), at)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		runs := c.line(py)
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
