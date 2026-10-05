package present

import (
	"bytes"
	"image"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/text"
)

type xtCell struct {
	bg     color.RGBA
	text   string
	img    int
	tx, ty int
}

type xtImage struct {
	pix    *image.RGBA
	marker int
}

type xterm struct {
	cell, size  image.Point
	x, y        int
	pen         color.RGBA
	top, bottom int
	cells       [][]xtCell
	images      map[int]*xtImage
	added       int
}

func newXterm(cell image.Point) *xterm { return sizedXterm(image.Pt(cols, rows), cell) }

func sizedXterm(size, cell image.Point) *xterm {
	m := &xterm{cell: cell, size: size, bottom: size.Y - 1, cells: make([][]xtCell, size.Y), images: map[int]*xtImage{}}
	for y := range m.cells {
		m.cells[y] = make([]xtCell, size.X)
	}
	return m
}

func number(p []byte) (int, []byte) {
	n := 0
	for len(p) > 0 && p[0] >= '0' && p[0] <= '9' {
		n, p = n*10+int(p[0]-'0'), p[1:]
	}
	return n, p
}

func (m *xterm) sixel(t *testing.T, body []byte) {
	t.Helper()
	body = body[bytes.IndexByte(body, 'q')+1:]
	var registers [256]color.RGBA
	var pix *image.RGBA
	pen, x, band := 0, 0, 0
	for len(body) > 0 {
		c := body[0]
		body = body[1:]
		switch {
		case c == '"':
			var v [4]int
			for i := range v {
				v[i], body = number(body)
				if i < 3 {
					body = body[1:]
				}
			}
			pix = image.NewRGBA(image.Rect(0, 0, v[2], v[3]))
		case c == '#':
			pen, body = number(body)
			if len(body) > 1 && body[0] == ';' && body[1] == '2' {
				var rgb [3]int
				body = body[2:]
				for i := range rgb {
					rgb[i], body = number(body[1:])
				}
				percent := func(v int) uint8 { return uint8((v*255 + 50) / 100) }
				registers[pen] = color.RGBA{R: percent(rgb[0]), G: percent(rgb[1]), B: percent(rgb[2]), A: 255}
			}
		case c == '!':
			var n int
			n, body = number(body)
			bits := body[0] - '?'
			body = body[1:]
			for range n {
				dot(pix, x, band, bits, registers[pen])
				x++
			}
		case c == '$':
			x = 0
		case c == '-':
			x, band = 0, band+6
		case c >= '?' && c <= '~':
			dot(pix, x, band, c-'?', registers[pen])
			x++
		default:
			t.Fatalf("the sixel model cannot read %q", c)
		}
	}
	m.added++
	id := m.added
	n := image.Pt((pix.Rect.Dx()+m.cell.X-1)/m.cell.X, (pix.Rect.Dy()+m.cell.Y-1)/m.cell.Y)
	if m.y+n.Y > m.size.Y {
		t.Fatalf("an image of %d rows at row %d runs past the screen and would scroll it", n.Y, m.y)
	}
	for r := range n.Y {
		for c := range min(n.X, m.size.X-m.x) {
			cell := &m.cells[m.y+r][m.x+c]
			cell.img, cell.tx, cell.ty = id, c, r
		}
	}
	m.images[id] = &xtImage{pix: pix, marker: m.y + n.Y - 1}
	m.y += n.Y - 1
}

func dot(pix *image.RGBA, x, band int, bits byte, c color.RGBA) {
	for b := range 6 {
		if bits>>b&1 != 0 && image.Pt(x, band+b).In(pix.Rect) {
			copy(pix.Pix[pix.PixOffset(x, band+b):], []byte{c.R, c.G, c.B, c.A})
		}
	}
}

func (m *xterm) erase(x, y int) {
	m.cells[y][x] = xtCell{bg: m.pen}
}

func (m *xterm) scroll(up bool, n int) {
	for range n {
		gone, from, to, step := m.top, m.top+1, m.bottom, -1
		if !up {
			gone, from, to, step = m.bottom, m.top, m.bottom-1, 1
		}
		for _, img := range m.images {
			switch {
			case img.marker == gone:
				img.marker = -1
			case img.marker >= from && img.marker <= to:
				img.marker += step
			}
		}
		if up {
			copy(m.cells[m.top:m.bottom], m.cells[m.top+1:m.bottom+1])
			m.cells[m.bottom] = make([]xtCell, m.size.X)
			continue
		}
		copy(m.cells[m.top+1:m.bottom+1], m.cells[m.top:m.bottom])
		m.cells[m.top] = make([]xtCell, m.size.X)
	}
}

