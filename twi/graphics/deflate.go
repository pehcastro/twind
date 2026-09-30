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
	tokens    []uint32
	rle       []uint32
	one       [1][]byte
	cur       []byte
	above     []byte
	filter    rowFilter
	bpp       int
	lead      int
	n         int
	sum       uint64
	moment    uint64
	runDist   int
	runLen    int
	out       []byte
	acc       uint64
	nacc      uint
	lit       [graphics.DeflateLiterals]int32
	dist      [graphics.DeflateDistances]int32
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

func pixel(r []byte, x int) uint32 { return binary.LittleEndian.Uint32(r[4*x:]) }

func equalRun(a, b []byte, x, w int) int {
	for end := min(x+graphics.ScalarRun, w); x < end; x++ {
		if pixel(a, x) != pixel(b, x) {
			return x
		}
	}
	step := 1
	for ; x+step <= w && bytes.Equal(a[4*x:4*(x+step)], b[4*x:4*(x+step)]); step *= 2 {
		x += step
	}
	for ; step > 0; step /= 2 {
		if x+step <= w && bytes.Equal(a[4*x:4*(x+step)], b[4*x:4*(x+step)]) {
			x += step
		}
	}
	return x
}

func straight(p uint32) uint32 {
	a := p >> 24
	if a == 0 || a == 0xff {
		return p
	}
	return p&0xff*0xffff/a>>8 | (p>>8&0xff*0xffff/a>>8)<<8 | (p>>16&0xff*0xffff/a>>8)<<16 | a<<24
}

func (d *deflater) residual(x int) uint32 {
	v, u := straight(pixel(d.cur, x)), straight(pixel(d.above, x))
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
	d.filter, d.bpp, d.lead = filter, bpp, 0
	if filter == pngUpRows {
		d.lead = 1
	}
	d.n = d.lead + w*bpp
	mask := uint32(1)<<(8*bpp) - 1
	up := filter == rawRows && d.n+bpp <= graphics.DeflateWindow
	d.tokens, d.runDist, d.runLen = d.tokens[:0], 0, 0
	clear(d.lit[:])
	clear(d.dist[:])
	var a, b, sum, weight uint64 = 1, 0, 0, 0
	for y, r := range rows {
		d.cur, d.above = r, nil
		if y > 0 && (up || filter == pngUpRows) {
			d.above = rows[y-1]
		}
		same := d.same[y] && d.above != nil
		if !same || filter == pngUpRows {
			d.sum, d.moment = 0, 0
			if filter == pngUpRows {
				d.sum = graphics.PNGUpFilter
				d.literals(graphics.PNGUpFilter, 1)
			}
			if same {
				d.span(0, w, 0)
			} else {
				d.spans = d.spans[:0]
				for x := 0; x < w; {
					p := pixel(d.cur, x)
					if bpp == 3 && p>>24 != 0xff {
						return nil, false
					}
					v, e := straight(p), x+1
					switch {
					case d.above == nil || filter == rawRows:
						e = equalRun(d.cur[4:], d.cur, x, w-1) + 1
					case p == pixel(d.above, x):
						v, e = 0, equalRun(d.cur, d.above, x, w)
					default:
						v = d.residual(x)
						for e < w && d.residual(e) == v {
							e = max(e+1, min(equalRun(d.cur[4:], d.cur, e, w-1), equalRun(d.above[4:], d.above, e, w-1))+1)
						}
					}
					d.span(x, e, v&mask)
					x = e
				}
				d.spans = append(d.spans, flatSpan{w, w})
			}
			sum, weight = d.sum%m, (uint64(d.n)*d.sum%m+m-d.moment%m)%m
		} else {
			x := 0
			for _, s := range d.spans {
				if end := min(s.x0+1, w); end > x {
					d.run(d.n, (end-x)*bpp)
				}
				if s.x1 > s.x0+1 {
					d.run(bpp, (s.x1-s.x0-1)*bpp)
				}
				x = s.x1
			}
		}
		b = (b + uint64(d.n)%m*a + weight) % m
		a = (a + sum) % m
	}
	d.flush()
	d.out = dst
	d.out = append(d.out, graphics.ZlibHeader...)
	d.block()
	return binary.BigEndian.AppendUint32(d.out, uint32(b<<16|a)), true
}

