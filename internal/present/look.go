package present

import (
	"bytes"
	"encoding/binary"
	"image"
	"math"
	"slices"
	"sync/atomic"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	rasterkonst "github.com/twind-dev/twind/internal/konst/raster"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
)

type pending struct {
	box    *scene.Box
	c      *cached
	ops    []raster.Op
	drawn  []bool
	pieces [][2]int
	pix    []uint8
	width  int
	jobs   int32
}

type job struct {
	look, left, top, bottom, at int
	piece                       [2]int
}

func (s *Screen) look(b *scene.Box) *cached {
	if c, ok := s.cache[b.Look]; ok {
		return c
	}
	s.shape = s.canonical(s.shape[:0], b)
	c, ok := s.shapes[string(s.shape)]
	if !ok {
		c = &cached{frame: s.frame, row: make([]int32, b.Visual.Dy()), id: s.rastered}
		c.ready.Add(1)
		s.shapes[string(s.shape)] = c
		s.pending = append(s.pending, pending{box: b, c: c})
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
	s.jobs, s.lookOps, s.lookRows, s.lookPieces, s.lookAt = s.jobs[:0], s.lookOps[:0], s.lookRows[:0], s.lookPieces[:0], s.lookAt[:0]
	pixels := 0
	for i := range s.pending {
		l := &s.pending[i]
		size, ops, rows, pieces := l.box.Visual.Size(), len(s.lookOps), len(s.lookRows), len(s.lookPieces)
		s.lookOps = append(s.lookOps, l.box.Ops...)
		for k := ops; k < len(s.lookOps); k++ {
			s.lookOps[k].Box.X -= float64(l.box.Visual.Min.X)
			s.lookOps[k].Box.Y -= float64(l.box.Visual.Min.Y)
		}
		s.turned = append(s.turned[:0], s.lookOps[ops:]...)
		for k := range s.turned {
			b, sh := &s.turned[k].Box, &s.turned[k].Shadow
			b.X, b.Y, b.W, b.H = b.Y, b.X, b.H, b.W
			b.Radii[1], b.Radii[3] = b.Radii[3], b.Radii[1]
			sh.X, sh.Y = sh.Y, sh.X
		}
		s.lookRows = slices.Grow(s.lookRows, size.Y)[:rows+size.Y]
		s.across = slices.Grow(s.across[:0], size.X)[:size.X]
		plan(s.lookRows[rows:], s.lookOps[ops:])
		plan(s.across, s.turned)
		l.width = 0
		for x, drawn := range s.across {
			switch {
			case !drawn:
				continue
			case x > 0 && s.across[x-1]:
				s.lookPieces[len(s.lookPieces)-1][1]++
			default:
				s.lookPieces = append(s.lookPieces, [2]int{x, x + 1})
			}
			l.width++
		}
		first, drawn := len(s.jobs), 0
		for y := 0; y < size.Y; {
			if !s.lookRows[rows+y] {
				y++
				continue
			}
			end := y + 1
			for end < size.Y && s.lookRows[rows+end] {
				end++
			}
			left := 0
			for _, p := range s.lookPieces[pieces:] {
				step := max(graphicskonst.JobPixels/(p[1]-p[0]), 1)
				for top := y; top < end; top += step {
					s.jobs = append(s.jobs, job{look: i, left: left, top: top, bottom: min(top+step, end), at: drawn + top - y, piece: p})
				}
				left += p[1] - p[0]
			}
			drawn += end - y
			y = end
		}
		l.jobs = int32(len(s.jobs) - first)
		s.lookAt = append(s.lookAt, [4]int{ops, rows, pieces, pixels})
		pixels += 4 * l.width * drawn
	}
	s.lookPix = slices.Grow(s.lookPix[:0], pixels)[:pixels]
	for i, a := range s.lookAt {
		l, next := &s.pending[i], [4]int{len(s.lookOps), len(s.lookRows), len(s.lookPieces), pixels}
		if i+1 < len(s.lookAt) {
			next = s.lookAt[i+1]
		}
		l.ops, l.drawn, l.pieces, l.pix = s.lookOps[a[0]:next[0]], s.lookRows[a[1]:next[1]], s.lookPieces[a[2]:next[2]], s.lookPix[a[3]:next[3]]
	}
	slices.SortStableFunc(s.jobs, func(a, b job) int {
		return (b.bottom-b.top)*(b.piece[1]-b.piece[0]) - (a.bottom-a.top)*(a.piece[1]-a.piece[0])
	})
	jobs := len(s.jobs)
	s.parallel(jobs+n, func(w *worker, i int) {
		if i >= jobs {
			then(w, i-jobs)
			return
		}
		j := s.jobs[i]
		l := &s.pending[j.look]
		stride := 4 * l.width
		canvas := image.RGBA{Pix: l.pix[j.at*stride+4*j.left:], Stride: stride, Rect: image.Rect(j.piece[0], j.top, j.piece[1], j.bottom)}
		w.raster.Draw(&canvas, l.ops, canvas.Rect)
		if atomic.AddInt32(&l.jobs, -1) == 0 {
			w.finish(l)
			l.c.ready.Done()
		}
	})
	s.pending = s.pending[:0]
}

func (w *worker) finish(l *pending) {
	c, size, stride := l.c, l.box.Visual.Size(), 4*l.width
	w.runs, w.ends = w.runs[:0], w.ends[:0]
	var previous []uint8
	for y, p := 0, 0; y < size.Y; y++ {
		if !l.drawn[y] {
			c.row[y] = c.row[y-1]
			continue
		}
		row, top := l.pix[p*stride:][:stride], y == 0 || !l.drawn[y-1]
		p++
		if same := !top && bytes.Equal(row, previous); same {
			previous, c.row[y] = row, c.row[y-1]
			continue
		}
		previous = row
		start, left := len(w.runs), 0
		for i, piece := range l.pieces {
			w.runs = encode(w.runs, start, row[4*left:4*(left+piece[1]-piece[0])], piece[0])
			left += piece[1] - piece[0]
			w.runs[len(w.runs)-1].End = int32(size.X)
			if i+1 < len(l.pieces) {
				w.runs[len(w.runs)-1].End = int32(l.pieces[i+1][0])
			}
		}
		k, before := len(w.ends), 0
		if k > 1 {
			before = w.ends[k-2]
		}
		if top && k > 0 && slices.Equal(w.runs[start:], w.runs[before:start]) {
			w.runs, c.row[y] = w.runs[:start], c.row[y-1]
			continue
		}
		c.row[y], w.ends = int32(k), append(w.ends, len(w.runs))
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
		if n := len(runs); n > start && runs[n-1].Pixel == px {
			runs[n-1].End = int32(x + j/4)
		} else {
			runs = append(runs, run{End: int32(x + j/4), Pixel: px})
		}
		i = j
	}
	return runs
}

func differ(a, b []run) [2]int32 {
	start := func(r []run, i int) int32 {
		if i == 0 {
			return 0
		}
		return r[i-1].End
	}
	lo, i, j := int32(0), 0, 0
	for i < len(a) && a[i].Pixel == b[j].Pixel {
		lo = min(a[i].End, b[j].End)
		if a[i].End == lo {
			i++
		}
		if b[j].End == lo {
			j++
		}
	}
	hi, i, j := a[len(a)-1].End, len(a)-1, len(b)-1
	for i >= 0 && a[i].Pixel == b[j].Pixel {
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

func plan(drawn []bool, ops []raster.Op) {
	h := len(drawn)
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
