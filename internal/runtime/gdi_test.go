package runtime_test

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/testdata/hello"
)

type gdiBackend struct {
	*backend
	paints chan terminal.Pixels
}

func (g gdiBackend) Paint(p terminal.Pixels) bool {
	g.paints <- p
	return true
}

type zedBackend struct {
	gdiBackend
	next chan terminal.Capabilities
}

func (z zedBackend) Capabilities() terminal.Capabilities {
	select {
	case z.caps = <-z.next:
	default:
	}
	return z.caps
}

func (z zedBackend) settle(t *testing.T, r run) {
	t.Helper()
	for {
		select {
		case <-r.b.frames:
		case <-time.After(300 * time.Millisecond):
			for len(z.paints) > 0 {
				<-z.paints
			}
			return
		}
	}
}

func TestZedOverlayGraphicsFollowTheBackend(t *testing.T) {
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	cells := terminal.Capabilities{Identity: terminal.IdentityZed, CellPixels: image.Pt(7, 16)}
	overlay := terminal.Capabilities{Identity: terminal.IdentityZed, Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(8, 17)}
	b := zedBackend{gdiBackend{newBackend(40, 15), make(chan terminal.Pixels, 64)}, make(chan terminal.Capabilities, 1)}
	b.caps = cells
	r := run{b: b.backend, clock: &clock{}, done: make(chan error, 1)}
	r.rt = twi.New(twi.Backend(b, r.clock), twi.Styles(s), twi.ColorProfile(color.TrueColor))
	app := hello.Surfaces(r.rt)
	go func() { r.done <- r.rt.Run(app) }()
	r.next(t)
	b.settle(t, r)
	b.next <- overlay
	b.events <- input.ResizeEvent{Width: 40, Height: 15, Cell: overlay.CellPixels}
	select {
	case p := <-b.paints:
		if !p.Clear || p.Cell != overlay.CellPixels {
			t.Errorf("first overlay paint clear %v cell %v, want a clear at %v", p.Clear, p.Cell, overlay.CellPixels)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("calibrated, yet no paint within 2s")
	}
	b.settle(t, r)
	b.next <- cells
	b.events <- input.ResizeEvent{Width: 40, Height: 15, Cell: cells.CellPixels}
	b.settle(t, r)
	b.events <- input.ResizeEvent{Width: 41, Height: 15}
	r.next(t)
	b.settle(t, r)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	if n := len(b.paints); n > 0 {
		t.Errorf("%d paints after the calibration was dropped, want the cells path", n)
	}
}

type markingBackend struct {
	*backend
	pages chan color.RGBA
}

func (m markingBackend) Marks(page color.RGBA) []terminal.Mark {
	m.pages <- page
	return []terminal.Mark{{Cell: image.Pt(0, 0), Color: color.RGBA{R: 1, G: 2, B: 3, A: 255}}}
}

func TestZedMarksReachTheFrame(t *testing.T) {
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	b := markingBackend{newBackend(40, 15), make(chan color.RGBA, 64)}
	r := run{b: b.backend, clock: &clock{}, done: make(chan error, 1)}
	r.rt = twi.New(twi.Backend(b, r.clock), twi.Styles(s), twi.ColorProfile(color.TrueColor))
	app := hello.Surfaces(r.rt)
	go func() { r.done <- r.rt.Run(app) }()
	first := r.next(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
	if len(b.pages) == 0 || !strings.Contains(first, "48;2;1;2;3") {
		t.Errorf("marks asked %d times, frame %q, want the mark written", len(b.pages), first)
	}
}

func TestForcedGDIWithoutAnOverlayDrawsCells(t *testing.T) {
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	first := func(opts ...twi.Option) string {
		b := newBackend(40, 15)
		b.caps = terminal.Capabilities{Identity: terminal.IdentityOther, CellPixels: image.Pt(10, 20)}
		r := run{b: b, clock: &clock{}, done: make(chan error, 1)}
		r.rt = twi.New(append([]twi.Option{twi.Backend(b, r.clock), twi.Styles(s), twi.ColorProfile(color.TrueColor)}, opts...)...)
		app := hello.Surfaces(r.rt)
		go func() { r.done <- r.rt.Run(app) }()
		f := r.next(t)
		if err := r.stop(t); err != nil {
			t.Fatal(err)
		}
		return f
	}
	if forced, cells := first(twi.Graphics(terminal.GraphicsGDI)), first(); forced != cells {
		t.Errorf("forcing GDI on a backend with no overlay wrote\n%q\nwant the cell frame\n%q", forced, cells)
	}
}

func TestConhostGDIPaintsSurfaces(t *testing.T) {
	for _, tc := range []struct {
		name   string
		caps   terminal.Capabilities
		pixels bool
	}{
		{"gdi", terminal.Capabilities{Identity: terminal.IdentityConhost, Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(10, 20)}, true},
		{"conhost without gdi", terminal.Capabilities{Identity: terminal.IdentityConhost}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := hello.Styles()
			if err != nil {
				t.Fatal(err)
			}
			b := gdiBackend{newBackend(40, 15), make(chan terminal.Pixels, 64)}
			b.caps = tc.caps
			r := run{b: b.backend, clock: &clock{}, done: make(chan error, 1)}
			r.rt = twi.New(twi.Backend(b, r.clock), twi.Styles(s), twi.ColorProfile(color.TrueColor))
			app := hello.Surfaces(r.rt)
			go func() { r.done <- r.rt.Run(app) }()
			first := r.next(t)
			if strings.Contains(first, "\x1bP") {
				t.Errorf("first frame wrote Sixel on conhost")
			}
			if err := r.stop(t); err != nil {
				t.Fatal(err)
			}
			if got := len(b.paints); got > 0 != tc.pixels {
				t.Fatalf("%d paints, want pixels %v", got, tc.pixels)
			}
			if !tc.pixels {
				return
			}
			p := <-b.paints
			drawn := 0
			for _, tile := range p.Tiles {
				if tile.Pix != nil {
					drawn++
				}
			}
			if !p.Clear || p.Cell != image.Pt(10, 20) || drawn == 0 {
				t.Errorf("first paint clear %v cell %v with %d drawn tiles, want a clear, 10x20 and tiles", p.Clear, p.Cell, drawn)
			}
		})
	}
}
