package budgets_test

import (
	"bytes"
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	docsapp "github.com/twind-dev/twind/apps/documentation"
	playground "github.com/twind-dev/twind/examples/playground/app"
	runkonst "github.com/twind-dev/twind/internal/konst/runtime"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

const (
	motionQuiet       = 300 * time.Millisecond
	motionContentFrom = 30
	motionMiddle      = "middle of the screen"
	motionPassing     = "glyph passing"
	sixelBand         = 6
	sixelPercent      = 100
	pixelBytes        = 4
	blinkStep         = 16
	blinkReturn       = 4
	edgeReach         = 4
	edgeFirst         = '─'
	edgeLast          = '▟'
)

type motionEvent struct {
	at    time.Duration
	write []byte
	paint *terminal.Pixels
}

type motionBackend struct {
	events     chan input.Event
	signal     chan struct{}
	now        func() time.Duration
	cols, rows int
	caps       terminal.Capabilities
	mu         sync.Mutex
	log        []motionEvent
}

func (be *motionBackend) record(e motionEvent) {
	be.mu.Lock()
	be.log = append(be.log, e)
	be.mu.Unlock()
	select {
	case be.signal <- struct{}{}:
	default:
	}
}

func (be *motionBackend) Write(p []byte) (int, error) {
	be.record(motionEvent{at: be.now(), write: slices.Clone(p)})
	return len(p), nil
}

func (be *motionBackend) Paint(p terminal.Pixels) bool {
	p.Tiles = slices.Clone(p.Tiles)
	for i := range p.Tiles {
		p.Tiles[i].Pix = slices.Clone(p.Tiles[i].Pix)
	}
	be.record(motionEvent{at: be.now(), paint: &p})
	return true
}

func (be *motionBackend) Events() <-chan input.Event           { return be.events }
func (be *motionBackend) Sync() bool                           { return be.caps.Sync }
func (be *motionBackend) Exit() error                          { return nil }
func (be *motionBackend) Size() (width, height int, err error) { return be.cols, be.rows, nil }
func (be *motionBackend) Capabilities() terminal.Capabilities  { return be.caps }

func (be *motionBackend) recorded() int {
	be.mu.Lock()
	defer be.mu.Unlock()
	return len(be.log)
}

func (be *motionBackend) quiet(from int) []motionEvent {
	for {
		select {
		case <-be.signal:
		case <-time.After(motionQuiet):
			be.mu.Lock()
			defer be.mu.Unlock()
			return slices.Clone(be.log[from:])
		}
	}
}

type motionCell struct {
	text string
	fg   uint64
	bg   uint64
	attr uint16
	pix  []byte
}

type motionModel struct {
	size, cell        image.Point
	replace           bool
	x, y, top, bottom int
	pen               motionCell
	cells             []motionCell
	writes            []int
	canvas            []byte
}

func newMotionModel(p motionPath) *motionModel {
	m := &motionModel{size: image.Pt(p.cols, p.rows), cell: p.cell(), replace: p.caps.Identity == terminal.IdentityVSCode, bottom: p.rows - 1, cells: make([]motionCell, p.cols*p.rows), writes: make([]int, p.cols*p.rows)}
	if p.caps.Graphics == terminal.GraphicsGDI {
		m.canvas = make([]byte, p.cols*p.cell().X*p.rows*p.cell().Y*pixelBytes)
	}
	return m
}

func (m *motionModel) at(x, y int) *motionCell { return &m.cells[y*m.size.X+x] }

func (m *motionModel) set(x, y int, c motionCell) {
	if x >= 0 && x < m.size.X && y >= 0 && y < m.size.Y {
		*m.at(x, y) = c
		m.writes[y*m.size.X+x]++
	}
}

func (m *motionModel) erase(x, y int) { m.set(x, y, motionCell{text: " ", bg: m.pen.bg}) }

func (m *motionModel) write(t testing.TB, p []byte) {
	for len(p) > 0 {
		switch {
		case bytes.HasPrefix(p, []byte("\x1bP")):
			end := bytes.Index(p, []byte(termkonst.ST))
			m.sixel(t, p[2:end])
			p = p[end+len(termkonst.ST):]
		case bytes.HasPrefix(p, []byte(termkonst.CSI)):
			i := len(termkonst.CSI)
			for p[i] >= '0' && p[i] <= '?' {
				i++
			}
			m.control(t, string(p[len(termkonst.CSI):i]), p[i])
			p = p[i+1:]
		case p[0] == '\x1b':
			t.Fatalf("the motion model cannot read %q", p[:min(len(p), 12)])
		default:
			end := bytes.IndexByte(p, '\x1b')
			if end < 0 {
				end = len(p)
			}
			for g := range text.Graphemes(string(p[:end])) {
				c := m.pen
				c.text = g
				m.set(m.x, m.y, c)
				m.x++
				if (text.Widths{}).Width(g) == 2 {
					m.set(m.x, m.y, m.pen)
					m.x++
				}
			}
			p = p[end:]
		}
	}
}

func (m *motionModel) control(t testing.TB, params string, final byte) {
	if strings.HasPrefix(params, "?") {
		return
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
		for x := m.x; x < m.x+arg(0, 1); x++ {
			m.erase(x, m.y)
		}
	case 'K':
		for x := range m.size.X {
			if x >= m.x || n[0] == 2 {
				m.erase(x, m.y)
			}
		}
	case 'J':
		for y := range m.size.Y {
			for x := range m.size.X {
				if y > m.y || y == m.y && x >= m.x || n[0] == 2 {
					m.erase(x, y)
				}
			}
		}
	case 'r':
		m.top, m.bottom, m.x, m.y = arg(0, 1)-1, arg(1, m.size.Y)-1, 0, 0
	case 'S', 'T':
		for range arg(0, 1) {
			rows := m.cells[m.top*m.size.X : (m.bottom+1)*m.size.X]
			if final == 'S' {
				copy(rows, rows[m.size.X:])
				clear(rows[len(rows)-m.size.X:])
				continue
			}
			copy(rows[m.size.X:], rows)
			clear(rows[:m.size.X])
		}
	case 'm':
		m.sgr(t, n)
	default:
		t.Fatalf("the motion model cannot read the final %q of %q", final, params)
	}
}

func (m *motionModel) sgr(t testing.TB, n []int) {
	for i := 0; i < len(n); i++ {
		switch code := n[i]; {
		case code == termkonst.SGRReset:
			m.pen = motionCell{}
		case code < termkonst.FgBase:
			m.pen.attr |= 1 << code
		case code == termkonst.FgDefault:
			m.pen.fg = 0
		case code == termkonst.FgDefault+termkonst.BgOffset:
			m.pen.bg = 0
		case code == termkonst.FgExtended || code == termkonst.FgExtended+termkonst.BgOffset:
			if n[i+1] != termkonst.PaletteRGB {
				t.Fatalf("the motion model reads only 24-bit colour, got SGR %v", n[i:])
			}
			ink := 1<<24 | uint64(n[i+2])<<16 | uint64(n[i+3])<<8 | uint64(n[i+4])
			if code == termkonst.FgExtended {
				m.pen.fg = ink
			} else {
				m.pen.bg = ink
			}
			i += 4
		default:
			t.Fatalf("the motion model cannot read SGR %d", code)
		}
	}
}

func motionNumber(p []byte) (int, []byte) {
	n := 0
	for len(p) > 0 && p[0] >= '0' && p[0] <= '9' {
		n, p = n*10+int(p[0]-'0'), p[1:]
	}
	return n, p
}

func (m *motionModel) sixel(t testing.TB, body []byte) {
	body = body[bytes.IndexByte(body, 'q')+1:]
	var registers [256][4]byte
	var pix *image.RGBA
	pen, x, band := 0, 0, 0
	dot := func(bits byte) {
		for b := range sixelBand {
			if bits>>b&1 != 0 && image.Pt(x, band+b).In(pix.Rect) {
				copy(pix.Pix[pix.PixOffset(x, band+b):], registers[pen][:])
			}
		}
		x++
	}
	for len(body) > 0 {
		c := body[0]
		body = body[1:]
		switch {
		case c == '"':
			var v [4]int
			for i := range v {
				v[i], body = motionNumber(body)
				if i < len(v)-1 {
					body = body[1:]
				}
			}
			pix = image.NewRGBA(image.Rect(0, 0, v[2], v[3]))
		case c == '#':
			pen, body = motionNumber(body)
			if len(body) > 1 && body[0] == ';' && body[1] == '2' {
				var rgb [3]int
				body = body[2:]
				for i := range rgb {
					rgb[i], body = motionNumber(body[1:])
				}
				for i, v := range rgb {
					registers[pen][i] = uint8((v*255 + sixelPercent/2) / sixelPercent)
				}
				registers[pen][3] = 255
			}
		case c == '!':
			var n int
			n, body = motionNumber(body)
			bits := body[0] - '?'
			body = body[1:]
			for range n {
				dot(bits)
			}
		case c == '$':
			x = 0
		case c == '-':
			x, band = 0, band+sixelBand
		case c >= '?' && c <= '~':
			dot(c - '?')
		default:
			t.Fatalf("the sixel model cannot read %q", c)
		}
	}
	n := image.Pt((pix.Rect.Dx()+m.cell.X-1)/m.cell.X, (pix.Rect.Dy()+m.cell.Y-1)/m.cell.Y)
	for r := range min(n.Y, m.size.Y-m.y) {
		for c := range min(n.X, m.size.X-m.x) {
			cell := m.at(m.x+c, m.y+r)
			block := make([]byte, m.cell.X*m.cell.Y*pixelBytes)
			if !m.replace && cell.pix != nil {
				copy(block, cell.pix)
			}
			seen := false
			for py := range m.cell.Y {
				for px := range m.cell.X {
					at := image.Pt(c*m.cell.X+px, r*m.cell.Y+py)
					if !at.In(pix.Rect) || pix.Pix[pix.PixOffset(at.X, at.Y)+3] == 0 {
						continue
					}
					copy(block[(py*m.cell.X+px)*pixelBytes:], pix.Pix[pix.PixOffset(at.X, at.Y):][:pixelBytes])
				}
			}
			for _, v := range block {
				seen = seen || v != 0
			}
			cell.pix = nil
			if seen {
				cell.pix = block
			}
			m.writes[(m.y+r)*m.size.X+m.x+c]++
		}
	}
	m.y += n.Y - 1
}

func (m *motionModel) paint(p *terminal.Pixels) {
	if p.Clear {
		clear(m.canvas)
	}
	stride := m.size.X * m.cell.X * pixelBytes
	for _, tile := range p.Tiles {
		r := image.Rect(tile.Cells.Min.X*m.cell.X, tile.Cells.Min.Y*m.cell.Y, tile.Cells.Max.X*m.cell.X, tile.Cells.Max.Y*m.cell.Y)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			row := m.canvas[y*stride+r.Min.X*pixelBytes : y*stride+r.Max.X*pixelBytes]
			if tile.Pix == nil {
				clear(row)
				continue
			}
			copy(row, tile.Pix[(y-r.Min.Y)*r.Dx()*pixelBytes:])
		}
	}
}

