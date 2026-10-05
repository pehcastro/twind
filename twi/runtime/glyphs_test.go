package runtime_test

import (
	"bytes"
	"image"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/twind-dev/twind/examples/playground/app"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

const pageCols, pageRows = 100, 30

type stepped struct {
	mu     sync.Mutex
	now    time.Time
	alarms []alarm
}

type alarm struct {
	at   time.Time
	ring chan time.Time
}

func (c *stepped) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *stepped) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ring := make(chan time.Time, 1)
	c.alarms = append(c.alarms, alarm{c.now.Add(d), ring})
	return ring
}

func (c *stepped) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	waiting := c.alarms[:0]
	for _, a := range c.alarms {
		if a.at.After(c.now) {
			waiting = append(waiting, a)
			continue
		}
		a.ring <- c.now
	}
	c.alarms = waiting
}

type screenCell struct {
	text  string
	image bool
}

type screenModel struct {
	x, y  int
	cells [pageRows][pageCols]screenCell
}

func (m *screenModel) write(t *testing.T, p []byte, cell image.Point) {
	t.Helper()
	for len(p) > 0 {
		switch {
		case bytes.HasPrefix(p, []byte("\x1bP")):
			end := bytes.Index(p, []byte("\x1b\\"))
			m.covered(p[sixelSize.FindIndex(p[:end])[1]:end], cell)
			p = p[end+2:]
		case bytes.HasPrefix(p, []byte(termkonst.CSI)):
			i := len(termkonst.CSI)
			for p[i] >= '0' && p[i] <= '?' {
				i++
			}
			params, final := string(p[len(termkonst.CSI):i]), p[i]
			p = p[i+1:]
			var n [2]int
			fields := strings.Split(params, ";")
			for k := range min(2, len(fields)) {
				n[k], _ = strconv.Atoi(fields[k])
			}
			switch {
			case strings.HasPrefix(params, "?") || final == 'm':
			case final == 'H':
				m.y, m.x = n[0]-1, n[1]-1
			case final == 'C':
				m.x += n[0]
			case final == 'X':
				for x := m.x; x < min(m.x+n[0], pageCols); x++ {
					m.cells[m.y][x] = screenCell{}
				}
			case final == 'J':
				m.cells = [pageRows][pageCols]screenCell{}
			default:
				t.Fatalf("the model cannot read %q", params+string(final))
			}
		case p[0] == '\x1b':
			t.Fatalf("the model cannot read %q", p[:min(len(p), 16)])
		default:
			end := bytes.IndexByte(p, '\x1b')
			if end < 0 {
				end = len(p)
			}
			for g := range text.Graphemes(string(p[:end])) {
				m.cells[m.y][m.x] = screenCell{text: g}
				m.x++
				if (text.Widths{}).Width(g) == 2 {
					m.cells[m.y][m.x] = screenCell{}
					m.x++
				}
			}
			p = p[end:]
		}
	}
}

func (m *screenModel) covered(body []byte, cell image.Point) {
	x, band, repeat := 0, 0, 1
	for len(body) > 0 {
		c := body[0]
		body = body[1:]
		switch {
		case c == '#' || c == '!':
			i := 0
			for i < len(body) && (body[i] >= '0' && body[i] <= '9' || c == '#' && body[i] == ';') {
				i++
			}
			if c == '!' {
				repeat, _ = strconv.Atoi(string(body[:i]))
			}
			body = body[i:]
		case c == '$':
			x = 0
		case c == '-':
			x, band = 0, band+6
		case c >= '?' && c <= '~':
			for range repeat {
				for b := range 6 {
					if py := band + b; (c-'?')&(1<<b) != 0 && py%cell.Y == cell.Y/2 {
						if cy, cx := m.y+py/cell.Y, m.x+x/cell.X; cy < pageRows && cx < pageCols {
							m.cells[cy][cx].image = true
						}
					}
				}
				x++
			}
			repeat = 1
		}
	}
}

func shot(t *testing.T, page, theme string, g terminal.Graphics) *screenModel {
	t.Helper()
	cell := image.Pt(10, 20)
	b := newBackend(pageCols, pageRows)
	b.caps = terminal.Capabilities{Graphics: terminal.GraphicsSixel, CellPixels: cell}
	sheet, err := app.Styles()
	if err != nil {
		t.Fatal(err)
	}
	clock := &stepped{now: time.Unix(0, 0)}
	rt := twi.New(twi.Backend(b, clock), twi.Styles(sheet), twi.ColorProfile(color.TrueColor), twi.Graphics(g))
	view, err := app.New(rt, app.Env{Cwd: "/", Profile: "truecolor", Size: func() string { return "100x30" }}, app.Start{Page: page, Theme: theme, Focus: "input"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- rt.Run(view) }()
	settle := func() {
		marker := make(chan struct{})
		rt.Dispatch(func() { rt.Dispatch(func() { close(marker) }) })
		<-marker
	}
	settle()
	for range 2 {
		b.events <- input.KeyEvent{Key: input.KeyTab}
		clock.advance(time.Second)
		settle()
	}
	rt.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	close(b.frames)
	m := &screenModel{}
	for f := range b.frames {
		m.write(t, []byte(f), cell)
	}
	return m
}

func playgroundPages(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("../../examples/playground/app/components.go")
	if err != nil {
		t.Fatal(err)
	}
	pages := []string{"1", "2", "3", "4", "5", "6", "7"}
	for _, m := range regexp.MustCompile(`\{"([a-z ]+)", \w+Page`).FindAllSubmatch(src, -1) {
		pages = append(pages, string(m[1]))
	}
	return pages
}

func TestEveryTextCellIsACellOverPixelSurfaces(t *testing.T) {
	pages := playgroundPages(t)
	if len(pages) < 40 {
		t.Fatalf("found %d playground pages, want every component page", len(pages))
	}
	checked := 0
	for _, theme := range []string{"twind-dark", "twind-light"} {
		for _, page := range pages {
			cells, pixels := shot(t, page, theme, terminal.GraphicsNone), shot(t, page, theme, terminal.GraphicsSixel)
			for y := range pageRows {
				for x := range pageCols {
					want, got := cells.cells[y][x].text, pixels.cells[y][x]
					if r := []rune(want); len(r) == 0 || want == " " || r[0] >= 0x2500 && r[0] <= 0x259f {
						continue
					}
					checked++
					if got.image || got.text != want {
						t.Errorf("%s %s: cell %d,%d is %+v over pixel surfaces, want the glyph %q as a cell", theme, page, x, y, got, want)
					}
				}
			}
		}
	}
	t.Logf("%d pages, %d text cells checked", 2*len(pages), checked)
}
