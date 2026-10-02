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
	konst "github.com/twind-dev/twind/internal/konst/present"
	scenekonst "github.com/twind-dev/twind/internal/konst/scene"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
)

type worker struct {
	raster raster.Raster
	runs   []run
	ends   []int
	layers []layer
	spans  [][2]int32
	store  []run
	base   []run
	key    []byte
	recipe []byte
	hashed uint64
	links  []link
	lazy   bool
	lines  [][]run
	joined []run
	cuts   []int
	bands  []band
	sums   [][4]int
	pix    []uint8
	rows   [][]byte
	sixel  *graphics.Sixel
	iterm  *graphics.ITerm
	parts  []part
	groups []group
	out    []byte
}

type link struct {
	k       int32
	n, size int
	hash    uint64
}

type piece struct {
	w      *worker
	lo, hi int
}

type recipe struct{ tile, at, end int }

type layer struct {
	buf [2][]run
	n   [2]int32
	cur int
}

type blend uint8

const (
	floodBlend blend = iota
	maskBlend
	copyBlend
)

type segment uint8

const (
	flatRows segment = iota
	lookRows
)

func (s *Screen) parallel(n, lead, tiles int, first func(), do func(w *worker, i int)) {
	workers := max(min(cmp.Or(s.Workers, runtime.GOMAXPROCS(0)), 1+tiles/konst.TilesPerWorker), 1)
	for len(s.workers) < workers {
		s.workers = append(s.workers, &worker{})
	}
	var next atomic.Int64
	take := func(w *worker, below int) {
		for i := next.Load(); int(i) < below; i = next.Load() {
			if next.CompareAndSwap(i, i+1) {
				do(w, int(i))
			}
		}
	}
	var claimed atomic.Bool
	once := func() {
		if first != nil && claimed.CompareAndSwap(false, true) {
			first()
		}
	}
	var wg sync.WaitGroup
	helpers := s.workers[1:workers]
	if first != nil && len(helpers) > 0 {
		lane := helpers[0]
		helpers = helpers[1:]
		wg.Go(func() {
			take(lane, lead)
			once()
		})
	}
	for _, w := range helpers[:min(len(helpers), max(n-1, 0))] {
		wg.Go(func() { take(w, n) })
	}
	if workers == 1 {
		once()
	}
	take(s.workers[0], n)
	once()
	wg.Wait()
}

