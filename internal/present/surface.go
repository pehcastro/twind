package present

import (
	"bytes"
	"encoding/binary"
	"hash/maphash"
	"image"
	imagecolor "image/color"
	"image/draw"
	"math"
	"slices"

	colorkonst "github.com/twind-dev/twind/internal/konst/color"
	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	scenekonst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func (s *Screen) composite(f *scene.Frame, t image.Rectangle) {
	s.groups, s.parts = s.groups[:0], s.parts[:0]
	if len(s.targets) == 0 {
		s.targets = append(s.targets, nil)
	}
	s.targets[0] = s.surface
	for i := range f.Layers {
		l := &f.Layers[i]
		for len(s.groups) > 0 && s.groups[len(s.groups)-1].layer != l.Parent {
			s.close()
		}
		into := 0
		if n := len(s.groups); n > 0 {
			into = s.groups[n-1].into
		}
		own := into >= 0 && l.Opacity > 0 && l.Opacity < 1
		switch {
		case own:
			into++
			s.parts = append(s.parts, part{step: openGroup, into: into})
		case l.Opacity <= 0:
			into = -1
		}
		s.groups = append(s.groups, group{layer: i, into: into, opacity: l.Opacity, own: own})
		if into < 0 || !l.Visual.Overlaps(t) {
			continue
		}
		for j := range l.Boxes {
			b := &l.Boxes[j]
			at := b.Visual.Add(l.Origin)
			if r := at.Intersect(t).Intersect(l.Clip); !r.Empty() {
				s.parts = append(s.parts, part{step: drawBox, c: s.boxRaster(b), r: r, at: at.Min, into: into})
			}
		}
	}
	for len(s.groups) > 0 {
		s.close()
	}
	for y := t.Min.Y; y < t.Max.Y; y++ {
		row := s.surface.Pix[s.surface.PixOffset(t.Min.X, y):s.surface.PixOffset(t.Max.X, y)]
		if y > t.Min.Y && s.repeats(y) {
			copy(row, s.surface.Pix[s.surface.PixOffset(t.Min.X, y-1):])
			continue
		}
		clear(row)
		u := image.Rect(t.Min.X, y, t.Max.X, y+1)
		for _, p := range s.parts {
			switch p.step {
			case openGroup:
				s.group(p.into, u)
			case drawBox:
				if r := p.r.Intersect(u); !r.Empty() {
					over(s.targets[p.into], r, p.c, r.Min.Sub(p.at))
				}
			case closeGroup:
				alpha := image.NewUniform(imagecolor.Alpha{A: uint8(math.Round(p.opacity * math.MaxUint8))})
				draw.DrawMask(s.targets[p.into-1], u, s.targets[p.into], u.Min, alpha, image.Point{}, draw.Over)
			}
		}
	}
}

func (s *Screen) close() {
	g := s.groups[len(s.groups)-1]
	s.groups = s.groups[:len(s.groups)-1]
	if g.own {
		s.parts = append(s.parts, part{step: closeGroup, into: g.into, opacity: g.opacity})
	}
}

func (s *Screen) repeats(y int) bool {
	for _, p := range s.parts {
		if p.step != drawBox {
			continue
		}
		in, was := y >= p.r.Min.Y && y < p.r.Max.Y, y > p.r.Min.Y && y <= p.r.Max.Y
		if in != was || in && p.c.row[y-p.at.Y] != p.c.row[y-1-p.at.Y] {
			return false
		}
	}
	return true
}

func over(dst *image.RGBA, r image.Rectangle, src *cached, at image.Point) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		d := dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Max.X, y)]
		k := src.row[at.Y+y-r.Min.Y]
		p := src.pix[int(k)*src.stride+4*at.X:][:len(d)]
		run := src.solid[k]
		lo := min(max(run[0]-4*at.X, 0), len(d))
		hi := min(max(run[1]-4*at.X, lo), len(d))
		blend(d[:lo], p[:lo])
		copy(d[lo:hi], p[lo:hi])
		blend(d[hi:], p[hi:])
	}
}

