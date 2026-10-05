package present

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/paint"
	termkonst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

var white = color.RGBA{R: 255, G: 255, B: 255, A: 255}

func withRamp(bg color.RGBA, at layout.Rect) scene.Node {
	page, ramp := flatPage(bg, "ab"), flatPage(white, "")
	ramp.Bounds, ramp.Padding, ramp.Content = at, at, at
	ramp.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight},
		From:         style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, A: 255}}},
		To:           style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{B: 255, A: 255}}, Position: 1},
	}
	page.Children = []scene.Node{ramp}
	return page
}

func images(g terminal.Graphics, frame []byte) int {
	if g == terminal.GraphicsKitty {
		return bytes.Count(frame, []byte("\x1b_Ga=T")) + bytes.Count(frame, []byte("\x1b_Ga=p"))
	}
	return bytes.Count(frame, []byte("\x1b]1337;File="))
}

var pixelProtocols = []terminal.Graphics{terminal.GraphicsKitty, terminal.GraphicsITerm2}

func TestPlainTilesAreCellBackgrounds(t *testing.T) {
	for _, g := range pixelProtocols {
		for _, bg := range []color.RGBA{white, {}} {
			s, out := screen(g)
			frame(t, s, flatPage(bg, "abc"))
			want := color.Color{}
			if bg.A > 0 {
				want = color.Color{Kind: color.Literal, RGBA: bg}
			}
			if n := images(g, out.last()); n != 0 || s.imageBytes != 0 {
				t.Errorf("graphics %d, page %+v: %d images in %d bytes, want none: every tile is one colour", g, bg, n, s.imageBytes)
			}
			for _, at := range [][2]int{{0, 0}, {1, 0}, {5, 0}, {40, 10}, {cols - 1, rows - 1}} {
				if got := s.shown.At(at[0], at[1]).Bg; got != want {
					t.Errorf("graphics %d, page %+v: cell %v has bg %+v, want %+v", g, bg, at, got, want)
				}
			}
		}
	}
}

func TestTranslucentFlatTileIsAnImage(t *testing.T) {
	for _, g := range pixelProtocols {
		s, out := screen(g)
		frame(t, s, flatPage(color.RGBA{A: 26}, ""))
		if n := images(g, out.last()); n != len(s.tiles) {
			t.Errorf("graphics %d: a 10%% black page sent %d images, want all %d tiles: a translucent colour is no cell background", g, n, len(s.tiles))
		}
	}
}

func TestMixedTileKeepsTheDefaultBackground(t *testing.T) {
	s, out := screen(terminal.GraphicsKitty)
	frame(t, s, withRamp(white, layout.Rect{X: 3, Y: 0, W: 1, H: 1}))
	if n := images(terminal.GraphicsKitty, out.last()); n != 1 {
		t.Fatalf("one ramp cell on a white page sent %d images, want 1: its tile", n)
	}
	for x := range konst.TileColumns {
		if got := s.shown.At(x, 0).Bg; got != (color.Color{}) {
			t.Errorf("cell %d,0 shares a tile with the ramp and has bg %+v, want the default so the image is not blended", x, got)
		}
	}
	if got := s.shown.At(konst.TileColumns, 0).Bg; got.RGBA != white {
		t.Errorf("cell %d,0 is in the next tile and has bg %+v, want white", konst.TileColumns, got)
	}
}

func TestPlainAgainDropsTheImage(t *testing.T) {
	ramp := layout.Rect{X: 9, Y: 2, W: 3, H: 1}
	for _, g := range pixelProtocols {
		s, out := screen(g)
		frame(t, s, withRamp(white, ramp))
		sent := kittyControls(out.last())
		frame(t, s, flatPage(white, "ab"))
		last := out.last()
		if g == terminal.GraphicsKitty && (len(sent) != 1 || !bytes.Contains(last, []byte("\x1b_Ga=d,d=I,i="+sent[0]["i"]+",q=2\x1b\\"))) {
			t.Errorf("kitty: removing the ramp wrote %q, want its tile's image deleted", last)
		}
		if n := images(g, last); n != 0 {
			t.Errorf("graphics %d: removing the ramp sent %d images, want none", g, n)
		}
		for x := 8; x < 16; x++ {
			if c := s.shown.At(x, 2); c.Bg.RGBA != white || c == (buffer.Cell{}) {
				t.Errorf("graphics %d: cell %d,2 after the ramp left is %+v, want a white background", g, x, c)
			}
		}
		frame(t, s, withRamp(white, ramp))
		if n := images(g, out.last()); n != 1 {
			t.Errorf("graphics %d: the ramp back sent %d images, want its tile again", g, n)
		}
		if c := s.shown.At(9, 2); g == terminal.GraphicsKitty && c.Bg != (color.Color{}) {
			t.Errorf("kitty: cell 9,2 under the ramp again has bg %+v, want the default", c.Bg)
		}
		if c := s.shown.At(10, 2); g == terminal.GraphicsITerm2 && c.Grapheme != "" {
			t.Errorf("iterm2: blank cell 10,2 under the ramp again is %+v, want it left to the image", c)
		}
	}
}

