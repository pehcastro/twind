package graphics

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"slices"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

type huffman struct {
	lengths [graphics.DeflateFixedLiterals]uint8
	codes   [graphics.DeflateFixedLiterals]uint16
}

type flatSpan struct{ x0, x1 int }

type rowFilter int

const (
	rawRows rowFilter = iota
	pngUpRows
)

type deflater struct {
	same      []bool
	spans     []flatSpan
	hint      []flatSpan
	hi        int
	tokens    []uint64
	used      []int
	rle       []uint32
	one       [1][]byte
	bpp       int
	sum       uint64
	pos       uint64
	moment    uint64
	runTok    uint64
	runLen    int
	up        uint64
	upLeft    uint64
	upRight   uint64
	left      uint64
	dists     [graphics.DistanceSlots]int
	litFreq   [graphics.DeflateLiterals]int32
	distFreq  [graphics.DeflateDistances]int32
	out       []byte
	acc       uint64
	nacc      uint
	freq      [graphics.TokenMask + 1]int32
	sym       [graphics.TokenMask + 1]uint64
	codeLen   [graphics.DeflateCodeLengths]int32
	order     [graphics.DeflateLiterals]uint64
	depth     [graphics.DeflateLiterals]int32
	lengths   [graphics.DeflateLiterals + graphics.DeflateDistances]uint8
	litCode   huffman
	distCode  huffman
	lenCode   huffman
	fixedLit  huffman
	fixedDist huffman
}

func (d *deflater) prepare(rows [][]byte) (prepared [][]byte, flat bool) {
	d.same = slices.Grow(d.same[:0], len(rows))[:len(rows)]
	flat = true
	for y, r := range rows {
		d.same[y] = y > 0 && bytes.Equal(r, rows[y-1])
		flat = flat && (d.same[y] || y == 0 && bytes.Equal(r[4:], r[:len(r)-4]))
	}
	if !flat {
		return rows, false
	}
	d.one = [1][]byte{rows[0][:4]}
	d.same = d.same[:1]
	return d.one[:], true
}

func pixel(r []byte, x int) uint32 { return binary.LittleEndian.Uint32(r[4*x : 4*x+4]) }

func straight(p uint32) uint32 {
	a := p >> 24
	if a == 0 || a == 0xff {
		return p
	}
	return p&0xff*0xffff/a>>8 | (p>>8&0xff*0xffff/a>>8)<<8 | (p>>16&0xff*0xffff/a>>8)<<16 | a<<24
}

func residual(p, above uint32) uint32 {
	v, u := straight(p), straight(above)
	return ((v | graphics.ByteHighBits) - (u &^ graphics.ByteHighBits)) ^ ((v ^ ^u) & graphics.ByteHighBits)
}

func (d *deflater) zlib(dst []byte, rows [][]byte, filter rowFilter) ([]byte, int) {
	if out, ok := d.encode(dst, rows, 3, filter); ok {
		return out, 3
	}
	out, _ := d.encode(dst, rows, 4, filter)
	return out, 4
}

