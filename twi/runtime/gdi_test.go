package runtime_test

import (
	"image"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/testdata/hello"
)

type gdiBackend struct {
	*backend
	paints chan terminal.Pixels
}

func (g gdiBackend) Paint(p terminal.Pixels) bool {
	g.paints <- p
	return true
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
