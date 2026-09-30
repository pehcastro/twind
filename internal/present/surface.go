package present

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"hash/maphash"
	"image"
	"math"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	rasterkonst "github.com/twind-dev/twind/internal/konst/raster"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
)

type worker struct {
	raster  raster.Raster
	ops     []raster.Op
	turned  []raster.Op
	drawn   []bool
	across  []bool
	pieces  [][2]int
	pix     []uint8
	runs    []run
	ends    []int
	line    []uint8
	view    image.RGBA
	targets []*image.RGBA
	rows    [][]byte
	encoder encoder
	parts   []part
	groups  []group
	spans   []image.Rectangle
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

func (s *Screen) limit() int {
	return min(runtime.GOMAXPROCS(0), cmp.Or(s.Workers, graphicskonst.Workers))
}

func (s *Screen) parallel(n int, do func(w *worker, i int)) {
	workers := max(min(n, s.limit()), 1)
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

func (s *Screen) gather(f *scene.Frame) {
	s.hidden = slices.Grow(s.hidden[:0], len(f.Layers))[:len(f.Layers)]
	for i := range f.Layers {
		l := &f.Layers[i]
		if s.hidden[i] = l.Opacity <= 0 || l.Parent >= 0 && s.hidden[l.Parent]; s.hidden[i] {
			continue
		}
		for j := range l.Boxes {
			v := l.Boxes[j].Visual.Add(l.Origin).Intersect(l.Clip)
			if slices.ContainsFunc(s.drawing, func(t int) bool { return v.Overlaps(s.pixels(s.tiles[t])) }) {
				s.look(&l.Boxes[j])
			}
		}
	}
}

func (w *worker) collect(s *Screen, f *scene.Frame, t int) {
	r := s.pixels(s.tiles[t])
	w.parts = w.parts[:0]
	for i := range f.Layers {
		l := &f.Layers[i]
		for len(w.groups) > 0 && w.groups[len(w.groups)-1].layer != l.Parent {
			w.close()
		}
		into := 0
		if n := len(w.groups); n > 0 {
			into = w.groups[n-1].into
		}
		own := into >= 0 && l.Opacity > 0 && l.Opacity < 1
		switch {
		case own:
			into++
			w.parts = append(w.parts, part{step: openGroup, into: into})
		case l.Opacity <= 0:
			into = -1
		}
		w.groups = append(w.groups, group{layer: i, into: into, opacity: l.Opacity, own: own})
		if into < 0 || !l.Visual.Overlaps(r) {
			continue
		}
		for j := range l.Boxes {
			b := &l.Boxes[j]
			at := b.Visual.Add(l.Origin)
			if v := at.Intersect(r).Intersect(l.Clip); !v.Empty() {
				w.parts = append(w.parts, part{step: drawBox, c: s.cache[b.Look], r: v, at: at.Min, into: into})
			}
		}
	}
	for len(w.groups) > 0 {
		w.close()
	}
}

func (w *worker) close() {
	g := w.groups[len(w.groups)-1]
	w.groups = w.groups[:len(w.groups)-1]
	n := len(w.parts)
	switch {
	case !g.own:
	case w.parts[n-1].step == openGroup:
		w.parts = w.parts[:n-1]
	case w.parts[n-2].step == openGroup && w.parts[n-1].step == drawBox && w.parts[n-1].opacity == 0:
		w.parts[n-1].into, w.parts[n-1].opacity = g.into-1, g.opacity
		w.parts[n-2] = w.parts[n-1]
		w.parts = w.parts[:n-1]
	default:
		w.parts = append(w.parts, part{step: closeGroup, into: g.into, opacity: g.opacity})
	}
}

func (w *worker) fill(s *Screen, f *scene.Frame, t int) {
	w.collect(s, f, t)
	r, c := s.lines(t)
	parts := w.parts
	for _, p := range parts {
		if p.step == drawBox {
			p.c.ready.Wait()
		}
	}
	if len(w.targets) == 0 {
		w.targets, w.spans = append(w.targets, &w.view), append(w.spans, image.Rectangle{})
	}
	again := len(s.bases[t]) > 0 && len(parts) >= len(s.bases[t]) && slices.Equal(parts[:len(s.bases[t])], s.bases[t])
	if !again {
		n := 0
		for n < len(parts) && parts[n].step == drawBox && parts[n].into == 0 && parts[n].opacity == 0 {
			n++
		}
		if n == len(parts) {
			n = 0
		}
		s.bases[t] = append(s.bases[t][:0], parts[:n]...)
	}
	kept, from := again && s.based[t], 0
	if again {
		from = len(s.bases[t])
	}
	s.based[t] = again
	row := slices.Grow(w.line[:0], c.width)[:c.width]
	w.line = row
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y > r.Min.Y && repeats(parts, y) {
			c.link(c.lineOf, y, c.lineOf[y-1])
			c.link(c.baseOf, y, c.baseOf[y-1])
			continue
		}
		u := image.Rect(r.Min.X, y, r.Max.X, y+1)
		w.view = image.RGBA{Pix: row, Stride: len(row), Rect: u}
		switch {
		case kept:
			copy(row, c.store[c.baseOf[y]])
		case from > 0:
			clear(row)
			w.draw(parts[:from], u)
			c.link(c.baseOf, y, c.intern(row, maphash.Bytes(s.seed, row)))
		default:
			clear(row)
			c.link(c.baseOf, y, -1)
		}
		w.draw(parts[from:], u)
		if s.Profile == color.ANSI256 {
			quantise(row)
		}
		c.link(c.lineOf, y, c.intern(row, maphash.Bytes(s.seed, row)))
	}
}

func (w *worker) draw(parts []part, u image.Rectangle) {
	for _, p := range parts {
		switch p.step {
		case openGroup:
			w.group(p.into, u)
		case drawBox:
			if d := p.r.Intersect(u); !d.Empty() {
				over(w.targets[p.into], d, p.c, d.Min.Sub(p.at), p.opacity)
				w.spans[p.into] = w.spans[p.into].Union(d)
			}
		case closeGroup:
			span := w.spans[p.into]
			w.spans[p.into], w.spans[p.into-1] = image.Rectangle{}, w.spans[p.into-1].Union(span)
			if span.Empty() {
				continue
			}
			lo, hi := 4*(span.Min.X-u.Min.X), 4*(span.Max.X-u.Min.X)
			src, dst, a := w.targets[p.into].Pix[lo:hi], w.targets[p.into-1].Pix[lo:hi], alpha(p.opacity)
			for i := 0; i < len(src); {
				px, j := binary.LittleEndian.Uint32(src[i:]), i+4
				for j < len(src) && binary.LittleEndian.Uint32(src[j:]) == px {
					j += 4
				}
				mask(dst[i:j], px, a)
				i = j
			}
			clear(src)
		}
	}
}

func repeats(parts []part, y int) bool {
	for _, p := range parts {
		if p.step != drawBox {
			continue
		}
		in, was := y >= p.r.Min.Y && y < p.r.Max.Y, y > p.r.Min.Y && y <= p.r.Max.Y
		if in != was {
			return false
		}
		if !in {
			continue
		}
		if k := p.c.row[y-p.at.Y]; k != p.c.row[y-1-p.at.Y] && int(p.c.change[k][0]) < p.r.Max.X-p.at.X && p.r.Min.X-p.at.X < int(p.c.change[k][1]) {
			return false
		}
	}
	return true
}

func find(runs []run, x int) int {
	i, _ := slices.BinarySearchFunc(runs, x+1, func(r run, x int) int { return cmp.Compare(int(r.end), x) })
	return i
}

func differ(a, b []run) [2]int32 {
	start := func(r []run, i int) int32 {
		if i == 0 {
			return 0
		}
		return r[i-1].end
	}
	lo, i, j := int32(0), 0, 0
	for i < len(a) && a[i].px == b[j].px {
		lo = min(a[i].end, b[j].end)
		if a[i].end == lo {
			i++
		}
		if b[j].end == lo {
			j++
		}
	}
	hi, i, j := a[len(a)-1].end, len(a)-1, len(b)-1
	for i >= 0 && a[i].px == b[j].px {
		hi = max(start(a, i), start(b, j))
		if start(a, i) == hi {
			i--
		}
		if start(b, j) == hi {
			j--
		}
	}
	return [2]int32{lo, hi}
}

func over(dst *image.RGBA, r image.Rectangle, src *cached, at image.Point, opacity float64) {
	a := alpha(opacity)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		d := dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Max.X, y)]
		runs := src.lines[src.row[at.Y+y-r.Min.Y]]
		for x, i := at.X, find(runs, at.X); x < at.X+r.Dx(); i++ {
			end := min(int(runs[i].end), at.X+r.Dx())
			if opacity > 0 {
				mask(d[4*(x-at.X):4*(end-at.X)], runs[i].px, a)
			} else {
				flood(d[4*(x-at.X):4*(end-at.X)], runs[i].px)
			}
			x = end
		}
	}
}

