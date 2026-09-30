package present

import (
	"bytes"
	"encoding/binary"
	"hash/maphash"
	"image"
	imagecolor "image/color"
	"image/draw"
	"math"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	rasterkonst "github.com/twind-dev/twind/internal/konst/raster"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
)

type worker struct {
	raster  raster.Raster
	ops     []raster.Op
	canvas  image.RGBA
	drawn   []bool
	line    []uint8
	view    image.RGBA
	targets []*image.RGBA
	rows    [][]byte
	sixel   graphics.Sixel
	iterm   graphics.ITerm
	out     []byte
}

type pending struct {
	box *scene.Box
	c   *cached
}

type piece struct {
	w      *worker
	lo, hi int
}

func (s *Screen) parallel(n int, do func(w *worker, i int)) {
	workers := max(min(n, runtime.GOMAXPROCS(0), graphicskonst.Workers), 1)
	for len(s.workers) < workers {
		s.workers = append(s.workers, &worker{})
	}
	var next atomic.Int64
	run := func(w *worker) {
		for i := int(next.Add(1)) - 1; i < n; i = int(next.Add(1)) - 1 {
			do(w, i)
		}
	}
	var wg sync.WaitGroup
	for _, w := range s.workers[1:workers] {
		wg.Go(func() { run(w) })
	}
	run(s.workers[0])
	wg.Wait()
}

func (s *Screen) collect(f *scene.Frame, t int) {
	r, start := s.pixels(s.tiles[t]), len(s.parts)
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
		if into < 0 || !l.Visual.Overlaps(r) {
			continue
		}
		for j := range l.Boxes {
			b := &l.Boxes[j]
			at := b.Visual.Add(l.Origin)
			if v := at.Intersect(r).Intersect(l.Clip); !v.Empty() {
				s.parts = append(s.parts, part{step: drawBox, c: s.look(b), r: v, at: at.Min, into: into})
			}
		}
	}
	for len(s.groups) > 0 {
		s.close()
	}
	s.spans[t] = [2]int{start, len(s.parts)}
}

func (s *Screen) close() {
	g := s.groups[len(s.groups)-1]
	s.groups = s.groups[:len(s.groups)-1]
	if g.own {
		s.parts = append(s.parts, part{step: closeGroup, into: g.into, opacity: g.opacity})
	}
}

func (w *worker) fill(s *Screen, t int) {
	r, c := s.lines(t)
	parts := s.parts[s.spans[t][0]:s.spans[t][1]]
	if len(w.targets) == 0 {
		w.targets = append(w.targets, &w.view)
	}
	row := slices.Grow(w.line[:0], c.width)[:c.width]
	w.line = row
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y > r.Min.Y && repeats(parts, y) {
			c.link(y, c.lineOf[y-1])
			continue
		}
		clear(row)
		u := image.Rect(r.Min.X, y, r.Max.X, y+1)
		w.view = image.RGBA{Pix: row, Stride: len(row), Rect: u}
		for _, p := range parts {
			switch p.step {
			case openGroup:
				w.group(p.into, u)
			case drawBox:
				if d := p.r.Intersect(u); !d.Empty() {
					over(w.targets[p.into], d, p.c, d.Min.Sub(p.at))
				}
			case closeGroup:
				alpha := image.NewUniform(imagecolor.Alpha{A: uint8(math.Round(p.opacity * math.MaxUint8))})
				draw.DrawMask(w.targets[p.into-1], u, w.targets[p.into], u.Min, alpha, image.Point{}, draw.Over)
			}
		}
		if s.Profile == color.ANSI256 {
			quantise(row)
		}
		c.link(y, c.intern(row, maphash.Bytes(s.seed, row)))
	}
}