func (d *deflater) encode(dst []byte, rows [][]byte, bpp int, filter rowFilter) ([]byte, bool) {
	const m = graphics.AdlerModulus
	w := len(rows[0]) / 4
	lead := uint64(0)
	if filter == pngUpRows {
		lead = 1
	}
	d.bpp = bpp
	n := int(lead) + w*bpp
	mask := uint32(1)<<(8*bpp) - 1
	up := filter == rawRows && n+bpp <= graphics.DeflateWindow
	d.tokens, d.runTok, d.runLen = d.tokens[:0], 0, 0
	clear(d.freq[:])
	d.used = d.used[:0]
	d.dists = [graphics.DistanceSlots]int{n, n + bpp, max(n-bpp, 1), bpp}
	var slots [graphics.DistanceSlots]uint64
	for id, s := range d.dists {
		slots[id] = uint64(graphics.TokenDistance+slices.Index(d.dists[:], s))<<graphics.TokenSlot | graphics.TokenNone<<(2*graphics.TokenSlot)
	}
	d.up, d.upLeft, d.upRight, d.left = slots[0], slots[1], slots[2], slots[3]
	var a, b, sum, weight uint64 = 1, 0, 0, 0
	for y, r := range rows {
		var above []byte
		if y > 0 && (up || filter == pngUpRows) {
			above = rows[y-1]
		}
		same := d.same[y] && above != nil
		if same && filter == rawRows {
			x := 0
			for _, s := range d.spans {
				if end := min(s.x0+1, w); end > x {
					d.run(d.up, (end-x)*bpp)
				}
				if s.x1 > s.x0+1 {
					d.run(d.left, (s.x1-s.x0-1)*bpp)
				}
				x = s.x1
			}
		} else {
			d.sum, d.pos, d.moment = 0, 0, 0
			ok := true
			switch {
			case filter == rawRows:
				ok = d.rawRow(r, above, w, mask)
			case same:
				d.literalByte(graphics.PNGUpFilter)
				d.upSpan(0, w, 0)
			default:
				d.literalByte(graphics.PNGUpFilter)
				ok = d.upRow(r, above, w, mask)
			}
			if !ok {
				return nil, false
			}
			s, moment := d.sum+graphics.PNGUpFilter*lead, lead*d.sum+uint64(bpp)*d.pos/2+d.moment
			sum, weight = s%m, (uint64(n)*s%m+m-moment%m)%m
		}
		b = (b + uint64(n)%m*a + weight) % m
		a = (a + sum) % m
	}
	d.flush()
	d.out = dst
	d.out = append(d.out, graphics.ZlibHeader...)
	d.block()
	return binary.BigEndian.AppendUint32(d.out, uint32(b<<16|a)), true
}

func (d *deflater) rawRow(cur, above []byte, w int, mask uint32) bool {
	d.hint, d.spans, d.hi = d.spans, d.hint[:0], 0
	bpp := d.bpp
	for x := 0; x < w; {
		p, e := pixel(cur, x), x+1
		if above != nil && pixel(above, x) == p {
			b := w
			if h := d.hintAt(x); h.x1 > 0 {
				b = h.x0
			}
			if b > x {
				b = equalRun(cur, above, x+1, b, 0)
				if q := pixel(cur, b-1); b < w && pixel(cur, b) == q {
					for b > x && pixel(cur, b-1) == q {
						b--
					}
				}
			}
			if b > x {
				var sum, pos, moment uint64
				for i := x; i < b; i++ {
					s, m := sums(straight(pixel(cur, i)) & mask)
					sum, pos, moment = sum+s, pos+s*uint64(2*i), moment+m
				}
				d.sum, d.pos, d.moment = d.sum+sum, d.pos+pos, d.moment+moment
				d.run(d.up, (b-x)*bpp)
				x = b
				continue
			}
		}
		if bpp == 3 && p>>24 != 0xff {
			return false
		}
		if e < w && pixel(cur, e) == p {
			e = equalRun(cur[4:], cur, e, w-1, d.hintAt(x).x1-1) + 1
		}
		c := straight(p) & mask
		d.adler(uint64(x), uint64(e), c)
		last := e
		if e-x >= graphics.DeflateFlatSpan {
			last = x + 1
			d.spans = append(d.spans, flatSpan{x, e})
		}
		switch {
		case above == nil:
			d.literal(c)
			last = x + 1
		case pixel(above, x) == p:
			d.run(d.up, bpp)
		case x > 0 && pixel(above, x-1) == p:
			d.run(d.upLeft, bpp)
		case x+1 < w && pixel(above, x+1) == p:
			d.run(d.upRight, bpp)
		default:
			d.literal(c)
		}
		for i := x + 1; i < last; i++ {
			if pixel(above, i) == p {
				d.run(d.up, bpp)
			} else {
				d.run(d.left, bpp)
			}
		}
		if e > last {
			d.run(d.left, (e-last)*bpp)
		}
		x = e
	}
	d.spans = append(d.spans, flatSpan{w, w})
	return true
}