func alpha(opacity float64) uint32 { return uint32(math.Round(opacity * math.MaxUint8)) }

func mask(d []uint8, px, alpha uint32) {
	if px>>24 == 0 {
		return
	}
	const widen = math.MaxUint16 / math.MaxUint8
	ma := alpha * widen
	keep := (math.MaxUint16 - (px>>24*widen)*ma/math.MaxUint16) * widen
	var under, out uint32
	for i := 0; i < len(d); {
		if u := binary.LittleEndian.Uint32(d[i:]); i == 0 || u != under {
			under, out = u, 0
			for shift := 0; shift < 32; shift += 8 {
				out |= ((under>>shift&math.MaxUint8)*keep + (px>>shift&math.MaxUint8)*widen*ma) / math.MaxUint16 >> 8 & math.MaxUint8 << shift
			}
		}
		binary.LittleEndian.PutUint32(d[i:], out)
		pairs, outs := uint64(under)<<32|uint64(under), uint64(out)<<32|uint64(out)
		for i += 4; i+8 <= len(d) && binary.LittleEndian.Uint64(d[i:]) == pairs; i += 8 {
			binary.LittleEndian.PutUint64(d[i:], outs)
		}
	}
}

func flood(d []uint8, px uint32) {
	switch a := px >> 24; a {
	case 0:
	case math.MaxUint8:
		binary.LittleEndian.PutUint32(d, px)
		for n := 4; n < len(d); n *= 2 {
			copy(d[n:], d[:n])
		}
	default:
		var under, out uint32
		for i := 0; i < len(d); i += 4 {
			if u := binary.LittleEndian.Uint32(d[i:]); i == 0 || u != under {
				under, out = u, 0
				for shift := 0; shift < 32; shift += 8 {
					kept := (under >> shift & math.MaxUint8) * (math.MaxUint8 - a)
					out |= uint32(uint8(px>>shift)+uint8((kept+math.MaxUint8/2)/math.MaxUint8)) << shift
				}
			}
			binary.LittleEndian.PutUint32(d[i:], out)
		}
	}
}

