package graphics

import (
	"encoding/binary"
	"math/bits"
	"slices"
	"strconv"

	"github.com/pehcastro/twind/internal/konst/graphics"
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
	edges     []byte
	runs      []byte
	tokens    []sixelToken
	used      []uint8
	registers [graphics.SixelRegisters]sixelRegister
}

type sixelToken struct {
	text, chars uint64
}

type sixelRegister struct {
	from, written int32
	bits          uint8
	live          bool
}

func (s *Sixel) Encode(dst []byte, rows [][]Run, at Placement) []byte {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return dst
	}
	if s.ids == nil {
		s.ids = map[uint32]int32{}
	}
	clear(s.ids)
	clear(s.cache[:])
	s.intern(0)
	s.colours = s.colours[:0]
	spans, stretches := s.rows[:0], s.stretches[:0]
	for y := 0; y < len(rows); {
		line, lo := rows[y], int32(len(stretches))
		for _, r := range line {
			slot := s.cache[r.Pixel*graphics.SixelCacheHash>>(32-graphics.SixelCacheBits)]
			if slot.pixel != r.Pixel {
				slot.id = s.intern(r.Pixel)
			}
			stretches = append(stretches, sixelStretch{end: r.End, id: slot.id})
		}
		first := y
		for y++; y < len(rows) && (&rows[y][0] == &line[0] || slices.Equal(rows[y], line)); y++ {
		}
		span, from := [2]int32{lo, int32(len(stretches))}, len(spans)
		spans = slices.Grow(spans, y-first)[:from+y-first]
		for i := from; i < len(spans); i++ {
			spans[i] = span
		}
	}
	s.rows, s.stretches = spans, stretches
	w, h := int(rows[0][len(rows[0])-1].End), len(rows)
	if len(s.colours) == 0 {
		return dst
	}
	s.quantise()
	arena := len(s.palette) * (w + graphics.SixelWord)
	s.runs = slices.Grow(s.runs[:0], arena)[:arena]
	s.tokens = slices.Grow(s.tokens, w+1-min(len(s.tokens), w+1))
	for n := len(s.tokens); n <= w; n++ {
		if n < graphics.SixelMinRepeat {
			s.tokens = append(s.tokens, sixelToken{chars: 1<<(graphics.SixelByteBits*n) - 1})
			continue
		}
		var text [graphics.SixelWord]byte
		size := len(strconv.AppendInt(append(text[:0], '!'), int64(n), 10))
		s.tokens = append(s.tokens, sixelToken{text: binary.LittleEndian.Uint64(text[:]), chars: (1<<graphics.SixelByteBits - 1) << (graphics.SixelByteBits * size)})
	}
	dst = strconv.AppendInt(append(dst, "\x1b["...), int64(at.Row+1), 10)
	dst = strconv.AppendInt(append(dst, ';'), int64(at.Col+1), 10)
	dst = strconv.AppendInt(append(dst, "H\x1bP0;1q\"1;1;"...), int64(w), 10)
	dst = strconv.AppendInt(append(dst, ';'), int64(h), 10)
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

func (s *Sixel) Quantised() bool { return len(s.colours) > graphics.SixelRegisters }

func percentKey(pixel uint32) uint32 {
	p := func(v uint32) uint32 { return (v&0xff*graphics.SixelPercent + 127) / 255 }
	return p(pixel)<<16 | p(pixel>>8)<<8 | p(pixel>>16)
}

func (s *Sixel) intern(pixel uint32) int32 {
	slot := &s.cache[pixel*graphics.SixelCacheHash>>(32-graphics.SixelCacheBits)]
	if pixel>>24 == 0 {
		*slot = sixelCached{pixel: pixel, id: -1}
		return -1
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
	s.palette = s.palette[:0]
	if n <= graphics.SixelRegisters {
		for _, c := range s.colours {
			s.palette = append(s.palette, c.key)
		}
		return
	}
	s.register = slices.Grow(s.register[:0], n)[:n]
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
	for i, st := range s.stretches {
		if st.id >= 0 {
			s.stretches[i].id = int32(s.register[st.id])
		}
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
	var spans [graphics.SixelBand][2]int32
	var cursor, held [graphics.SixelBand]int32
	var masks [graphics.SixelBand]uint8
	s.edges = slices.Grow(s.edges[:0], w+graphics.SixelWord)[:w+graphics.SixelWord]
	stride, groups := w+graphics.SixelWord, 0
	for r := range rows {
		g := 0
		for g < groups && spans[g] != s.rows[y0+r] {
			g++
		}
		if g == groups {
			spans[g], cursor[g], held[g] = s.rows[y0+r], s.rows[y0+r][0], -1
			for _, st := range s.stretches[spans[g][0] : spans[g][1]-1] {
				s.edges[st.end] |= 1 << g
			}
			groups++
		}
		masks[g] |= 1 << r
	}
	s.edges[0] |= 1<<groups - 1
	used, runs, tokens, stretches, edges := s.used[:0], s.runs, s.tokens, s.stretches, s.edges
	for x := 0; x < w; x += graphics.SixelWord {
		word := binary.LittleEndian.Uint64(edges[x:])
		if word == 0 {
			continue
		}
		binary.LittleEndian.PutUint64(edges[x:], 0)
		for ; word != 0; word &= word - 1 {
			b := bits.TrailingZeros64(word)
			at, g := int32(x+b/graphics.SixelByteBits), b%graphics.SixelByteBits
			reg := stretches[cursor[g]].id
			cursor[g]++
			if reg == held[g] {
				continue
			}
			if held[g] >= 0 {
				advance(runs, tokens, &s.registers[uint8(held[g])], masks[g], at)
			}
			if reg >= 0 {
				r := &s.registers[uint8(reg)]
				if !r.live {
					*r = sixelRegister{written: int32(len(used) * stride), live: true}
					used = append(used, uint8(reg))
				}
				advance(runs, tokens, r, masks[g], at)
			}
			held[g] = reg
		}
	}
	for i, reg := range used {
		if i > 0 {
			dst = append(dst, '$')
		}
		r := &s.registers[reg]
		if r.bits != 0 {
			advance(runs, tokens, r, r.bits, int32(w))
		}
		dst = append(strconv.AppendInt(append(dst, '#'), int64(reg), 10), runs[i*stride:r.written]...)
		*r = sixelRegister{}
	}
	s.used = used
	return dst
}

func advance(runs []byte, tokens []sixelToken, r *sixelRegister, mask uint8, at int32) {
	token := tokens[at-r.from]
	word := token.text | token.chars&(uint64(graphics.SixelZero+r.bits)*graphics.SixelSpread)
	binary.LittleEndian.PutUint64(runs[r.written:], word)
	r.written += int32(bits.Len64(word)+graphics.SixelByteBits-1) / graphics.SixelByteBits
	r.bits ^= mask
	r.from = at
}