func kittyControls(frame []byte) []map[string]string {
	var all []map[string]string
	for _, m := range regexp.MustCompile("\x1b_G([^;\x1b]*)").FindAllSubmatch(frame, -1) {
		keys := map[string]string{}
		for kv := range strings.SplitSeq(string(m[1]), ",") {
			k, v, _ := strings.Cut(kv, "=")
			keys[k] = v
		}
		all = append(all, keys)
	}
	return all
}

func TestKittyRepeatsArePlacements(t *testing.T) {
	ramps := func(at ...int) scene.Node {
		page := withRamp(white, layout.Rect{X: at[0], Y: 2, W: konst.TileColumns, H: 1})
		for _, x := range at[1:] {
			r := withRamp(white, layout.Rect{X: x, Y: 2, W: konst.TileColumns, H: 1}).Children[0]
			page.Children = append(page.Children, r)
		}
		return page
	}
	s, out := screen(terminal.GraphicsKitty)
	frame(t, s, ramps(0, 16, 32))
	var id string
	placed := map[string]bool{}
	for _, c := range kittyControls(out.last()) {
		if id == "" {
			id = c["i"]
		}
		if c["i"] != id || c["a"] != "T" && c["a"] != "p" || c["z"] != "-1" || c["C"] != "1" {
			t.Errorf("three equal ramps: command %v, want one image %s transmitted then placed, under the text, cursor kept", c, id)
		}
		placed[c["a"]+c["p"]] = true
	}
	p := func(x int) string { return strconv.Itoa(s.tileAt(x, 2) + 1) }
	if want := map[string]bool{"T" + p(0): true, "p" + p(16): true, "p" + p(32): true}; fmt.Sprint(placed) != fmt.Sprint(want) {
		t.Errorf("three equal ramps: actions and placements %v, want %v: one transmit, two placements, one per tile", placed, want)
	}
	for _, step := range []struct {
		name  string
		root  scene.Node
		wants []string
	}{
		{"the middle ramp leaves", ramps(0, 32), []string{"a=d,d=i,i=" + id + ",p=" + p(16) + ","}},
		{"the first ramp leaves", ramps(32), []string{"a=d,d=i,i=" + id + ",p=" + p(0) + ","}},
		{"the last ramp leaves", ramps(40), []string{"a=d,d=I,i=" + id + ",", "a=T,"}},
	} {
		frame(t, s, step.root)
		for _, want := range step.wants {
			if !bytes.Contains(out.last(), []byte("\x1b_G"+want)) {
				t.Errorf("%s: wrote %q, want %q", step.name, out.last(), want)
			}
		}
		if n := bytes.Count(out.last(), []byte("\x1b_Ga=d")); n != 1 {
			t.Errorf("%s: %d deletes, want 1", step.name, n)
		}
	}
	last := kittyControls(out.last())
	if err := s.Frame(ramps(40), cols-10, rows); err != nil {
		t.Fatal(err)
	}
	if want := "\x1b_Ga=d,d=I,i=" + last[len(last)-1]["i"] + ","; !bytes.HasPrefix(out.last()[len(termkonst.SyncBegin):], []byte(want)) {
		t.Errorf("a resize wrote %q first, want the image on screen deleted: %q", out.last()[:min(len(out.last()), 80)], want)
	}
}

func TestPlainColourChangeRewritesCells(t *testing.T) {
	grey := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	for _, g := range pixelProtocols {
		s, out := screen(g)
		frame(t, s, flatPage(white, ""))
		frame(t, s, flatPage(grey, ""))
		if n := images(g, out.last()); n != 0 {
			t.Errorf("graphics %d: a white page turned grey sent %d images, want none", g, n)
		}
		for _, at := range [][2]int{{0, 0}, {40, 10}, {cols - 1, rows - 1}} {
			if got := s.shown.At(at[0], at[1]).Bg; got.RGBA != grey {
				t.Errorf("graphics %d: cell %v has bg %+v after the page turned grey", g, at, got)
			}
		}
	}
}

func TestPlainOnlyAtTrueColor(t *testing.T) {
	for _, g := range pixelProtocols {
		s, out := screen(g)
		s.Profile = color.ANSI256
		frame(t, s, flatPage(white, ""))
		if n := images(g, out.last()); n != len(s.tiles) {
			t.Errorf("graphics %d at ANSI256: %d images, want all %d tiles: a cell colour goes through the terminal palette", g, n, len(s.tiles))
		}
	}
}