func (w *worker) group(into int, t image.Rectangle) {
	if len(w.targets) == into {
		w.targets, w.spans = append(w.targets, &image.RGBA{}), append(w.spans, image.Rectangle{})
	}
	g := w.targets[into]
	n := 4 * t.Dx() * t.Dy()
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], 4*t.Dx(), t
}

func (s *Screen) look(b *scene.Box) *cached {
	if c, ok := s.cache[b.Look]; ok {
		return c
	}
	s.shape = s.canonical(s.shape[:0], b)
	c, ok := s.shapes[string(s.shape)]
	if !ok {
		c = &cached{frame: s.frame, row: make([]int32, b.Visual.Dy())}
		c.ready.Add(1)
		s.shapes[string(s.shape)] = c
		s.pending = append(s.pending, pending{b, c})
		s.rastered++
	}
	s.cache[b.Look] = c
	return c
}

func (s *Screen) canonical(key []byte, b *scene.Box) []byte {
	size, origin := b.Visual.Size(), b.Visual.Min
	number := func(v ...float64) {
		for _, f := range v {
			key = binary.LittleEndian.AppendUint64(key, math.Float64bits(f))
		}
	}
	number(float64(size.X), float64(size.Y))
	s.idle = s.idle[:0]
	for _, op := range b.Ops {
		x, y, r := op.Box.X-float64(origin.X), op.Box.Y-float64(origin.Y), op.Box.Radii
		switch op.Kind {
		case raster.Clip:
			idle := r == [4]float64{} && x <= 0 && y <= 0 && x+op.Box.W >= float64(size.X) && y+op.Box.H >= float64(size.Y)
			if s.idle = append(s.idle, idle); idle {
				continue
			}
		case raster.Opacity:
			s.idle = append(s.idle, false)
		case raster.Pop:
			n := len(s.idle) - 1
			idle := n >= 0 && s.idle[n]
			if s.idle = s.idle[:max(n, 0)]; idle {
				continue
			}
		}
		inset := byte(0)
		if op.Shadow.Inset {
			inset = 1
		}
		key = binary.AppendUvarint(append(key, byte(op.Kind), byte(op.Dash), inset, op.Color.R, op.Color.G, op.Color.B, op.Color.A), uint64(len(op.Stops)))
		number(x, y, op.Box.W, op.Box.H, r[0], r[1], r[2], r[3], op.Angle, op.Width, op.Opacity, op.Shadow.X, op.Shadow.Y, op.Shadow.Blur, op.Shadow.Spread)
		for _, stop := range op.Stops {
			key = append(key, stop.Color.R, stop.Color.G, stop.Color.B, stop.Color.A)
			number(stop.At)
		}
	}
	return key
}