func (m *xterm) write(t *testing.T, p []byte) {
	t.Helper()
	for len(p) > 0 {
		switch {
		case bytes.HasPrefix(p, []byte("\x1bP")):
			end := bytes.Index(p, []byte("\x1b\\"))
			m.sixel(t, p[2:end])
			p = p[end+2:]
		case bytes.HasPrefix(p, []byte("\x1b[")):
			i := 2
			for p[i] >= '0' && p[i] <= '?' {
				i++
			}
			params, final := string(p[2:i]), p[i]
			p = p[i+1:]
			if strings.HasPrefix(params, "?") {
				continue
			}
			var n []int
			for f := range strings.SplitSeq(params, ";") {
				v, _ := strconv.Atoi(f)
				n = append(n, v)
			}
			arg := func(i, fallback int) int {
				if i < len(n) && n[i] > 0 {
					return n[i]
				}
				return fallback
			}
			switch final {
			case 'H':
				m.y, m.x = arg(0, 1)-1, arg(1, 1)-1
			case 'C':
				m.x += arg(0, 1)
			case 'X':
				for x := m.x; x < min(m.x+arg(0, 1), m.size.X); x++ {
					m.erase(x, m.y)
				}
			case 'J':
				for y := m.y; y < m.size.Y; y++ {
					for x := range m.size.X {
						if y > m.y || x >= m.x || n[0] == 2 {
							m.erase(x, y)
						}
					}
				}
			case 'r':
				m.top, m.bottom, m.x, m.y = arg(0, 1)-1, arg(1, m.size.Y)-1, 0, 0
			case 'S', 'T':
				m.scroll(final == 'S', arg(0, 1))
			case 'm':
				for i := 0; i < len(n); i++ {
					switch n[i] {
					case 0, 49:
						m.pen = color.RGBA{}
					case 48:
						m.pen, i = color.RGBA{R: uint8(n[i+2]), G: uint8(n[i+3]), B: uint8(n[i+4]), A: 255}, i+4
					case 38:
						i += 4
					}
				}
			default:
				t.Fatalf("the xterm model cannot read the final %q", final)
			}
		case p[0] == '\x1b':
			t.Fatalf("the xterm model cannot read %q", p[:min(len(p), 12)])
		default:
			r, size := utf8.DecodeRune(p)
			g := string(r)
			m.cells[m.y][m.x] = xtCell{bg: m.pen, text: g}
			if (text.Widths{}).Width(g) == 2 {
				m.x++
				m.cells[m.y][m.x] = xtCell{bg: m.pen}
			}
			m.x++
			p = p[size:]
		}
	}
}

func (m *xterm) shown(t *testing.T) (*image.RGBA, []string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, m.size.X*m.cell.X, m.size.Y*m.cell.Y))
	var glyphs []string
	for y, row := range m.cells {
		for x, c := range row {
			glyphs = append(glyphs, strings.TrimSpace(c.text))
			src := m.images[c.img]
			if c.img != 0 && src.marker < 0 {
				t.Errorf("cell %d,%d shows the placeholder of a deleted image", x, y)
			}
			for py := range m.cell.Y {
				for px := range m.cell.X {
					v := []byte{c.bg.R, c.bg.G, c.bg.B, c.bg.A}
					if at := image.Pt(c.tx*m.cell.X+px, c.ty*m.cell.Y+py); c.img != 0 && src.marker >= 0 && at.In(src.pix.Rect) && src.pix.RGBAAt(at.X, at.Y).A != 0 {
						v = src.pix.Pix[src.pix.PixOffset(at.X, at.Y):][:4]
					}
					copy(img.Pix[img.PixOffset(x*m.cell.X+px, y*m.cell.Y+py):], v)
				}
			}
		}
	}
	return img, glyphs
}

func sameScreen(t *testing.T, name string, got, want *xterm) {
	t.Helper()
	gotImg, gotText := got.shown(t)
	wantImg, wantText := want.shown(t)
	for i := range gotText {
		if gotText[i] != wantText[i] {
			t.Fatalf("%s: cell %d,%d shows %q, want %q", name, i%got.size.X, i/got.size.X, gotText[i], wantText[i])
		}
	}
	for i := 0; i < len(gotImg.Pix); i += 4 {
		if !bytes.Equal(gotImg.Pix[i:i+4], wantImg.Pix[i:i+4]) {
			x, y := i/4%gotImg.Rect.Dx(), i/4/gotImg.Rect.Dx()
			t.Fatalf("%s: pixel %d,%d (cell %d,%d) is %v, want %v", name, x, y, x/got.cell.X, y/got.cell.Y, gotImg.Pix[i:i+4], wantImg.Pix[i:i+4])
		}
	}
}