type motionLook struct {
	text   string
	glyph  string
	pixels bool
	colour [3]int
}

func (l motionLook) far(o motionLook, step int) bool {
	for i := range l.colour {
		if d := l.colour[i] - o.colour[i]; max(d, -d) > step {
			return true
		}
	}
	return false
}

func (m *motionModel) state() []motionLook {
	looks := make([]motionLook, len(m.cells))
	stride := m.size.X * m.cell.X * pixelBytes
	area := m.cell.X * m.cell.Y
	for i, c := range m.cells {
		l := &looks[i]
		if strings.TrimSpace(c.text) != "" {
			l.text, l.glyph = c.text, fmt.Sprint(c.text, c.fg, c.attr)
		}
		bg := [3]int{int(c.bg >> 16 & 0xff), int(c.bg >> 8 & 0xff), int(c.bg & 0xff)}
		x, y := i%m.size.X, i/m.size.X
		for py := range m.cell.Y {
			for px := range m.cell.X {
				var p []byte
				switch {
				case m.canvas != nil:
					at := (y*m.cell.Y+py)*stride + (x*m.cell.X+px)*pixelBytes
					q := m.canvas[at : at+pixelBytes]
					p = []byte{q[2], q[1], q[0], q[3]}
				case c.pix != nil:
					p = c.pix[(py*m.cell.X+px)*pixelBytes:][:pixelBytes]
				}
				if p == nil || p[3] == 0 {
					for k := range bg {
						l.colour[k] += bg[k]
					}
					continue
				}
				l.pixels = true
				for k := range bg {
					l.colour[k] += int(p[k])
				}
			}
		}
		for k := range l.colour {
			l.colour[k] /= area
		}
	}
	return looks
}