func blend(d, p []uint8) {
	var src, under, out uint32
	for i := 0; i < len(d); i += 4 {
		switch a := uint32(p[i+3]); a {
		case 0:
		case math.MaxUint8:
			copy(d[i:i+4], p[i:i+4])
		default:
			px := d[i : i+4 : i+4]
			if s, u := binary.LittleEndian.Uint32(p[i:]), binary.LittleEndian.Uint32(px); s != src || u != under || out == 0 {
				keep := math.MaxUint8 - a
				for c := range 4 {
					px[c] = p[i+c] + uint8((uint32(px[c])*keep+math.MaxUint8/2)/math.MaxUint8)
				}
				src, under, out = s, u, binary.LittleEndian.Uint32(px)
				continue
			}
			binary.LittleEndian.PutUint32(px, out)
		}
	}
}

func (s *Screen) group(into int, t image.Rectangle) {
	if len(s.targets) == into {
		s.targets = append(s.targets, &image.RGBA{})
	}
	g := s.targets[into]
	n := 4 * t.Dx() * t.Dy()
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], 4*t.Dx(), t
	clear(g.Pix)
}

func (s *Screen) boxRaster(b *scene.Box) *cached {
	if c, ok := s.cache[b.Look]; ok {
		return c
	}
	size := b.Visual.Size()
	s.ops = append(s.ops[:0], b.Ops...)
	for i := range s.ops {
		s.ops[i].Box.X -= float64(b.Visual.Min.X)
		s.ops[i].Box.Y -= float64(b.Visual.Min.Y)
	}
	stride := 4 * size.X
	c := &cached{stride: stride, row: make([]int32, size.Y), frame: s.frame}
	s.plan(size.Y)
	s.lines = s.lines[:0]
	for y := 0; y < size.Y; {
		if !s.drawn[y] {
			c.row[y] = c.row[y-1]
			y++
			continue
		}
		end := y + 1
		for end < size.Y && s.drawn[end] {
			end++
		}
		n := len(s.lines)
		s.lines = slices.Grow(s.lines, stride*(end-y))
		s.canvas.Pix, s.canvas.Stride, s.canvas.Rect = s.lines[n:n+stride*(end-y)], stride, image.Rect(0, y, size.X, end)
		s.raster.Draw(&s.canvas, s.ops, s.canvas.Rect)
		for at := n; y < end; y, at = y+1, at+stride {
			line := s.lines[at : at+stride]
			if n > 0 && bytes.Equal(line, s.lines[n-stride:n]) {
				c.row[y] = c.row[y-1]
				continue
			}
			c.row[y] = int32(len(c.solid))
			c.solid = append(c.solid, solid(line))
			n += copy(s.lines[n:n+stride], line)
		}
		s.lines = s.lines[:n]
	}
	c.pix = slices.Clone(s.lines)
	for y, start := 1, 0; y <= size.Y; y++ {
		if y < size.Y && c.row[y] == c.row[y-1] {
			continue
		}
		if y-start > c.uniform[1]-c.uniform[0] {
			c.uniform = [2]int{start, y}
		}
		start = y
	}
	s.rastered++
	s.cache[b.Look] = c
	return c
}

func (s *Screen) plan(h int) {
	if cap(s.drawn) < h {
		s.drawn = make([]bool, h)
	}
	s.drawn = s.drawn[:h]
	clear(s.drawn)
	for _, op := range s.ops {
		if op.Kind != raster.Fill && op.Kind != raster.Border || len(op.Stops) > 0 || op.Dash != raster.Solid {
			for y := range s.drawn {
				s.drawn[y] = true
			}
			return
		}
	}
	if h > 0 {
		s.drawn[0] = true
	}
	for _, op := range s.ops {
		b, radii, width := op.Box, op.Box.Radii, 0.0
		if op.Kind == raster.Border {
			width = op.Width
		}
		top, bottom := max(radii[0], radii[1], width), max(radii[2], radii[3], width)
		for _, edge := range [2][2]float64{{b.Y, b.Y + top}, {b.Y + b.H - bottom, b.Y + b.H}} {
			for y := max(int(math.Floor(edge[0]))-1, 0); y <= min(int(math.Ceil(edge[1]))+1, h-1); y++ {
				s.drawn[y] = true
			}
		}
	}
}

func solid(row []uint8) [2]int {
	var run [2]int
	start := 0
	for i := 0; i <= len(row); i += 4 {
		if i < len(row) && row[i+3] == math.MaxUint8 {
			continue
		}
		if i-start > run[1]-run[0] {
			run = [2]int{start, i}
		}
		start = i + 4
	}
	return run
}