func (d *deflater) upRow(cur, above []byte, w int, mask uint32) bool {
	d.hint, d.spans, d.hi = d.spans, d.hint[:0], 0
	if above == nil {
		for x := 0; x < w; {
			p := pixel(cur, x)
			if d.bpp == 3 && p>>24 != 0xff {
				return false
			}
			e := equalRun(cur[4:], cur, x, w-1, 0) + 1
			d.upSpan(x, e, straight(p)&mask)
			x = e
		}
		return true
	}
	x0, x := 0, 0
	p, u := pixel(cur, 0), pixel(above, 0)
	v0 := residual(p, u)
	for {
		if d.bpp == 3 && p>>24 != 0xff {
			return false
		}
		e := x + 1
		switch {
		case p == u:
			e = equalRun(cur, above, e, w, d.hintAt(x).x1)
		case e < w && pixel(cur, e) == p && pixel(above, e) == u:
			g := d.hintAt(x).x1
			e = min(equalRun(cur[4:], cur, e, w-1, g-1), equalRun(above[4:], above, e, w-1, g-1)) + 1
		}
		if e >= w {
			break
		}
		p, u = pixel(cur, e), pixel(above, e)
		if v := residual(p, u); v != v0 {
			d.hintSpan(x0, e, v0&mask)
			x0, v0 = e, v
		}
		x = e
	}
	d.hintSpan(x0, w, v0&mask)
	return true
}

func (d *deflater) hintSpan(x0, x1 int, c uint32) {
	if x1-x0 >= graphics.DeflateFlatSpan {
		d.spans = append(d.spans, flatSpan{x0, x1})
	}
	d.upSpan(x0, x1, c)
}

func (d *deflater) hintAt(x int) flatSpan {
	for d.hi < len(d.hint) && d.hint[d.hi].x1 <= x {
		d.hi++
	}
	if d.hi == len(d.hint) {
		return flatSpan{}
	}
	return d.hint[d.hi]
}

func equalRun(a, b []byte, x, w, guess int) int {
	if g := min(guess, w) - graphics.RunGuard; g-x >= graphics.RunGuess && bytes.Equal(a[4*x:4*g], b[4*x:4*g]) {
		x = g
	}
	for end := min(x+graphics.RunBlock, w); x < end; x++ {
		if pixel(a, x) != pixel(b, x) {
			return x
		}
	}
	step := graphics.RunBlock
	for ; x+step <= w && bytes.Equal(a[4*x:4*(x+step)], b[4*x:4*(x+step)]); step *= 2 {
		x += step
	}
	for step /= 2; step > 0; step /= 2 {
		if x+step <= w && bytes.Equal(a[4*x:4*(x+step)], b[4*x:4*(x+step)]) {
			x += step
		}
	}
	return x
}

func (d *deflater) upSpan(x0, x1 int, c uint32) {
	d.adler(uint64(x0), uint64(x1), c)
	d.literal(c)
	if x1 > x0+1 {
		d.run(d.left, (x1-x0-1)*d.bpp)
	}
}

func sums(c uint32) (uint64, uint64) {
	return uint64(c&0xff + c>>8&0xff + c>>16&0xff + c>>24), uint64(c>>8&0xff + c>>15&0x1fe + c>>24*3)
}

func (d *deflater) adler(x0, x1 uint64, c uint32) {
	s, m := sums(c)
	d.sum += (x1 - x0) * s
	d.pos += s * (x1 - x0) * (x0 + x1 - 1)
	d.moment += (x1 - x0) * m
}

func byteToken(c uint32) uint64 {
	return uint64(c) | graphics.TokenNone<<graphics.TokenSlot | graphics.TokenNone<<(2*graphics.TokenSlot)
}

func (d *deflater) literal(c uint32) {
	if d.runLen > 0 {
		d.flush()
	}
	d.tokens = append(d.tokens, uint64(c&0xff)|uint64(c>>8&0xff)<<graphics.TokenSlot|uint64(c>>16&0xff)<<(2*graphics.TokenSlot))
	d.freq[c&0xff]++
	d.freq[c>>8&0xff]++
	d.freq[c>>16&0xff]++
	if d.bpp == 4 {
		d.literalByte(c >> 24)
	}
}

func (d *deflater) literalByte(c uint32) {
	d.flush()
	d.tokens = append(d.tokens, byteToken(c))
	d.freq[c]++
}