type motionFrame struct {
	at     time.Duration
	synced bool
	write  []byte
	paint  *terminal.Pixels
}

func motionFrames(events []motionEvent) []motionFrame {
	var frames []motionFrame
	for _, e := range events {
		if e.paint != nil && len(frames) > 0 && frames[len(frames)-1].paint == nil && e.at-frames[len(frames)-1].at < runkonst.FrameInterval/2 {
			frames[len(frames)-1].paint = e.paint
			continue
		}
		frames = append(frames, motionFrame{at: e.at, write: e.write, paint: e.paint, synced: e.write == nil || bytes.HasPrefix(e.write, []byte(termkonst.SyncBegin)) && bytes.HasSuffix(e.write, []byte(termkonst.SyncEnd))})
	}
	return frames
}

type motionBlink struct {
	frame, x, y int
	kind        string
	seen        [3]motionLook
}

type motionResult struct {
	frames                []motionFrame
	first                 time.Duration
	blinks                []motionBlink
	unsynced, wiped, torn int
	cycles                uint64
}

func (m *motionModel) apply(t testing.TB, frames []motionFrame, r *motionResult) {
	states := [][]motionLook{m.state()}
	for i, f := range frames {
		clear(m.writes)
		m.write(t, f.write)
		if f.paint != nil {
			m.paint(f.paint)
		}
		now := m.state()
		for c, n := range m.writes {
			if m.canvas != nil && n > 0 && states[len(states)-1][c].pixels && !painted(f.paint, c%m.size.X, c/m.size.X) {
				r.wiped++
			}
			if n > 1 && !f.synced {
				r.torn++
			}
		}
		if !f.synced {
			r.unsynced++
		}
		states = append(states, now)
		if len(states) < 3 {
			continue
		}
		a, b := states[len(states)-3], states[len(states)-2]
		for c := range now {
			kind := ""
			switch {
			case a[c].glyph == now[c].glyph && b[c].glyph != now[c].glyph && now[c].glyph == "":
				kind = motionPassing
			case a[c].glyph == now[c].glyph && b[c].glyph != now[c].glyph:
				kind = "glyph gone"
			case a[c].pixels && now[c].pixels && !b[c].pixels && a[c].far(b[c], blinkStep) && now[c].far(b[c], blinkStep):
				kind = "pixels gone"
			case !a[c].far(now[c], blinkReturn) && a[c].far(b[c], blinkStep) && now[c].far(b[c], blinkStep):
				kind = "colour"
			}
			if kind != "" && !m.swept(a, b, now, c) {
				r.blinks = append(r.blinks, motionBlink{frame: i, x: c % m.size.X, y: c / m.size.X, kind: kind, seen: [3]motionLook{a[c], b[c], now[c]}})
			}
		}
	}
}