func (d *deflater) span(x0, x1 int, c uint32) {
	count := uint64(x1 - x0)
	c0, c1, c2, c3 := uint64(c&0xff), uint64(c>>8&0xff), uint64(c>>16&0xff), uint64(c>>24)
	d.sum += count * (c0 + c1 + c2 + c3)
	d.moment += (c0+c1+c2+c3)*(count*uint64(d.lead)+uint64(d.bpp)*(count*uint64(x0+x1-1)/2)) + count*(c1+2*c2+3*c3)
	last := x1
	if d.filter == pngUpRows || x1-x0 >= graphics.DeflateFlatSpan {
		last = x0 + 1
	}
	copies := d.filter == rawRows && d.above != nil
	for x := x0; x < last; x++ {
		p := pixel(d.cur, x)
		switch {
		case copies && p == pixel(d.above, x):
			d.run(d.n, d.bpp)
		case x > x0:
			d.run(d.bpp, d.bpp)
		case copies && x > 0 && p == pixel(d.above, x-1):
			d.run(d.n+d.bpp, d.bpp)
		case copies && 4*x+4 < len(d.cur) && p == pixel(d.above, x+1):
			d.run(d.n-d.bpp, d.bpp)
		default:
			d.literals(c, d.bpp)
		}
	}
	if x1 > last {
		d.run(d.bpp, (x1-last)*d.bpp)
		if d.filter == rawRows {
			d.spans = append(d.spans, flatSpan{x0, x1})
		}
	}
}

func (d *deflater) literals(c uint32, n int) {
	d.flush()
	for j := range n {
		v := byte(c >> (8 * j))
		d.tokens = append(d.tokens, uint32(v))
		d.lit[v]++
	}
}

func (d *deflater) run(dist, length int) {
	if dist != d.runDist {
		d.flush()
		d.runDist = dist
	}
	d.runLen += length
}

func (d *deflater) flush() {
	if d.runLen == 0 {
		return
	}
	dc, _, _ := distCode(d.runDist)
	for n := d.runLen; n > 0; {
		take := min(n, graphics.DeflateMaxMatch)
		if n-take > 0 && n-take < graphics.DeflateMinMatch {
			take = n - graphics.DeflateMinMatch
		}
		lc, _, _ := lengthCode(take)
		d.lit[lc]++
		d.dist[dc]++
		d.tokens = append(d.tokens, uint32(take)<<16|uint32(d.runDist))
		n -= take
	}
	d.runLen = 0
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

func (h *huffman) put(d *deflater, s int) {
	d.put(uint32(h.codes[s]), uint(h.lengths[s]))
}

func (d *deflater) block() {
	d.lit[graphics.DeflateEndOfBlock]++
	d.acc, d.nacc = 0, 0
	lit, dist := &d.litCode, &d.distCode
	if len(d.tokens) < graphics.DeflateFixedTokens {
		lit, dist = d.fixedCodes()
		d.put(graphics.DeflateFixedFinal, 3)
	} else {
		d.dynamicHeader()
	}
	for _, t := range d.tokens {
		if t < 1<<16 {
			lit.put(d, int(t))
			continue
		}
		lc, le, lv := lengthCode(int(t >> 16))
		dc, de, dv := distCode(int(t & 0xffff))
		d.put(uint32(lit.codes[lc])|lv<<lit.lengths[lc], uint(lit.lengths[lc])+le)
		d.put(uint32(dist.codes[dc])|dv<<dist.lengths[dc], uint(dist.lengths[dc])+de)
	}
	lit.put(d, graphics.DeflateEndOfBlock)
	for ; d.nacc > 0; d.nacc -= min(d.nacc, 8) {
		d.out = append(d.out, byte(d.acc))
		d.acc >>= 8
	}
}

func (d *deflater) dynamicHeader() {
	d.build(&d.litCode, d.lit[:], graphics.DeflateMaxBits)
	d.build(&d.distCode, d.dist[:], graphics.DeflateMaxBits)
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
		d.lenCode.put(d, int(r&0xff))
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
	clear(h.lengths[:])
	i := 0
	for l := limit; l > 0; l-- {
		for range count[l] {
			h.lengths[order[i]&0xffff] = uint8(l)
			i++
		}
	}
	h.canonical()
}

func (h *huffman) canonical() {
	var count, next [graphics.DeflateMaxBits + 1]uint16
	for _, l := range h.lengths {
		count[l]++
	}
	count[0] = 0
	for l := 1; l <= graphics.DeflateMaxBits; l++ {
		next[l] = (next[l-1] + count[l-1]) << 1
	}
	for s, l := range h.lengths {
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
		d.fixedLit.canonical()
		d.fixedDist.canonical()
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