func (d *deflater) match(length int, repeat int32) {
	if d.freq[graphics.TokenLength+length] == 0 {
		d.used = append(d.used, length)
	}
	d.tokens = append(d.tokens, graphics.TokenLength+uint64(length)|d.runTok|uint64(repeat-1)<<graphics.TokenRepeat)
	d.freq[graphics.TokenLength+length] += repeat
	d.freq[d.runTok>>graphics.TokenSlot&graphics.TokenMask] += repeat
}

func (d *deflater) run(dist uint64, length int) {
	if dist != d.runTok {
		if d.runLen > 0 {
			d.flush()
		}
		d.runTok = dist
	}
	d.runLen += length
}

func (d *deflater) flush() {
	if d.runLen == 0 {
		return
	}
	if d.runLen <= graphics.DeflateMaxMatch {
		d.match(d.runLen, 1)
		d.runLen = 0
		return
	}
	full, r := d.runLen/graphics.DeflateMaxMatch, d.runLen%graphics.DeflateMaxMatch
	d.runLen = 0
	three := r > 0 && r < graphics.DeflateMinMatch
	if three {
		full--
		r += graphics.DeflateMaxMatch - graphics.DeflateMinMatch
	}
	if full > 0 {
		d.match(graphics.DeflateMaxMatch, int32(full))
	}
	if r > 0 {
		d.match(r, 1)
	}
	if three {
		d.match(graphics.DeflateMinMatch, 1)
	}
}

func lengthCode(l int) (int, uint, uint32) {
	if l == graphics.DeflateMaxMatch {
		return graphics.DeflateLiterals - 1, 0, 0
	}
	v := l - graphics.DeflateMinMatch
	e := max(bits.Len(uint(v))-3, 0)
	return graphics.DeflateEndOfBlock + 1 + 4*e + v>>e, uint(e), uint32(v & (1<<e - 1))
}

func distCode(dist int) (int, uint, uint32) {
	v := dist - 1
	e := max(bits.Len(uint(v))-2, 0)
	return 2*e + v>>e, uint(e), uint32(v & (1<<e - 1))
}

func (d *deflater) put(v uint32, n uint) {
	d.acc |= uint64(v) << d.nacc
	d.nacc += n
	if d.nacc >= 32 {
		d.out = binary.LittleEndian.AppendUint32(d.out, uint32(d.acc))
		d.acc >>= 32
		d.nacc -= 32
	}
}

func (h *huffman) symbol(s int, extra uint, v uint32) uint64 {
	l := uint(h.lengths[s])
	return uint64(h.codes[s]) | uint64(v)<<l | uint64(l+extra)<<graphics.SymbolLength
}