func (m *motionModel) swept(a, b, now []motionLook, c int) bool {
	l := b[c]
	r := []rune(l.text)
	fill := l.glyph == "" && a[c].glyph == "" && now[c].glyph == ""
	if !fill && (len(r) != 1 || r[0] < edgeFirst || r[0] > edgeLast) {
		return false
	}
	same := func(o motionLook) bool { return o.text == l.text && (!fill || !o.far(l, blinkReturn)) }
	for _, d := range []image.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		for k := 1; k <= edgeReach; k++ {
			p := image.Pt(c%m.size.X+d.X*k, c/m.size.X+d.Y*k)
			if !p.In(image.Rect(0, 0, m.size.X, m.size.Y)) {
				break
			}
			n := p.Y*m.size.X + p.X
			if !same(b[n]) && same(now[n]) != same(a[n]) {
				return true
			}
		}
	}
	return false
}

func painted(p *terminal.Pixels, x, y int) bool {
	if p == nil {
		return false
	}
	if p.Clear {
		return true
	}
	return slices.ContainsFunc(p.Tiles, func(t terminal.Tile) bool { return image.Pt(x, y).In(t.Cells) })
}

type motionPath struct {
	name       string
	cols, rows int
	caps       terminal.Capabilities
}

func (p motionPath) cell() image.Point {
	if p.caps.Graphics == terminal.GraphicsNone {
		return image.Pt(1, 1)
	}
	return p.caps.CellPixels
}

type motionStep struct {
	name   string
	events func(at map[string]image.Point) []input.Event
}

type motionScenario struct {
	name       string
	page       string
	targets    []string
	cycle      []motionStep
	playground bool
}