func (s *Screen) evict(f *scene.Frame) {
	for _, l := range f.Layers {
		for _, b := range l.Boxes {
			if c, ok := s.cache[b.Look]; ok {
				c.frame = s.frame
			}
		}
	}
	for look, c := range s.cache {
		if c.frame != s.frame {
			delete(s.cache, look)
		}
	}
}

func (s *Screen) damage(r image.Rectangle) {
	r = r.Intersect(s.surface.Rect)
	cells := image.Rect(r.Min.X/s.Cell.X, r.Min.Y/s.Cell.Y, (r.Max.X+s.Cell.X-1)/s.Cell.X, (r.Max.Y+s.Cell.Y-1)/s.Cell.Y)
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		for x := cells.Min.X; x < cells.Max.X; x++ {
			s.dirty[s.tileOf[y*s.cols+x]] = true
		}
	}
}

func (s *Screen) pixels(cells image.Rectangle) image.Rectangle {
	return image.Rect(cells.Min.X*s.Cell.X, cells.Min.Y*s.Cell.Y, cells.Max.X*s.Cell.X, cells.Max.Y*s.Cell.Y)
}

func (s *Screen) hash(r image.Rectangle) uint64 {
	h := uint64(scenekonst.HashSeed)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		h = (h ^ maphash.Bytes(s.seed, s.surface.Pix[s.surface.PixOffset(r.Min.X, y):s.surface.PixOffset(r.Max.X, y)])) * scenekonst.HashPrime
	}
	return h
}

func (s *Screen) sample(x, y int) color.Color {
	i := y*s.cols + x
	if s.sampled[i] {
		return s.samples[i]
	}
	s.sampled[i], s.samples[i] = true, color.Color{}
	if s.plain[s.tileOf[i]] {
		if p := s.surface.Pix[s.surface.PixOffset(x*s.Cell.X, y*s.Cell.Y):]; p[3] > 0 {
			s.samples[i] = color.Color{Kind: color.Literal, RGBA: color.RGBA{R: p[0], G: p[1], B: p[2], A: math.MaxUint8}}
		}
		return s.samples[i]
	}
	if m := raster.Mean(s.surface, s.pixels(image.Rect(x, y, x+1, y+1))); m.A > 0 {
		if s.Graphics == terminal.GraphicsSixel {
			m.R, m.G, m.B = register(m.R), register(m.G), register(m.B)
		}
		m.A = math.MaxUint8
		s.samples[i] = color.Color{Kind: color.Literal, RGBA: m}
	}
	return s.samples[i]
}

func (s *Screen) quantise(r image.Rectangle) {
	var from, to [3]uint8
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := s.surface.Pix[s.surface.PixOffset(r.Min.X, y):s.surface.PixOffset(r.Max.X, y)]
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

func (s *Screen) plainTile(cells image.Rectangle) bool {
	px, cell := s.pixels(cells), 4*s.Cell.X
	for y := px.Min.Y; y < px.Max.Y; y += s.Cell.Y {
		top := s.surface.Pix[s.surface.PixOffset(px.Min.X, y):s.surface.PixOffset(px.Max.X, y)]
		for at := 0; at < len(top); at += cell {
			first := binary.LittleEndian.Uint32(top[at:])
			if a := first >> 24; a != 0 && a != math.MaxUint8 {
				return false
			}
			for i := at + 4; i < at+cell; i += 4 {
				if binary.LittleEndian.Uint32(top[i:]) != first {
					return false
				}
			}
		}
		for py := y + 1; py < y+s.Cell.Y; py++ {
			if !bytes.Equal(s.surface.Pix[s.surface.PixOffset(px.Min.X, py):s.surface.PixOffset(px.Max.X, py)], top) {
				return false
			}
		}
	}
	return true
}

func (s *Screen) flat(x, y int) bool {
	r := s.pixels(image.Rect(x, y, x+1, y+1))
	first := s.surface.Pix[s.surface.PixOffset(r.Min.X, r.Min.Y):][:4]
	for py := r.Min.Y; py < r.Max.Y; py++ {
		row := s.surface.Pix[s.surface.PixOffset(r.Min.X, py):s.surface.PixOffset(r.Max.X, py)]
		for i := 0; i < len(row); i += 4 {
			if !bytes.Equal(row[i:i+4], first) {
				return false
			}
		}
	}
	return true
}