func (d *deflater) block() {
	freq := &d.freq
	lit, dist := d.litFreq[:], d.distFreq[:]
	clear(dist)
	copy(lit, freq[:graphics.DeflateLiterals])
	symbols := 0
	for _, f := range lit {
		symbols += int(f)
	}
	for _, l := range d.used {
		f := freq[graphics.TokenLength+l]
		lc, _, _ := lengthCode(l)
		lit[lc] += f
		symbols += int(f)
	}
	for id, s := range d.dists {
		if f := freq[graphics.TokenDistance+id]; f > 0 {
			dc, _, _ := distCode(s)
			dist[dc] += f
		}
	}
	lit[graphics.DeflateEndOfBlock]++
	d.acc, d.nacc = 0, 0
	lh, dh := &d.litCode, &d.distCode
	if symbols < graphics.DeflateFixedTokens {
		lh, dh = d.fixedCodes()
		d.put(graphics.DeflateFixedFinal, 3)
	} else {
		d.dynamicHeader(lit, dist)
	}
	sym := &d.sym
	for s, f := range lit[:graphics.DeflateEndOfBlock+1] {
		if f > 0 {
			sym[s] = lh.symbol(s, 0, 0)
		}
	}
	for id, s := range d.dists {
		if freq[graphics.TokenDistance+id] > 0 {
			dc, de, dv := distCode(s)
			sym[graphics.TokenDistance+id] = dh.symbol(dc, de, dv)
		}
	}
	for _, l := range d.used {
		lc, le, lv := lengthCode(l)
		sym[graphics.TokenLength+l] = lh.symbol(lc, le, lv)
	}
	pos := len(d.out)
	buf := slices.Grow(d.out, graphics.DeflateTokenBytes*(symbols+2))
	buf = buf[:cap(buf)]
	binary.LittleEndian.PutUint64(buf[pos:], d.acc)
	pos += int(d.nacc >> 3)
	acc, nacc := d.acc>>(d.nacc&^7), d.nacc&7
	d.tokens = append(d.tokens, byteToken(graphics.DeflateEndOfBlock))
	tokens := d.tokens
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		a, b, c := sym[t&graphics.TokenMask], sym[t>>graphics.TokenSlot&graphics.TokenMask], sym[t>>(2*graphics.TokenSlot)&graphics.TokenMask]
		la, lb := a>>graphics.SymbolLength, b>>graphics.SymbolLength
		acc |= (a&graphics.SymbolBits | b&graphics.SymbolBits<<(la&graphics.ShiftMask) | c&graphics.SymbolBits<<((la+lb)&graphics.ShiftMask)) << (nacc & graphics.ShiftMask)
		nacc += uint(la + lb + c>>graphics.SymbolLength)
		binary.LittleEndian.PutUint64(buf[pos:], acc)
		pos += int(nacc >> 3)
		acc >>= nacc & graphics.ByteShiftMask
		nacc &= 7
		if t >= 1<<graphics.TokenRepeat {
			tokens[i] -= 1 << graphics.TokenRepeat
			i--
		}
	}
	d.out = buf[:pos+int(nacc+7)/8]
}

func (d *deflater) dynamicHeader(lit, dist []int32) {
	d.build(&d.litCode, lit, graphics.DeflateMaxBits)
	d.build(&d.distCode, dist, graphics.DeflateMaxBits)
	nlit, ndist := graphics.DeflateLiterals, graphics.DeflateDistances
	for d.litCode.lengths[nlit-1] == 0 {
		nlit--
	}
	for d.distCode.lengths[ndist-1] == 0 {
		ndist--
	}
	seq := append(append(d.lengths[:0], d.litCode.lengths[:nlit]...), d.distCode.lengths[:ndist]...)
	clear(d.codeLen[:])
	d.rle = d.rle[:0]
	emit := func(sym, extra, n int) {
		d.codeLen[sym]++
		d.rle = append(d.rle, uint32(sym|extra<<8|n<<16))
	}
	for i := 0; i < len(seq); {
		v, r := seq[i], 1
		for i+r < len(seq) && seq[i+r] == v {
			r++
		}
		i += r
		if v == 0 {
			for ; r > graphics.DeflateZerosMax; r -= min(r, graphics.DeflateLongZerosMax) {
				emit(graphics.DeflateLongZeros, min(r, graphics.DeflateLongZerosMax)-graphics.DeflateZerosMax-1, bits.Len(graphics.DeflateLongZerosMax-graphics.DeflateZerosMax-1))
			}
			if r >= graphics.DeflateRepeatMin {
				emit(graphics.DeflateZeros, r-graphics.DeflateRepeatMin, bits.Len(graphics.DeflateZerosMax-graphics.DeflateRepeatMin))
				r = 0
			}
		} else {
			emit(int(v), 0, 0)
			for r--; r >= graphics.DeflateRepeatMin; r -= min(r, graphics.DeflateRepeatMax) {
				emit(graphics.DeflateRepeat, min(r, graphics.DeflateRepeatMax)-graphics.DeflateRepeatMin, bits.Len(graphics.DeflateRepeatMax-graphics.DeflateRepeatMin))
			}
		}
		for ; r > 0; r-- {
			emit(int(v), 0, 0)
		}
	}
	d.build(&d.lenCode, d.codeLen[:], graphics.DeflateCodeLengthBits)
	order := graphics.DeflateCodeLengthOrder
	ncl := len(order)
	for d.lenCode.lengths[order[ncl-1]] == 0 {
		ncl--
	}
	d.put(graphics.DeflateDynamicFinal, 3)
	d.put(uint32(nlit-graphics.DeflateEndOfBlock-1), 5)
	d.put(uint32(ndist-1), 5)
	d.put(uint32(ncl-4), 4)
	for i := range ncl {
		d.put(uint32(d.lenCode.lengths[order[i]]), 3)
	}
	for _, r := range d.rle {
		d.put(uint32(d.lenCode.codes[r&0xff]), uint(d.lenCode.lengths[r&0xff]))
		d.put(r>>8&0xff, uint(r>>16))
	}
}