func (sc motionScenario) app(rt *twi.Runtime) (func() twi.Node, error) {
	if sc.playground {
		return playground.New(rt, playground.Env{Profile: "truecolor", Size: func() string { return "bench" }}, playground.Start{Page: sc.page, Theme: "twind-dark", Focus: "input"})
	}
	return docsapp.New(rt, docsapp.Start{Page: sc.page, Theme: "twind-dark"})
}

func (sc motionScenario) styles(t testing.TB) style.Sheet {
	styles := docsapp.Styles
	if sc.playground {
		styles = playground.Styles
	}
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func motionClick(target string) func(map[string]image.Point) []input.Event {
	return func(at map[string]image.Point) []input.Event { return docsClick(at[target].X, at[target].Y) }
}

func motionMove(target string, dx int) func(map[string]image.Point) []input.Event {
	return func(at map[string]image.Point) []input.Event {
		return []input.Event{input.MouseEvent{X: at[target].X + dx, Y: at[target].Y, Button: input.MouseNone, Action: input.MouseMove}}
	}
}

func motionKey(k input.Key) func(map[string]image.Point) []input.Event {
	return func(map[string]image.Point) []input.Event { return []input.Event{input.KeyEvent{Key: k}} }
}

func motionWheel(button input.MouseButton) func(map[string]image.Point) []input.Event {
	return func(at map[string]image.Point) []input.Event {
		return []input.Event{input.MouseEvent{X: at[motionMiddle].X, Y: at[motionMiddle].Y, Button: button, Action: input.MouseScroll}}
	}
}

func motionTargets(t testing.TB, sheet style.Sheet, p motionPath, sc motionScenario) map[string]image.Point {
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		view, err := sc.app(rt)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}, drive.Size(p.cols, p.rows), drive.Styles(sheet))
	cells := d.Frame().Cells()
	at := map[string]image.Point{motionMiddle: image.Pt(p.cols/2, p.rows/2)}
	for _, target := range sc.targets {
		from := motionContentFrom
		if sc.page == "card" {
			from = 0
		}
	search:
		for y := range cells.Height() {
			var row strings.Builder
			var xs []int
			for x, c := range cells.Row(y) {
				if c.Width == buffer.Continuation {
					continue
				}
				for range len(c.Grapheme) {
					xs = append(xs, x)
				}
				row.WriteString(c.Grapheme)
			}
			for i := 0; i < len(xs); {
				found := strings.Index(row.String()[i:], target)
				if found < 0 {
					break
				}
				if x := xs[i+found]; x >= from {
					at[target] = image.Pt(x+1, y)
					break search
				}
				i += found + 1
			}
		}
		if _, ok := at[target]; !ok {
			t.Fatalf("%s: %q is not on the %s page at %dx%d", sc.name, target, sc.page, p.cols, p.rows)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	return at
}

func motionPaths() []motionPath {
	sixel := func(id terminal.Identity, cell image.Point) terminal.Capabilities {
		return terminal.Capabilities{Identity: id, Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: cell}
	}
	return []motionPath{
		{"wt", 120, 36, sixel(terminal.IdentityOther, image.Pt(10, 20))},
		{"vscode", 248, 36, sixel(terminal.IdentityVSCode, image.Pt(7, 17))},
		{"conhost", 120, 30, terminal.Capabilities{Identity: terminal.IdentityConhost, Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(8, 16)}},
		{"overlay", 187, 40, terminal.Capabilities{Identity: terminal.IdentityZed, Sync: true, Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(8, 17)}},
		{"cells", 120, 36, terminal.Capabilities{Sync: true}},
	}
}

func motionScenarios() []motionScenario {
	return []motionScenario{
		{"dialog", "dialog", []string{"Edit profile"}, []motionStep{{"open", motionClick("Edit profile")}, {"close", motionKey(input.KeyEscape)}}, false},
		{"dropdown", "dropdown-menu", []string{"Open menu"}, []motionStep{{"open", motionClick("Open menu")}, {"close", motionKey(input.KeyEscape)}}, false},
		{"toast", "toaster", []string{"Show toast"}, []motionStep{{"in", motionClick("Show toast")}, {"out", motionKey(input.KeyEscape)}}, false},
		{"button", "button", []string{"↑"}, []motionStep{{"hover", motionMove("↑", 0)}, {"leave", motionMove("↑", 6)}}, false},
		{"sidebar", "card", []string{"Introduction", "Installation"}, []motionStep{{"hover", motionMove("Introduction", 0)}, {"next", motionMove("Installation", 0)}}, false},
		{"accordion", "accordion", []string{"What is your return policy?", "What are your shipping options?"}, []motionStep{{"returns", motionClick("What is your return policy?")}, {"shipping", motionClick("What are your shipping options?")}}, false},
		{"tabs", "tabs", []string{"Password", "Account"}, []motionStep{{"password", motionClick("Password")}, {"account", motionClick("Account")}}, false},
		{"wheel", "card", nil, []motionStep{{"down", motionWheel(input.MouseWheelDown)}, {"up", motionWheel(input.MouseWheelUp)}}, false},
		{"spring", "motion", []string{"Open menu"}, []motionStep{{"open", motionClick("Open menu")}, {"close", motionClick("Open menu")}}, true},
		{"reorder", "motion", []string{"Rotate"}, []motionStep{{"rotate", motionClick("Rotate")}, {"again", motionClick("Rotate")}}, true},
	}
}

func runMotion(t testing.TB, now func() time.Duration, p motionPath, sc motionScenario, rounds int) map[string][]motionResult {
	sheet := sc.styles(t)
	at := motionTargets(t, sheet, p, sc)
	be := &motionBackend{events: make(chan input.Event), signal: make(chan struct{}, 1), now: now, cols: p.cols, rows: p.rows, caps: p.caps}
	rt := twi.New(twi.Backend(be, realDocsClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	view, err := sc.app(rt)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- rt.Run(view) }()
	model := newMotionModel(p)
	be.quiet(0)
	be.events <- input.MouseEvent{X: p.cols - 1, Y: p.rows - 1, Button: input.MouseNone, Action: input.MouseMove}
	model.apply(t, motionFrames(be.quiet(0)), &motionResult{})
	results := map[string][]motionResult{}
	for range rounds {
		for _, s := range sc.cycle {
			c0, begin, from := cycles(), now(), be.recorded()
			for _, e := range s.events(at) {
				be.events <- e
			}
			got := be.quiet(from)
			spent := cycles() - c0
			frames := motionFrames(got)
			if len(frames) == 0 {
				t.Fatalf("%s %s %s: no frame after the events", p.name, sc.name, s.name)
			}
			r := motionResult{frames: frames, first: frames[0].at - begin, cycles: spent}
			model.apply(t, frames, &r)
			results[s.name] = append(results[s.name], r)
		}
	}
	rt.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	return results
}

func BenchmarkMotion(b *testing.B) {
	for _, p := range motionPaths() {
		for _, sc := range motionScenarios() {
			for _, s := range sc.cycle {
				b.Run(p.name+"/"+sc.name+"/"+s.name, func(b *testing.B) {
					results := runMotion(b, clock(b), p, sc, b.N)[s.name]
					reportMotion(b, results)
				})
			}
		}
	}
}

func TestMotionHasNoBlinks(t *testing.T) {
	hoverFollowsLayout := map[string]bool{"accordion": true}
	for _, p := range motionPaths() {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			for _, sc := range motionScenarios() {
				for step, results := range runMotion(t, clock(t), p, sc, 1) {
					for _, r := range results {
						var shown []motionBlink
						for _, k := range r.blinks {
							if hoverFollowsLayout[sc.name] && k.kind == "glyph gone" && k.seen[0].text == k.seen[1].text {
								continue
							}
							shown = append(shown, k)
						}
						if len(shown) > 0 {
							k := shown[0]
							t.Errorf("%s %s: %d blinks, the first %s at frame %d of %d, cell %d,%d: %+v", sc.name, step, len(shown), k.kind, k.frame, len(r.frames), k.x, k.y, k.seen)
						}
					}
				}
			}
		})
	}
}

