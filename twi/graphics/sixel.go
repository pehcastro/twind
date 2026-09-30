package graphics

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"strconv"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type sixelColour struct {
	key   uint32
	count uint32
}

type sixelStretch struct {
	end, id int32
}

type sixelCached struct {
	pixel uint32
	id    int32
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
	stretches []sixelStretch
	rows      [][2]int32
	ids       map[uint32]int32
	cache     [1 << graphics.SixelCacheBits]sixelCached
	colours   []sixelColour
	order     []int32
	register  []uint8
	palette   []uint32
	spans     []sixelSpan
	used      []uint8
	bits      [graphics.SixelRegisters]uint8
	changed   [graphics.SixelRegisters]bool
	head      [graphics.SixelRegisters]int32
	tail      [graphics.SixelRegisters]int32
}

func (s *Sixel) Encode(dst []byte, rows [][]byte, at Placement) []byte {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return dst
	}
	w, h := len(rows[0])/4, len(rows)
	s.scan(rows)
	if len(s.colours) == 0 {
		return dst
	}
	s.quantise()
	dst = fmt.Appendf(dst, "\x1b[%d;%dH\x1bP0;1q\"1;1;%d;%d", at.Row+1, at.Col+1, w, h)
	for i, key := range s.palette {
		dst = append(strconv.AppendInt(append(dst, '#'), int64(i), 10), ";2"...)
		for _, shift := range [3]uint{16, 8, 0} {
			dst = strconv.AppendUint(append(dst, ';'), uint64(key>>shift&0xff), 10)
		}
	}
	var from, to int
	for y := 0; y < h; y += graphics.SixelBand {
		rows := min(graphics.SixelBand, h-y)
		if y > 0 {
			dst = append(dst, '-')
			if slices.Equal(s.rows[y:y+rows], s.rows[y-graphics.SixelBand:y]) {
				dst = append(dst, dst[from:to]...)
				continue
			}
		}
		from = len(dst)
		dst = s.band(dst, w, y, rows)
		to = len(dst)
	}
	return append(dst, "\x1b\\"...)
}

func percentKey(pixel uint32) uint32 {
	p := func(v uint32) uint32 { return (v&0xff*graphics.SixelPercent + 127) / 255 }
	return p(pixel)<<16 | p(pixel>>8)<<8 | p(pixel>>16)
}

func sixelRun(dst []byte, ch byte, n int) []byte {
	if n < graphics.SixelMinRepeat {
		for range n {
			dst = append(dst, ch)
		}
		return dst
	}
	return append(strconv.AppendInt(append(dst, '!'), int64(n), 10), ch)
}

func (s *Sixel) scan(rows [][]byte) {
	w := len(rows[0]) / 4
	if s.ids == nil {
		s.ids = map[uint32]int32{}
	}
	clear(s.ids)
	clear(s.cache[:])
	s.colours = s.colours[:0]
	s.stretches = s.stretches[:0]
	s.rows = s.rows[:0]
	var above []byte
	for y, pix := range rows {
		if bytes.Equal(pix, above) {
			s.rows = append(s.rows, s.rows[y-1])
			continue
		}
		lo := int32(len(s.stretches))
		for x := 0; x < w; {
			v, end := binary.LittleEndian.Uint32(pix[4*x:]), x+1
			for end+graphics.SixelSkip <= w && binary.LittleEndian.Uint32(pix[4*end:]) == v {
				block := pix[4*end-4 : 4*(end+graphics.SixelSkip)]
				if !bytes.Equal(block[4:], block[:len(block)-4]) {
					break
				}
				end += graphics.SixelSkip
			}
			for end < w && binary.LittleEndian.Uint32(pix[4*end:]) == v {
				end++
			}
			s.stretches = append(s.stretches, sixelStretch{end: int32(end), id: s.lookup(v)})
			x = end
		}
		s.rows = append(s.rows, [2]int32{lo, int32(len(s.stretches))})
		above = pix
	}
}

func (s *Sixel) lookup(pixel uint32) int32 {
	if pixel>>24 == 0 {
		return -1
	}
	slot := &s.cache[pixel*graphics.SixelCacheHash>>(32-graphics.SixelCacheBits)]
	if slot.pixel == pixel {
		return slot.id
	}
	key := percentKey(pixel)
	id, ok := s.ids[key]
	if !ok {
		id = int32(len(s.colours))
		s.ids[key] = id
		s.colours = append(s.colours, sixelColour{key: key})
	}
	*slot = sixelCached{pixel: pixel, id: id}
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
	for _, r := range s.rows {
		x := int32(0)
		for _, st := range s.stretches[r[0]:r[1]] {
			if st.id >= 0 {
				s.colours[st.id].count += uint32(st.end - x)
			}
			x = st.end
		}
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
	var cursor, ends [graphics.SixelBand]int32
	var held [graphics.SixelBand]int
	for r := range rows {
		cursor[r], held[r] = s.rows[y0+r][0]-1, -1
	}
	for x := int32(0); x < int32(w); {
		var changed [2 * graphics.SixelBand]uint8
		n, end := 0, int32(w)
		for r := range rows {
			if ends[r] == x {
				cursor[r]++
				st := s.stretches[cursor[r]]
				ends[r] = st.end
				reg := -1
				if st.id >= 0 {
					reg = int(s.register[st.id])
				}
				if reg != held[r] {
					for _, touched := range [2]int{held[r], reg} {
						if touched >= 0 {
							s.bits[touched] ^= 1 << r
							if !s.changed[touched] {
								s.changed[touched] = true
								changed[n] = uint8(touched)
								n++
							}
						}
					}
					held[r] = reg
				}
			}
			end = min(end, ends[r])
		}
		for _, reg := range changed[:n] {
			s.changed[reg] = false
			if s.head[reg] != 0 {
				if last := &s.spans[s.tail[reg]-1]; last.n == 0 {
					last.n = x - last.x
				}
			}
			if s.bits[reg] == 0 {
				continue
			}
			s.spans = append(s.spans, sixelSpan{x: x, bits: s.bits[reg]})
			at := int32(len(s.spans))
			if s.head[reg] == 0 {
				s.head[reg] = at
				s.used = append(s.used, reg)
			} else {
				s.spans[s.tail[reg]-1].next = at
			}
			s.tail[reg] = at
		}
		x = end
	}
	for i, reg := range s.used {
		if i > 0 {
			dst = append(dst, '$')
		}
		dst = strconv.AppendInt(append(dst, '#'), int64(reg), 10)
		if last := &s.spans[s.tail[reg]-1]; last.n == 0 {
			last.n = int32(w) - last.x
		}
		s.bits[reg] = 0
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
