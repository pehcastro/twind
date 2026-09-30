package present

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

var sixelRaster = regexp.MustCompile(`^\x1bP0;1q"1;1;(\d+);(\d+)`)

type term struct {
	x, y  int
	pen   color.RGBA
	cells [rows][cols]termCell
}

type termCell struct {
	bg    color.RGBA
	image bool
	text  string
	wide  bool
}

func (m *term) set(x, y int, c termCell) {
	row := &m.cells[y]
	if x > 0 && row[x-1].wide {
		row[x-1] = termCell{bg: row[x-1].bg}
	}
	if row[x].wide && x+1 < cols {
		row[x+1] = termCell{bg: row[x+1].bg}
	}
	row[x] = c
}

func (m *term) write(t *testing.T, p []byte) {
	t.Helper()
	for len(p) > 0 {
		switch {
		case bytes.HasPrefix(p, []byte("\x1bP")):
			end := bytes.Index(p, []byte("\x1b\\"))
			size := sixelRaster.FindSubmatch(p[:end])
			w, _ := strconv.Atoi(string(size[1]))
			h, _ := strconv.Atoi(string(size[2]))
			for y := m.y; y < min(m.y+(h+wt.Y-1)/wt.Y, rows); y++ {
				for x := m.x; x < min(m.x+(w+wt.X-1)/wt.X, cols); x++ {
					m.cells[y][x].image = true
				}
			}
			p = p[end+2:]
		case bytes.HasPrefix(p, []byte(termkonst.CSI)):
			i := len(termkonst.CSI)
			for p[i] >= '0' && p[i] <= '?' {
				i++
			}
			params, final := string(p[len(termkonst.CSI):i]), p[i]
			p = p[i+1:]
			if strings.HasPrefix(params, "?") {
				continue
			}
			var n []int
			for f := range strings.SplitSeq(params, ";") {
				v, _ := strconv.Atoi(f)
				n = append(n, v)
			}
			switch final {
			case 'H':
				m.y, m.x = n[0]-1, n[1]-1
			case 'C':
				m.x += n[0]
			case 'X':
				for x := m.x; x < min(m.x+n[0], cols); x++ {
					m.set(x, m.y, termCell{bg: m.pen})
				}
			case 'J':
				m.cells = [rows][cols]termCell{}
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
				t.Fatalf("the model cannot read %q", final)
			}
		case p[0] == '\x1b':
			t.Fatalf("the model cannot read %q", p[:min(len(p), 12)])
		default:
			r, size := utf8.DecodeRune(p)
			g := string(r)
			wide := text.Widths{}.Width(g) == 2
			m.set(m.x, m.y, termCell{bg: m.pen, text: g, wide: wide})
			if wide {
				m.x++
				m.cells[m.y][m.x] = termCell{bg: m.pen}
			}
			m.x++
			p = p[size:]
		}
	}
}

func grounded(t *testing.T, s *Screen, m *term, name string) {
	t.Helper()
	img := s.image()
	for y := range rows {
		for x := range cols {
			c, want := m.cells[y][x], s.text.At(x, y)
			sample := s.sample(x, y).RGBA
			if sample.A != 0 {
				sample.A = 255
			}
			ground := sample
			for p := range wt.X * wt.Y {
				if img.RGBAAt(x*wt.X+p%wt.X, y*wt.Y+p/wt.X).A != 255 {
					ground = color.RGBA{}
				}
			}
			switch {
			case want.Width == buffer.Continuation:
			case !blank(want) && (c.image || c.text != want.Grapheme):
				t.Fatalf("%s: cell %d,%d shows %+v, want the glyph %q over the image", name, x, y, c, want.Grapheme)
			case !blank(want):
			case c.image && c.bg != ground:
				t.Fatalf("%s: blank cell %d,%d under the image has the background %v, want %v: the colour sampled from the image where it is opaque, the default elsewhere", name, x, y, c.bg, ground)
			case !c.image && sample.A != 0 && (c.bg != sample || !s.flat(x, y)):
				t.Fatalf("%s: blank cell %d,%d shows %+v and no image, want the image over a %v background", name, x, y, c, sample)
			}
		}
	}
}

func TestImagesSitOnTheirSampledBackground(t *testing.T) {
	var names []string
	var steps []scene.Node
	add := func(name string, n scene.Node) {
		names, steps = append(names, name), append(steps, n)
	}
	add("dialog", tree(t, demo.Dialog()))
	add("typed", tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", "Dashboard")))
	for hover := range 4 {
		add("hover "+strconv.Itoa(hover), tree(t, demo.List(hover)))
	}
	for i, bg := range []color.RGBA{ink(200, 0, 0, 255), ink(0, 0, 200, 255)} {
		p := flatPage(ink(255, 255, 255, 255), "abcdefg中x")
		p.Children = []scene.Node{box(10, 0, 2, 1, bg)}
		add("wide at a tile edge "+strconv.Itoa(i), p)
	}
	for _, x := range []int{4, 9} {
		glass := flatPage(color.RGBA{}, "")
		glass.Children = []scene.Node{box(2, 2, 12, 4, ink(255, 255, 255, 90)), box(x, 3, 3, 1, ink(0, 0, 0, 255))}
		add("glass with a box at "+strconv.Itoa(x), glass)
	}
	s, out := screen(terminal.GraphicsSixel)
	m := &term{}
	for i, name := range names {
		frame(t, s, steps[i])
		m.write(t, out.last())
		grounded(t, s, m, name)
	}
}