func TestBlinkTellsAMovingEdge(t *testing.T) {
	at := func(row int, s string) string { return termkonst.CSI + strconv.Itoa(row+1) + ";1H" + s }
	red, dim := termkonst.CSI+"48;2;200;0;0m  "+termkonst.CSI+"0m", termkonst.CSI+"48;2;100;0;0m  "+termkonst.CSI+"0m"
	for _, c := range []struct {
		name   string
		frames []string
		want   []image.Point
	}{
		{"edge moves down", []string{at(1, "──"), at(1, "  ") + at(3, "──")}, nil},
		{"edge moves up over a border", []string{at(2, "──") + at(4, "──"), at(2, "▀▀"), at(2, "──") + at(1, "▀▀")}, nil},
		{"edge moves up and the box closes", []string{at(2, "──"), at(2, "  ") + at(1, "──"), at(1, "  ")}, nil},
		{"edge shows once", []string{at(1, "──"), at(1, "  ")}, []image.Point{{0, 1}, {1, 1}}},
		{"edge shows once beside one that stays", []string{at(1, "──") + at(2, "──"), at(1, "  ")}, []image.Point{{0, 1}, {1, 1}}},
		{"edge bounces", []string{at(2, "──"), at(2, "  ") + at(1, "──"), at(1, "  ") + at(2, "──")}, []image.Point{{0, 1}, {1, 1}, {0, 2}, {1, 2}}},
		{"text blinks while the row below clears", []string{at(1, "ab") + at(2, "ab"), "", at(1, "  "), at(1, "ab") + at(2, "  ")}, []image.Point{{0, 1}, {1, 1}}},
		{"fill moves down", []string{at(1, red), at(1, "  ") + at(2, red)}, nil},
		{"fill flashes", []string{at(1, red), at(1, "  ")}, []image.Point{{0, 1}, {1, 1}}},
		{"fill flashes as a dimmer one moves in", []string{at(1, red), at(1, "  ") + at(2, dim)}, []image.Point{{0, 1}, {1, 1}}},
		{"text jumps", []string{at(1, "ab"), at(1, "  ") + at(3, "ab")}, []image.Point{{0, 1}, {1, 1}}},
	} {
		m := newMotionModel(motionPath{"cells", 4, 6, terminal.Capabilities{Sync: true}})
		var frames []motionFrame
		for _, f := range c.frames {
			frames = append(frames, motionFrame{synced: true, write: []byte(f)})
		}
		var r motionResult
		m.apply(t, frames, &r)
		var got []image.Point
		for _, k := range r.blinks {
			got = append(got, image.Pt(k.x, k.y))
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: blinks at %v, want %v", c.name, got, c.want)
		}
	}
}

