package graphics

import (
	"encoding/binary"
	"fmt"
	"image"
	"slices"
	"strconv"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type sixelColour struct {
	key   uint32
	count uint32
}

type sixelSpan struct {
	x, n int32
	next int32
	bits byte
}

type sixelBox struct {
	lo, hi int
	shift  uint
	span   uint32
}

type Sixel struct {
	plane    []int32
	ids      map[uint32]int32
	colours  []sixelColour
	order    []int32
	register []uint8
	palette  []uint32
	spans    []sixelSpan
	used     []uint8
	head     [graphics.SixelRegisters]int32
	tail     [graphics.SixelRegisters]int32
}

func (s *Sixel) Encode(dst []byte, img *image.RGBA, at Placement) []byte {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w == 0 || h == 0 {
		return dst
	}
	one := flat(img)
	if one != nil && one.Pix[3] == 0 {
		return dst
	}
	if one != nil {
		s.palette = append(s.palette[:0], percentKey(one.Pix))
	} else {
		s.index(img)
		s.quantise()
	}
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1bP0;1q\"1;1;%d;%d", at.Row+1, at.Col+1, w, h)
	for i, key := range s.palette {
		dst = fmt.Appendf(dst, "#%d;2;%d;%d;%d", i, key>>16, key>>8&0xff, key&0xff)
	}
	for y := 0; y < h; y += graphics.SixelBand {
		if y > 0 {
			dst = append(dst, '-')
		}
		rows := min(graphics.SixelBand, h-y)
		if one != nil {
			dst = sixelRun(append(dst, "#0"...), graphics.SixelZero+byte(1<<rows-1), w)
		} else {
			dst = s.band(dst, w, y, rows)
		}
	}
	return append(dst, "\x1b\\"...)
}

func percentKey(rgb []byte) uint32 {
	p := func(v byte) uint32 { return (uint32(v)*graphics.SixelPercent + 127) / 255 }
	return p(rgb[0])<<16 | p(rgb[1])<<8 | p(rgb[2])
}

func sixelRun(dst []byte, ch byte, n int) []byte {
	if n < graphics.SixelMinRepeat {
		for range n {
			dst = append(dst, ch)
		}
		return dst
	}
	dst = append(dst, '!')
	dst = strconv.AppendInt(dst, int64(n), 10)
	return append(dst, ch)
}

func (s *Sixel) index(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	s.plane = slices.Grow(s.plane[:0], w*h)[:w*h]
	if s.ids == nil {
		s.ids = map[uint32]int32{}
	}
	clear(s.ids)
	s.colours = s.colours[:0]
	last := ^binary.LittleEndian.Uint32(row(img, 0))
	id := int32(-1)
	for y := range h {
		pix, out := row(img, y), s.plane[y*w:(y+1)*w]
		for x := range out {
			p := pix[4*x : 4*x+4]
			if v := binary.LittleEndian.Uint32(p); v != last {
				last, id = v, s.lookup(p)
			}
			out[x] = id
			if id >= 0 {
				s.colours[id].count++
			}
		}
	}
}

func (s *Sixel) lookup(p []byte) int32 {
	if p[3] == 0 {
		return -1
	}
	key := percentKey(p)
	id, ok := s.ids[key]
	if !ok {
		id = int32(len(s.colours))
		s.ids[key] = id
		s.colours = append(s.colours, sixelColour{key: key})
	}
	return id
}