func (d *deflater) build(h *huffman, freq []int32, limit int) {
	order := d.order[:0]
	for s, f := range freq {
		if f > 0 {
			order = append(order, uint64(f)<<16|uint64(s))
		}
	}
	for s := 0; len(order) < 2; s++ {
		if freq[s] == 0 {
			freq[s] = 1
			order = append(order, 1<<16|uint64(s))
		}
	}
	slices.Sort(order)
	depth := d.depth[:len(order)]
	for i, o := range order {
		depth[i] = int32(o >> 16)
	}
	minimumRedundancy(depth)
	var count [graphics.DeflateMaxBits + 1]int
	excess := -1 << limit
	for _, l := range depth {
		l = min(l, int32(limit))
		count[l]++
		excess += 1 << (limit - int(l))
	}
	for ; excess > 0; excess-- {
		b := limit - 1
		for count[b] == 0 {
			b--
		}
		count[b]--
		count[b+1] += 2
		count[limit]--
	}
	clear(h.lengths[:len(freq)])
	i := 0
	for l := limit; l > 0; l-- {
		for range count[l] {
			h.lengths[order[i]&0xffff] = uint8(l)
			i++
		}
	}
	h.canonical(len(freq))
}

func (h *huffman) canonical(n int) {
	var count, next [graphics.DeflateMaxBits + 1]uint16
	for _, l := range h.lengths[:n] {
		count[l]++
	}
	count[0] = 0
	for l := 1; l <= graphics.DeflateMaxBits; l++ {
		next[l] = (next[l-1] + count[l-1]) << 1
	}
	for s, l := range h.lengths[:n] {
		if l > 0 {
			h.codes[s] = bits.Reverse16(next[l]) >> (16 - l)
			next[l]++
		}
	}
}

func (d *deflater) fixedCodes() (*huffman, *huffman) {
	if d.fixedLit.lengths[0] == 0 {
		for s := range d.fixedLit.lengths {
			d.fixedLit.lengths[s] = 8
			if s >= graphics.DeflateFixedNineFrom && s < graphics.DeflateEndOfBlock {
				d.fixedLit.lengths[s] = 9
			}
			if s >= graphics.DeflateEndOfBlock && s < graphics.DeflateFixedEightFrom {
				d.fixedLit.lengths[s] = 7
			}
		}
		for s := range graphics.DeflateDistances {
			d.fixedDist.lengths[s] = 5
		}
		d.fixedLit.canonical(graphics.DeflateFixedLiterals)
		d.fixedDist.canonical(graphics.DeflateDistances)
	}
	return &d.fixedLit, &d.fixedDist
}

func minimumRedundancy(a []int32) {
	n := int32(len(a))
	a[0] += a[1]
	root, leaf := int32(0), int32(2)
	for next := int32(1); next < n-1; next++ {
		if leaf >= n || a[root] < a[leaf] {
			a[next], a[root] = a[root], next
			root++
		} else {
			a[next] = a[leaf]
			leaf++
		}
		if leaf >= n || (root < next && a[root] < a[leaf]) {
			a[next] += a[root]
			a[root] = next
			root++
		} else {
			a[next] += a[leaf]
			leaf++
		}
	}
	a[n-2] = 0
	for next := n - 3; next >= 0; next-- {
		a[next] = a[a[next]] + 1
	}
	avail, used, depth, next := 1, 0, int32(0), n-1
	for root = n - 2; avail > 0; avail, used, depth = 2*used, 0, depth+1 {
		for root >= 0 && a[root] == depth {
			used++
			root--
		}
		for ; avail > used; avail-- {
			a[next] = depth
			next--
		}
	}
}