func reportMotion(b *testing.B, results []motionResult) {
	var gaps, firsts []time.Duration
	frames, written, blinks, passing, unsynced, wiped, torn := 0, 0, 0, 0, 0, 0, 0
	var spent uint64
	var span time.Duration
	for _, r := range results {
		firsts = append(firsts, r.first)
		for i, f := range r.frames {
			if i > 0 {
				gaps = append(gaps, f.at-r.frames[i-1].at)
			}
			written += len(f.write)
		}
		frames += len(r.frames)
		span += r.frames[len(r.frames)-1].at - r.frames[0].at
		unsynced, wiped, torn, spent = unsynced+r.unsynced, wiped+r.wiped, torn+r.torn, spent+r.cycles
		kinds := map[string]int{}
		for _, k := range r.blinks {
			if k.kind == motionPassing {
				passing++
			} else {
				blinks++
			}
			if kinds[k.kind]++; kinds[k.kind] == 1 {
				b.Logf("blink %s: frame %d of %d, cell %d,%d: %+v", k.kind, k.frame, len(r.frames), k.x, k.y, k.seen)
			}
		}
		if len(kinds) > 0 {
			b.Logf("op blinks by kind: %v", kinds)
		}
	}
	n := float64(len(results))
	b.ReportMetric(float64(frames)/n, "frames/op")
	if len(gaps) > 0 {
		slices.Sort(gaps)
		b.ReportMetric(float64(frames-len(results))/span.Seconds(), "fps")
		b.ReportMetric(float64(gaps[len(gaps)/2])/float64(time.Millisecond), "gap-p50-ms")
		b.ReportMetric(float64(gaps[len(gaps)*95/100])/float64(time.Millisecond), "gap-p95-ms")
		b.ReportMetric(float64(gaps[len(gaps)-1])/float64(time.Millisecond), "gap-max-ms")
	}
	slices.Sort(firsts)
	b.ReportMetric(float64(firsts[len(firsts)/2])/float64(time.Millisecond), "first-ms")
	b.ReportMetric(float64(written)/float64(frames), "B/frame")
	b.ReportMetric(float64(spent)/float64(frames), "cycles/frame")
	b.ReportMetric(float64(blinks)/n, "blinks/op")
	b.ReportMetric(float64(passing)/n, "passing/op")
	b.ReportMetric(float64(unsynced)/n, "unsynced/op")
	b.ReportMetric(float64(wiped)/n, "wiped/op")
	b.ReportMetric(float64(torn)/n, "torn/op")
}