func repeats(parts []part, y int) bool {
	for _, p := range parts {
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

func (w *worker) group(into int, t image.Rectangle) {
	if len(w.targets) == into {
		w.targets = append(w.targets, &image.RGBA{})
	}
	g := w.targets[into]
	n := 4 * t.Dx() * t.Dy()
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], 4*t.Dx(), t
	clear(g.Pix)
}

func (s *Screen) look(b *scene.Box) *cached {
	if c, ok := s.cache[b.Look]; ok {
		return c
	}
	size, drawn := b.Visual.Size(), 0
	s.drawn, drawn = plan(s.drawn, b.Ops, float64(b.Visual.Min.Y), size.Y)
	c := &cached{frame: s.frame, stride: 4 * size.X, row: make([]int32, size.Y), pix: make([]uint8, 0, 4*size.X*drawn), solid: make([][2]int, 0, drawn)}
	s.cache[b.Look] = c
	s.pending = append(s.pending, pending{b, c})
	s.rastered++
	return c
}

func (s *Screen) rasterise() {
	slices.SortStableFunc(s.pending, func(a, b pending) int {
		return b.box.Visual.Dx()*b.box.Visual.Dy() - a.box.Visual.Dx()*a.box.Visual.Dy()
	})
	s.parallel(len(s.pending), func(w *worker, i int) { w.rasterise(s.pending[i].box, s.pending[i].c) })
	s.pending = s.pending[:0]
}

func (w *worker) rasterise(b *scene.Box, c *cached) {
	size := b.Visual.Size()
	w.ops = append(w.ops[:0], b.Ops...)
	for i := range w.ops {
		w.ops[i].Box.X -= float64(b.Visual.Min.X)
		w.ops[i].Box.Y -= float64(b.Visual.Min.Y)
	}
	stride := c.stride
	w.drawn, _ = plan(w.drawn, w.ops, 0, size.Y)
	for y := 0; y < size.Y; {
		if !w.drawn[y] {
			c.row[y] = c.row[y-1]
			y++
			continue
		}
		end := y + 1
		for end < size.Y && w.drawn[end] {
			end++
		}
		n := len(c.pix)
		w.canvas.Pix, w.canvas.Stride, w.canvas.Rect = c.pix[n:n+stride*(end-y)], stride, image.Rect(0, y, size.X, end)
		w.raster.Draw(&w.canvas, w.ops, w.canvas.Rect)
		for at := n; y < end; y, at = y+1, at+stride {
			line := c.pix[at : at+stride]
			if n > 0 && bytes.Equal(line, c.pix[n-stride:n]) {
				c.row[y] = c.row[y-1]
				continue
			}
			c.row[y] = int32(len(c.solid))
			c.solid = append(c.solid, solid(line))
			n += copy(c.pix[n:n+stride], line)
		}
		c.pix = c.pix[:n]
	}
	for y, start := 1, 0; y <= size.Y; y++ {
		if y < size.Y && c.row[y] == c.row[y-1] {
			continue
		}
		if y-start > c.uniform[1]-c.uniform[0] {
			c.uniform = [2]int{start, y}
		}
		start = y
	}
}

func plan(drawn []bool, ops []raster.Op, origin float64, h int) ([]bool, int) {
	drawn = slices.Grow(drawn[:0], h)[:h]
	clear(drawn)
	mark := func(from, to float64) {
		for y := max(int(math.Floor(from))-1, 0); y <= min(int(math.Ceil(to))+1, h-1); y++ {
			drawn[y] = true
		}
	}
	inside := func(y float64) bool { return y > 0 && y < float64(h) }
	mark(0, 0)
	for _, op := range ops {
		b, r := op.Box, op.Box.Radii
		b.Y -= origin
		top, bottom := max(r[0], r[1], op.Width), max(r[2], r[3], op.Width)
		switch {
		case op.Kind == raster.Opacity || len(op.Stops) > 0 || op.Dash != raster.Solid || op.Shadow.Inset:
			mark(0, float64(h))
		case op.Kind == raster.Clip && r == [4]float64{} && (inside(b.Y) || inside(b.Y+b.H)):
			mark(0, float64(h))
		case op.Kind == raster.Shadow:
			s := op.Shadow
			pad := s.Blur*rasterkonst.SigmaPerBlur*rasterkonst.ShadowReach + math.Abs(s.Spread) + 1
			y, height := b.Y+s.Y-s.Spread, max(b.H+2*s.Spread, 0)
			mark(y-pad, y+top+pad)
			mark(y+height-bottom-pad, y+height+pad)
		}
		if op.Kind != raster.Pop {
			mark(b.Y, b.Y+top)
			mark(b.Y+b.H-bottom, b.Y+b.H)
		}
	}
	rows := 0
	for _, d := range drawn {
		if d {
			rows++
		}
	}
	return drawn, rows
}

func solid(row []uint8) [2]int {
	var run [2]int
	start := 0
	for i := 0; i <= len(row); i += 4 {
		for i+8 <= len(row) && binary.LittleEndian.Uint64(row[i:])&graphicskonst.AlphaPair == graphicskonst.AlphaPair {
			i += 8
		}
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
	r = r.Intersect(s.bounds)
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