func (s *Screen) rasterise(n int, then func(w *worker, i int)) {
	slices.SortStableFunc(s.pending, func(a, b pending) int {
		return b.box.Visual.Dx()*b.box.Visual.Dy() - a.box.Visual.Dx()*a.box.Visual.Dy()
	})
	looks := len(s.pending)
	s.parallel(looks+n, func(w *worker, i int) {
		if i >= looks {
			then(w, i-looks)
			return
		}
		w.rasterise(s.pending[i].box, s.pending[i].c)
		s.pending[i].c.ready.Done()
	})
	s.pending = s.pending[:0]
}

func (w *worker) rasterise(b *scene.Box, c *cached) {
	size := b.Visual.Size()
	w.ops = append(w.ops[:0], b.Ops...)
	for i := range w.ops {
		w.ops[i].Box.X -= float64(b.Visual.Min.X)
		w.ops[i].Box.Y -= float64(b.Visual.Min.Y)
	}
	w.turned = append(w.turned[:0], w.ops...)
	for i := range w.turned {
		b, s := &w.turned[i].Box, &w.turned[i].Shadow
		b.X, b.Y, b.W, b.H = b.Y, b.X, b.H, b.W
		b.Radii[1], b.Radii[3] = b.Radii[3], b.Radii[1]
		s.X, s.Y = s.Y, s.X
	}
	w.drawn, w.across = plan(w.drawn, w.ops, size.Y), plan(w.across, w.turned, size.X)
	w.pieces = w.pieces[:0]
	width := 0
	for x, drawn := range w.across {
		switch {
		case !drawn:
			continue
		case x > 0 && w.across[x-1]:
			w.pieces[len(w.pieces)-1][1]++
		default:
			w.pieces = append(w.pieces, [2]int{x, x + 1})
		}
		width++
	}
	w.runs, w.ends = w.runs[:0], w.ends[:0]
	for y := 0; y < size.Y; {
		if !w.drawn[y] {
			c.row[y] = c.row[y-1]
			y++
			continue
		}
		top, end := y, y+1
		for end < size.Y && w.drawn[end] {
			end++
		}
		stride := 4 * width
		w.pix = slices.Grow(w.pix[:0], stride*(end-top))[:stride*(end-top)]
		left := 0
		for _, p := range w.pieces {
			canvas := image.RGBA{Pix: w.pix[4*left:], Stride: stride, Rect: image.Rect(p[0], top, p[1], end)}
			w.raster.Draw(&canvas, w.ops, canvas.Rect)
			left += p[1] - p[0]
		}
		for ; y < end; y++ {
			row := w.pix[(y-top)*stride:][:stride]
			if y > top && bytes.Equal(row, w.pix[(y-top-1)*stride:][:stride]) {
				c.row[y] = c.row[y-1]
				continue
			}
			start, left := len(w.runs), 0
			for i, p := range w.pieces {
				w.runs = encode(w.runs, start, row[4*left:4*(left+p[1]-p[0])], p[0])
				left += p[1] - p[0]
				w.runs[len(w.runs)-1].end = int32(size.X)
				if i+1 < len(w.pieces) {
					w.runs[len(w.runs)-1].end = int32(w.pieces[i+1][0])
				}
			}
			k, previous := len(w.ends), 0
			if k > 1 {
				previous = w.ends[k-2]
			}
			if y == top && k > 0 && slices.Equal(w.runs[start:], w.runs[previous:start]) {
				w.runs, c.row[y] = w.runs[:start], c.row[y-1]
				continue
			}
			c.row[y], w.ends = int32(k), append(w.ends, len(w.runs))
		}
	}
	all, from := slices.Clone(w.runs), 0
	c.lines, c.change = make([][]run, len(w.ends)), make([][2]int32, len(w.ends))
	for k, to := range w.ends {
		c.lines[k], from = all[from:to:to], to
		if k > 0 {
			c.change[k] = differ(c.lines[k], c.lines[k-1])
		}
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

func encode(runs []run, start int, row []uint8, x int) []run {
	for i := 0; i < len(row); {
		px := binary.LittleEndian.Uint32(row[i:])
		j, pair := i+4, uint64(px)<<32|uint64(px)
		for j+8 <= len(row) && binary.LittleEndian.Uint64(row[j:]) == pair {
			j += 8
		}
		for j < len(row) && binary.LittleEndian.Uint32(row[j:]) == px {
			j += 4
		}
		if n := len(runs); n > start && runs[n-1].px == px {
			runs[n-1].end = int32(x + j/4)
		} else {
			runs = append(runs, run{end: int32(x + j/4), px: px})
		}
		i = j
	}
	return runs
}

func plan(drawn []bool, ops []raster.Op, h int) []bool {
	drawn = slices.Grow(drawn[:0], h)[:h]
	clear(drawn)
	mark := func(from, to float64) {
		for y := max(int(math.Floor(from))-1, 0); y <= min(int(math.Ceil(to))+1, h-1); y++ {
			drawn[y] = true
		}
	}
	mark(0, 0)
	for _, op := range ops {
		b, r, fit := op.Box, op.Box.Radii, 1.0
		for _, side := range [4][3]float64{{b.W, r[0], r[1]}, {b.W, r[3], r[2]}, {b.H, r[0], r[3]}, {b.H, r[1], r[2]}} {
			if sum := side[1] + side[2]; sum > side[0] {
				fit = min(fit, side[0]/sum)
			}
		}
		top, bottom := max(fit*max(r[0], r[1]), op.Width), max(fit*max(r[2], r[3]), op.Width)
		switch {
		case op.Kind == raster.Opacity || len(op.Stops) > 0 || op.Dash != raster.Solid || op.Shadow.Inset:
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
	return drawn
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
	for shape, c := range s.shapes {
		if c.frame != s.frame {
			delete(s.shapes, shape)
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