func (s *Screen) gather(f *scene.Frame) {
	s.hidden = slices.Grow(s.hidden[:0], len(f.Layers))[:len(f.Layers)]
	whole := len(s.drawing) == len(s.tiles)
	for i := range f.Layers {
		l := &f.Layers[i]
		if s.hidden[i] = l.Opacity <= 0 || l.Parent >= 0 && s.hidden[l.Parent]; s.hidden[i] {
			continue
		}
		for j := range l.Boxes {
			v := l.Boxes[j].Visual.Add(l.Origin).Intersect(l.Clip)
			if whole && v.Overlaps(s.bounds) || !whole && slices.ContainsFunc(s.drawing, func(t int) bool { return v.Overlaps(s.pixels(s.tiles[t])) }) {
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

func (w *worker) fill(s *Screen, f *scene.Frame, t int) (twin int, memo bool) {
	w.lazy = false
	w.collect(s, f, t)
	r, c := s.lines(t)
	parts, deep := w.parts, 0
	for _, p := range parts {
		if deep = max(deep, p.into); p.step == drawBox {
			p.c.ready.Wait()
		}
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
	if cap(w.spans) < r.Dy() {
		w.spans = make([][2]int32, r.Dy())
	}
	spans, store := w.spans[:r.Dy()], w.store[:0]
	if from == 0 {
		w.recipe = w.describe(w.recipe[:0], r)
		w.hashed = maphash.Bytes(s.seed, w.recipe)
		s.lock.Lock()
		twin := s.recall(w.hashed, w.recipe)
		s.lock.Unlock()
		if twin >= 0 {
			area, source := s.lines(twin)
			links := w.links[:0]
			source.lock.Lock()
			for y := area.Min.Y; y < area.Max.Y; {
				k, n := source.lineOf[y], 1
				for y+n < area.Max.Y && source.lineOf[y+n] == k {
					n++
				}
				links = append(links, link{k: k, n: n, hash: source.hash[k], size: len(source.store[k])})
				if source != c {
					store = append(store, source.store[k]...)
				}
				y += n
			}
			if source != c {
				source.lock.Unlock()
				c.lock.Lock()
			}
			y, at := r.Min.Y, 0
			for _, e := range links {
				if source != c {
					e.k, at = c.intern(store[at:at+e.size], e.hash), at+e.size
				}
				for range e.n {
					c.link(c.lineOf, y, e.k)
					if len(c.baseOf) > 0 {
						c.link(c.baseOf, y, -1)
					}
					y++
				}
			}
			c.lock.Unlock()
			w.links, w.lazy = links, true
			w.keep(store)
			s.hashes[t] = s.hashes[twin]
			return twin, false
		}
	}
	for len(w.layers) <= deep {
		w.layers = append(w.layers, layer{})
	}
	l, width := &w.layers[0], int32(c.width)
	h, hash, linked := uint64(scenekonst.HashSeed)^uint64(s.page), uint64(0), r.Min.Y
	for y := r.Min.Y; y < r.Max.Y; y++ {
		if y > r.Min.Y && repeats(parts, y) {
			h = (h ^ hash) * scenekonst.HashPrime
			spans[y-r.Min.Y] = spans[y-r.Min.Y-1]
			continue
		}
		l.clear(width)
		base, below := uint64(0), []run(nil)
		switch {
		case kept:
			c.lock.Lock()
			l.set(c.store[c.baseOf[y]])
			c.lock.Unlock()
		case from > 0:
			w.draw(parts[:from], y, r.Min.X, width)
			w.base = append(w.base[:0], l.runs()...)
			below, base = w.base, s.digest(&w.key, w.base)
		}
		w.draw(parts[from:], y, r.Min.X, width)
		if s.page != 0 {
			l.n[l.cur] = int32(len(unpaged(l.runs(), s.page)))
		}
		if s.Profile == color.ANSI256 {
			l.n[l.cur] = int32(len(quantise(l.runs())))
		}
		row := l.runs()
		hash = s.digest(&w.key, row)
		h = (h ^ hash) * scenekonst.HashPrime
		spans[y-r.Min.Y] = [2]int32{int32(len(store)), int32(len(store) + len(row))}
		store = append(store, row...)
		c.lock.Lock()
		repeat(c, linked, y)
		switch {
		case kept:
		case from > 0:
			if len(c.baseOf) == 0 {
				c.baseOf = slices.Grow(c.baseOf, len(c.lineOf))[:len(c.lineOf)]
				for i := range c.baseOf {
					c.baseOf[i] = -1
				}
			}
			c.link(c.baseOf, y, c.intern(below, base))
		case len(c.baseOf) > 0:
			c.link(c.baseOf, y, -1)
		}
		c.link(c.lineOf, y, c.intern(row, hash))
		c.lock.Unlock()
		linked = y + 1
	}
	c.lock.Lock()
	repeat(c, linked, r.Max.Y)
	c.lock.Unlock()
	w.keep(store)
	s.hashes[t] = h
	return -1, from == 0
}

func (w *worker) keep(store []run) {
	if cap(store) != cap(w.store) {
		w.store = store
	}
}

func (w *worker) rowLines(n int) [][]run {
	lines := slices.Grow(w.lines[:0], n)[:n]
	for i, sp := range w.spans[:n] {
		if i > 0 && sp == w.spans[i-1] {
			lines[i] = lines[i-1]
			continue
		}
		lines[i] = w.store[sp[0]:sp[1]]
	}
	w.lines = lines
	return lines
}

func (w *worker) describe(key []byte, r image.Rectangle) []byte {
	key = binary.AppendUvarint(binary.AppendUvarint(key, uint64(r.Dx())), uint64(r.Dy()))
	for _, p := range w.parts {
		key = binary.AppendUvarint(append(key, byte(p.step)), uint64(p.into))
		key = binary.LittleEndian.AppendUint64(key, math.Float64bits(p.opacity))
		if p.step != drawBox {
			continue
		}
		for _, v := range [4]int{p.r.Min.X - r.Min.X, p.r.Min.Y - r.Min.Y, p.r.Max.X - r.Min.X, p.r.Max.Y - r.Min.Y} {
			key = binary.AppendUvarint(key, uint64(v))
		}
		lo, hi := p.r.Min.X-p.at.X, int32(p.r.Max.X-p.at.X)
		flat, rows := uint32(0), 0
		flush := func() {
			if rows > 0 {
				key = binary.AppendUvarint(binary.LittleEndian.AppendUint32(append(key, byte(flatRows)), flat), uint64(rows))
			}
			rows = 0
		}
		uniform, pixel := false, uint32(0)
		for y := p.r.Min.Y; y < p.r.Max.Y; {
			k := p.c.row[y-p.at.Y]
			n := min(int(p.c.starts[k+1])+p.at.Y, p.r.Max.Y) - y
			if change := p.c.change[k]; y == p.r.Min.Y || change[0] < hi && int32(lo) < change[1] {
				line := p.c.lines[k]
				i := find(line, lo)
				uniform, pixel = line[i].End >= hi, line[i].Pixel
			}
			y += n
			if uniform {
				if pixel != flat {
					flush()
				}
				flat, rows = pixel, rows+n
				continue
			}
			flush()
			key = append(key, byte(lookRows))
			for _, v := range [4]int{p.c.id, int(k), lo, n} {
				key = binary.AppendUvarint(key, uint64(v))
			}
		}
		flush()
	}
	return key
}

func (s *Screen) recall(hash uint64, key []byte) int {
	if e, ok := s.recipes[hash]; ok && bytes.Equal(s.arena[e.at:e.end], key) {
		return e.tile
	}
	return -1
}

func (w *worker) remember(s *Screen, t int) {
	s.lock.Lock()
	if _, taken := s.recipes[w.hashed]; !taken {
		s.recipes[w.hashed] = recipe{tile: t, at: len(s.arena), end: len(s.arena) + len(w.recipe)}
		s.arena = append(s.arena, w.recipe...)
	}
	s.lock.Unlock()
}

func repeat(c *column, from, to int) {
	for y := from; y < to; y++ {
		c.link(c.lineOf, y, c.lineOf[y-1])
		if len(c.baseOf) > 0 {
			c.link(c.baseOf, y, c.baseOf[y-1])
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

func (w *worker) draw(parts []part, y, left int, width int32) {
	for _, p := range parts {
		switch p.step {
		case openGroup:
			w.layers[p.into].clear(width)
		case drawBox:
			if y >= p.r.Min.Y && y < p.r.Max.Y {
				mode := maskBlend
				if p.opacity == 0 {
					mode = floodBlend
				}
				w.paint(p.into, int32(p.r.Min.X-left), int32(p.r.Max.X-left), p.c.lines[p.c.row[y-p.at.Y]], int32(left-p.at.X), mode, alpha(p.opacity))
			}
		case closeGroup:
			w.paint(p.into-1, 0, width, w.layers[p.into].runs(), 0, maskBlend, alpha(p.opacity))
		}
	}
}

func (w *worker) paint(into int, lo, hi int32, src []run, shift int32, mode blend, a uint32) {
	l := &w.layers[into]
	other := 1 - l.cur
	out := compose(l.buf[other][:0], l.runs(), lo, hi, src, shift, mode, a)
	if cap(out) != cap(l.buf[other]) {
		l.buf[other] = out
	}
	l.cur, l.n[other] = other, int32(len(out))
}

func (l *layer) runs() []run { return l.buf[l.cur][:l.n[l.cur]] }

func (l *layer) clear(width int32) {
	if cap(l.buf[l.cur]) == 0 {
		l.buf[l.cur] = make([]run, 1, graphicskonst.RunChunk)
	}
	l.buf[l.cur][:1][0], l.n[l.cur] = run{End: width}, 1
}

func (l *layer) set(runs []run) {
	l.buf[l.cur] = append(l.buf[l.cur][:0], runs...)
	l.n[l.cur] = int32(len(runs))
}

func compose(out, dst []run, lo, hi int32, src []run, shift int32, mode blend, a uint32) []run {
	i := 0
	for ; dst[i].End <= lo; i++ {
		out = append(out, dst[i])
	}
	if i > 0 && dst[i-1].End < lo || i == 0 && lo > 0 {
		out = add(out, lo, dst[i].Pixel)
	}
	for x, j := lo, find(src, int(lo+shift)); x < hi; {
		end := min(hi, dst[i].End, src[j].End-shift)
		px := src[j].Pixel
		switch mode {
		case floodBlend:
			px = flooded(dst[i].Pixel, px)
		case maskBlend:
			px = masked(dst[i].Pixel, px, a)
		case copyBlend:
		}
		out = add(out, end, px)
		if dst[i].End == end {
			i++
		}
		if src[j].End-shift == end {
			j++
		}
		x = end
	}
	for ; i < len(dst); i++ {
		out = add(out, dst[i].End, dst[i].Pixel)
	}
	return out
}

func add(out []run, end int32, px uint32) []run {
	if n := len(out); n > 0 && out[n-1].Pixel == px {
		out[n-1].End = end
		return out
	}
	return append(out, run{End: end, Pixel: px})
}

func find(runs []run, x int) int {
	lo, hi := 0, len(runs)
	for lo < hi {
		if m := int(uint(lo+hi) >> 1); int(runs[m].End) <= x {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

func alpha(opacity float64) uint32 { return uint32(math.Round(opacity * math.MaxUint8)) }

func masked(under, px, alpha uint32) uint32 {
	if px>>24 == 0 {
		return under
	}
	const widen = math.MaxUint16 / math.MaxUint8
	ma := alpha * widen
	keep := (math.MaxUint16 - (px>>24*widen)*ma/math.MaxUint16) * widen
	var out uint32
	for shift := 0; shift < 32; shift += 8 {
		out |= ((under>>shift&math.MaxUint8)*keep + (px>>shift&math.MaxUint8)*widen*ma) / math.MaxUint16 >> 8 & math.MaxUint8 << shift
	}
	return out
}

func flooded(under, px uint32) uint32 {
	switch a := px >> 24; a {
	case 0:
		return under
	case math.MaxUint8:
		return px
	default:
		return lanes(under, px, math.MaxUint8-a) | lanes(under>>8, px>>8, math.MaxUint8-a)<<8
	}
}

func lanes(under, px, keep uint32) uint32 {
	t := (under&konst.ByteLanes)*keep + konst.HalfLanes
	t += t >> 8 & konst.ByteLanes
	return (px&konst.ByteLanes + t>>8&konst.ByteLanes) & konst.ByteLanes
}

func (s *Screen) damage(r image.Rectangle) {
	r = r.Intersect(s.bounds)
	cells := image.Rect(r.Min.X/s.Cell.X, r.Min.Y/s.Cell.Y, (r.Max.X+s.Cell.X-1)/s.Cell.X, (r.Max.Y+s.Cell.Y-1)/s.Cell.Y)
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		for x := cells.Min.X; x < cells.Max.X; x++ {
			s.dirty[s.tileAt(x, y)] = true
		}
	}
}

func (s *Screen) pixels(cells image.Rectangle) image.Rectangle {
	return image.Rect(cells.Min.X*s.Cell.X, cells.Min.Y*s.Cell.Y, cells.Max.X*s.Cell.X, cells.Max.Y*s.Cell.Y)
}
