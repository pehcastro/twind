package present

import (
	"bytes"
	"encoding/binary"
	"image"
	"math"

	colorkonst "github.com/twind-dev/twind/internal/konst/color"
	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	scenekonst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/terminal"
)

type column struct {
	width   int
	lineOf  []int32
	store   [][]uint8
	spare   []uint8
	hash    []uint64
	holders []int32
	free    []int32
	memo    [1 << graphicskonst.LineMemoBits]int32
}

func (c *column) line(y int) []uint8 { return c.store[c.lineOf[y]] }

func (c *column) newLine() int32 {
	if n := len(c.free); n > 0 {
		k := c.free[n-1]
		c.free = c.free[:n-1]
		return k
	}
	if len(c.spare) == 0 {
		c.spare = make([]uint8, c.width*graphicskonst.LineChunk)
	}
	c.store, c.spare = append(c.store, c.spare[:c.width:c.width]), c.spare[c.width:]
	c.hash, c.holders = append(c.hash, 0), append(c.holders, 0)
	return int32(len(c.holders) - 1)
}

func (c *column) link(y int, k int32) {
	c.holders[k]++
	if old := c.lineOf[y]; old >= 0 {
		if c.holders[old]--; c.holders[old] == 0 {
			c.free = append(c.free, old)
		}
	}
	c.lineOf[y] = k
}

func (c *column) intern(row []uint8, h uint64) int32 {
	slot := &c.memo[h>>(64-graphicskonst.LineMemoBits)]
	if k := *slot - 1; k >= 0 && c.holders[k] > 0 && c.hash[k] == h && bytes.Equal(c.store[k], row) {
		return k
	}
	k := c.newLine()
	copy(c.store[k], row)
	c.hash[k], *slot = h, k+1
	return k
}

func (s *Screen) column(x int) *column { return &s.columns[x/(konst.TileColumns*s.Cell.X)] }

func (s *Screen) lines(t int) (image.Rectangle, *column) {
	r := s.pixels(s.tiles[t])
	return r, s.column(r.Min.X)
}

func (s *Screen) tileLines(t int, dst [][]byte) [][]byte {
	r, c := s.lines(t)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst = append(dst, c.line(y))
	}
	return dst
}

func (s *Screen) placement(t int) graphics.Placement {
	cells := s.tiles[t]
	return graphics.Placement{Col: cells.Min.X, Row: cells.Min.Y, Cols: cells.Dx(), Rows: cells.Dy()}
}

func (w *worker) encode(s *Screen, t int) {
	lo := len(w.out)
	w.rows = s.tileLines(t, w.rows[:0])
	if s.Graphics == terminal.GraphicsSixel {
		w.out = w.sixel.Encode(w.out, w.rows, s.placement(t))
	} else {
		w.out = w.iterm.Encode(w.out, w.rows, s.placement(t))
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

func (s *Screen) opaque(t int) bool {
	r, c := s.lines(t)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y > r.Min.Y && c.lineOf[y] == c.lineOf[y-1] {
			continue
		}
		row := c.line(y)
		for i := 3; i < len(row); i += 4 {
			if row[i] != math.MaxUint8 {
				return false
			}
		}
	}
	return true
}

func (s *Screen) area(x, y int) (r image.Rectangle, c *column, at int) {
	r = s.pixels(image.Rect(x, y, x+1, y+1))
	return r, s.column(r.Min.X), 4 * (r.Min.X % (konst.TileColumns * s.Cell.X))
}

func (s *Screen) sample(x, y int) color.Color {
	i := y*s.cols + x
	if s.sampled[i] {
		return s.samples[i]
	}
	s.sampled[i], s.samples[i] = true, color.Color{}
	r, c, at := s.area(x, y)
	if s.plain[s.tileOf[i]] {
		if p := c.line(r.Min.Y)[at:]; p[3] > 0 {
			s.samples[i] = color.Color{Kind: color.Literal, RGBA: color.RGBA{R: p[0], G: p[1], B: p[2], A: math.MaxUint8}}
		}
		return s.samples[i]
	}
	var sum, part [4]int
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y == r.Min.Y || c.lineOf[y] != c.lineOf[y-1] {
			part = [4]int{}
			for ch, v := range c.line(y)[at : at+4*s.Cell.X] {
				part[ch%4] += int(v)
			}
		}
		for ch := range sum {
			sum[ch] += part[ch]
		}
	}
	if n := s.Cell.X * s.Cell.Y; (sum[3]+n/2)/n == 0 {
		return s.samples[i]
	}
	unmul := func(v int) uint8 { return uint8((v*math.MaxUint8 + sum[3]/2) / sum[3]) }
	m := color.RGBA{R: unmul(sum[0]), G: unmul(sum[1]), B: unmul(sum[2]), A: math.MaxUint8}
	if s.Graphics == terminal.GraphicsSixel {
		m.R, m.G, m.B = register(m.R), register(m.G), register(m.B)
	}
	s.samples[i] = color.Color{Kind: color.Literal, RGBA: m}
	return s.samples[i]
}

func (s *Screen) flat(x, y int) bool {
	r, c, at := s.area(x, y)
	first := binary.LittleEndian.Uint32(c.line(r.Min.Y)[at:])
	for py := r.Min.Y; py < r.Max.Y; py++ {
		if py > r.Min.Y && c.lineOf[py] == c.lineOf[py-1] {
			continue
		}
		row := c.line(py)[at : at+4*s.Cell.X]
		for i := 0; i < len(row); i += 4 {
			if binary.LittleEndian.Uint32(row[i:]) != first {
				return false
			}
		}
	}
	return true
}

func (s *Screen) plainTile(t int) bool {
	px, c := s.lines(t)
	for y := px.Min.Y; y < px.Max.Y; y += s.Cell.Y {
		top := c.line(y)
		for at := 0; at < len(top); at += 4 * s.Cell.X {
			first := binary.LittleEndian.Uint32(top[at:])
			if a := first >> 24; a != 0 && a != math.MaxUint8 {
				return false
			}
			for i := at + 4; i < at+4*s.Cell.X; i += 4 {
				if binary.LittleEndian.Uint32(top[i:]) != first {
					return false
				}
			}
		}
		for py := y + 1; py < y+s.Cell.Y; py++ {
			if !bytes.Equal(c.line(py), top) {
				return false
			}
		}
	}
	return true
}

func quantise(row []uint8) {
	var from, to [3]uint8
	for i := 0; i < len(row); i += 4 {
		if row[i+3] != math.MaxUint8 {
			continue
		}
		if rgb := [3]uint8(row[i : i+3]); rgb != from {
			from, to = rgb, palette(color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2]}.ANSI256())
		}
		copy(row[i:i+3], to[:])
	}
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