func (s *Sixel) quantise() {
	n := len(s.colours)
	s.register = slices.Grow(s.register[:0], n)[:n]
	s.palette = s.palette[:0]
	if n <= graphics.SixelRegisters {
		for i, c := range s.colours {
			s.register[i] = uint8(i)
			s.palette = append(s.palette, c.key)
		}
		return
	}
	s.order = s.order[:0]
	for i := range n {
		s.order = append(s.order, int32(i))
	}
	boxes := []sixelBox{s.measure(0, n)}
	for len(boxes) < graphics.SixelRegisters {
		widest := 0
		for i, b := range boxes {
			if b.span > boxes[widest].span {
				widest = i
			}
		}
		b := boxes[widest]
		if b.span == 0 {
			break
		}
		part := s.order[b.lo:b.hi]
		slices.SortFunc(part, func(i, j int32) int {
			return int(s.colours[i].key>>b.shift&0xff) - int(s.colours[j].key>>b.shift&0xff)
		})
		var total, half uint64
		for _, id := range part {
			total += uint64(s.colours[id].count)
		}
		split := b.hi - 1
		for i, id := range part[:len(part)-1] {
			if half += uint64(s.colours[id].count); 2*half >= total {
				split = b.lo + i + 1
				break
			}
		}
		boxes[widest] = s.measure(b.lo, split)
		boxes = append(boxes, s.measure(split, b.hi))
	}
	for r, b := range boxes {
		for _, id := range s.order[b.lo:b.hi] {
			s.register[id] = uint8(r)
		}
		s.palette = append(s.palette, s.weightedMedian(b))
	}
}

func (s *Sixel) measure(lo, hi int) sixelBox {
	b := sixelBox{lo: lo, hi: hi}
	for _, shift := range [3]uint{16, 8, 0} {
		least, most := uint32(0xff), uint32(0)
		for _, id := range s.order[lo:hi] {
			v := s.colours[id].key >> shift & 0xff
			least, most = min(least, v), max(most, v)
		}
		if most-least > b.span {
			b.span, b.shift = most-least, shift
		}
	}
	return b
}

func (s *Sixel) weightedMedian(b sixelBox) uint32 {
	var hist [3][graphics.SixelPercent + 1]uint64
	var total uint64
	for _, id := range s.order[b.lo:b.hi] {
		c := s.colours[id]
		for i, shift := range [3]uint{16, 8, 0} {
			hist[i][c.key>>shift&0xff] += uint64(c.count)
		}
		total += uint64(c.count)
	}
	var key uint32
	for i := range hist {
		var seen uint64
		v := 0
		for seen += hist[i][0]; 2*seen < total; seen += hist[i][v] {
			v++
		}
		key = key<<8 | uint32(v)
	}
	return key
}

func (s *Sixel) band(dst []byte, w, y0, rows int) []byte {
	s.spans = s.spans[:0]
	s.used = s.used[:0]
	for x := range w {
		var regs [graphics.SixelBand]uint8
		var bits [graphics.SixelBand]byte
		n := 0
		for r := range rows {
			id := s.plane[(y0+r)*w+x]
			if id < 0 {
				continue
			}
			reg := s.register[id]
			k := 0
			for k < n && regs[k] != reg {
				k++
			}
			if k == n {
				regs[n], bits[n] = reg, 0
				n++
			}
			bits[k] |= 1 << r
		}
		for k := range n {
			reg := regs[k]
			if s.head[reg] != 0 {
				if last := &s.spans[s.tail[reg]-1]; last.bits == bits[k] && last.x+last.n == int32(x) {
					last.n++
					continue
				}
			}
			s.spans = append(s.spans, sixelSpan{x: int32(x), n: 1, bits: bits[k]})
			at := int32(len(s.spans))
			if s.head[reg] == 0 {
				s.head[reg] = at
				s.used = append(s.used, reg)
			} else {
				s.spans[s.tail[reg]-1].next = at
			}
			s.tail[reg] = at
		}
	}
	for i, reg := range s.used {
		if i > 0 {
			dst = append(dst, '$')
		}
		dst = append(dst, '#')
		dst = strconv.AppendInt(dst, int64(reg), 10)
		x := int32(0)
		for at := s.head[reg]; at != 0; at = s.spans[at-1].next {
			span := s.spans[at-1]
			dst = sixelRun(dst, graphics.SixelZero, int(span.x-x))
			dst = sixelRun(dst, graphics.SixelZero+span.bits, int(span.n))
			x = span.x + span.n
		}
		s.head[reg] = 0
	}
	return dst
}
