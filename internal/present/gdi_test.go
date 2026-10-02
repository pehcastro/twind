package present

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

type painted struct {
	calls  []terminal.Pixels
	refuse bool
}

func (p *painted) paint(px terminal.Pixels) bool {
	defer func() { p.refuse = false }()
	for i := range px.Tiles {
		px.Tiles[i].Pix = bytes.Clone(px.Tiles[i].Pix)
	}
	px.Tiles = slices.Clone(px.Tiles)
	p.calls = append(p.calls, px)
	return !p.refuse
}

func TestGDIRefusedPaintSendsEverythingNext(t *testing.T) {
	s, _, p := gdiScreen()
	frame(t, s, ramp(color.RGBA{A: 255}, "abc"))
	first := len(p.calls[0].Tiles)
	if g := p.calls[0].Grid; g != image.Pt(cols, rows) {
		t.Errorf("paint grid %v, want %v", g, image.Pt(cols, rows))
	}
	p.refuse = true
	frame(t, s, ramp(color.RGBA{A: 255}, "ab"))
	frame(t, s, ramp(color.RGBA{A: 255}, "ab"))
	if len(p.calls) != 3 || !p.calls[2].Clear || len(p.calls[2].Tiles) != first {
		t.Fatalf("after a refused paint: %d paints, last clear %v with %d tiles, want a third paint clearing and sending all %d", len(p.calls), p.calls[len(p.calls)-1].Clear, len(p.calls[len(p.calls)-1].Tiles), first)
	}
}

func (p *painted) tile(t *testing.T, cells image.Rectangle) terminal.Tile {
	t.Helper()
	last := p.calls[len(p.calls)-1]
	for _, tile := range last.Tiles {
		if tile.Cells == cells {
			return tile
		}
	}
	t.Fatalf("no tile at %v in the last paint, tiles %d", cells, len(last.Tiles))
	return terminal.Tile{}
}

func gdiScreen() (*Screen, *writes, *painted) {
	s, out := screen(terminal.GraphicsGDI)
	p := &painted{}
	s.Paint = p.paint
	return s, out, p
}

func ramp(page color.RGBA, text string) scene.Node {
	n := flatPage(page, text)
	n.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight},
		From:         style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, A: 255}}},
		To:           style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{B: 255, A: 255}}, Position: 1},
	}
	return n
}

func cellAlpha(tile terminal.Tile, cell image.Point, x int) []byte {
	stride := tile.Cells.Dx() * cell.X * 4
	var alphas []byte
	for y := range cell.Y {
		for px := x * cell.X; px < (x+1)*cell.X; px++ {
			alphas = append(alphas, tile.Pix[y*stride+px*4+3])
		}
	}
	return alphas
}

func all(alphas []byte, v byte) bool {
	return !slices.ContainsFunc(alphas, func(a byte) bool { return a != v })
}

func TestGDITextCellsAreLeftToTheConsole(t *testing.T) {
	s, out, p := gdiScreen()
	frame(t, s, ramp(color.RGBA{A: 255}, "ab中"))
	if len(p.calls) != 1 || !p.calls[0].Clear || p.calls[0].Cell != wt {
		t.Fatalf("first frame painted %d times, want once with Clear and cell %v", len(p.calls), wt)
	}
	if bytes.Contains(out.last(), []byte("\x1bP")) || bytes.Contains(out.last(), []byte("\x1b_G")) || bytes.Contains(out.last(), []byte("\x1b]1337")) {
		t.Errorf("GDI frame wrote an image escape: %q", out.last())
	}
	tile := p.tile(t, image.Rect(0, 0, 8, 1))
	if len(tile.Pix) != 8*wt.X*wt.Y*4 {
		t.Fatalf("tile has %d bytes, want %d", len(tile.Pix), 8*wt.X*wt.Y*4)
	}
	for x := range 8 {
		text := x < 4
		if a := cellAlpha(tile, wt, x); text && !all(a, 0) || !text && !all(a, 255) {
			t.Errorf("cell %d text %v: alphas %v, want all %v", x, text, a[:4], map[bool]int{true: 0, false: 255}[text])
		}
	}
	if c := s.shown.At(0, 0); c.Grapheme != "a" || c.Bg.RGBA.A != 255 || c.Bg.RGBA.R < 200 {
		t.Errorf("text cell %+v, want the glyph on the sampled ramp colour", c)
	}
}

func TestGDIMaskChangeSendsTheTileAgain(t *testing.T) {
	s, _, p := gdiScreen()
	frame(t, s, ramp(color.RGBA{A: 255}, "abc"))
	frame(t, s, ramp(color.RGBA{A: 255}, "ab"))
	if len(p.calls) != 2 {
		t.Fatalf("%d paints, want 2: the glyph turned blank over pixels", len(p.calls))
	}
	if a := cellAlpha(p.tile(t, image.Rect(0, 0, 8, 1)), wt, 2); !all(a, 255) {
		t.Errorf("cell 2 after its glyph went: alphas %v, want covered", a[:4])
	}
	if len(p.calls[1].Tiles) != 1 || p.calls[1].Clear {
		t.Errorf("second paint has %d tiles, clear %v, want only the changed tile", len(p.calls[1].Tiles), p.calls[1].Clear)
	}
	frame(t, s, ramp(color.RGBA{A: 255}, "ab"))
	if len(p.calls) != 2 {
		t.Errorf("an unchanged frame painted again")
	}
}

func TestGDIPageColourIsTransparent(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	withBox := func(on bool) scene.Node {
		page := flatPage(white, "")
		if on {
			box := ramp(white, "")
			box.Bounds, box.Padding, box.Content = layout.Rect{X: 2, W: 4, H: 1}, layout.Rect{X: 2, W: 4, H: 1}, layout.Rect{X: 2, W: 4, H: 1}
			page.Children = []scene.Node{box}
		}
		return page
	}
	s, _, p := gdiScreen()
	frame(t, s, withBox(true))
	tile := p.tile(t, image.Rect(0, 0, 8, 1))
	for x := range 8 {
		box := x >= 2 && x < 6
		if a := cellAlpha(tile, wt, x); box && !all(a, 255) || !box && !all(a, 0) {
			t.Errorf("cell %d box %v: alphas %v", x, box, a[:4])
		}
	}
	for i := 0; i < len(tile.Pix); i += 4 {
		if a := tile.Pix[i+3]; a != 0 && a != 255 {
			t.Fatalf("pixel %d has alpha %d, want 0 or 255", i/4, a)
		}
	}
	frame(t, s, withBox(false))
	if got := p.tile(t, image.Rect(0, 0, 8, 1)); got.Pix != nil {
		t.Errorf("a tile turned page colour kept %d bytes of pixels, want it handed back to the console", len(got.Pix))
	}
	if err := s.Frame(withBox(true), cols-10, rows); err != nil {
		t.Fatal(err)
	}
	if !p.calls[len(p.calls)-1].Clear {
		t.Errorf("a resize did not clear what was drawn")
	}
}

func identities(t *testing.T) []string {
	var lines []string
	dialog := tree(t, demo.Dialog())
	trees := append([]scene.Node{dialog, tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", "Dashboard")), tree(t, demo.List(0)), tree(t, demo.List(1))}, scrolled(t, 0, 1, 2, 10)...)
	for _, id := range []terminal.Identity{terminal.IdentityOther, terminal.IdentityConhost, terminal.IdentityInboxConPTY, terminal.IdentityZed} {
		for _, g := range []terminal.Graphics{terminal.GraphicsNone, terminal.GraphicsSixel, terminal.GraphicsITerm2, terminal.GraphicsKitty} {
			s, out := screen(g)
			s.Identity = id
			for _, root := range trees {
				frame(t, s, root)
			}
			sum := sha256.New()
			for _, f := range out.frames {
				sum.Write(f)
			}
			lines = append(lines, fmt.Sprintf("identity %d graphics %d frames %d %s", id, g, len(out.frames), hex.EncodeToString(sum.Sum(nil)[:8])))
		}
	}
	return lines
}

func TestGDIRoundedCornerCellsShowThePage(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	page := flatPage(white, "")
	pill := flatPage(color.RGBA{A: 255}, "")
	pill.Bounds, pill.Padding, pill.Content = layout.Rect{X: 2, Y: 2, W: 6, H: 1}, layout.Rect{X: 2, Y: 2, W: 6, H: 1}, layout.Rect{X: 2, Y: 2, W: 6, H: 1}
	pill.Border.Radius = style.RadiusFull
	page.Children = []scene.Node{pill}
	s, _, p := gdiScreen()
	frame(t, s, page)
	tile := p.tile(t, image.Rect(0, 2, 8, 3))
	if a := cellAlpha(tile, wt, 2); all(a, 0) || all(a, 255) {
		t.Fatalf("the pill's end cell is not partly drawn: alphas %v", a)
	}
	for _, x := range []int{2, 7} {
		if bg := s.shown.At(x, 2).Bg; bg.RGBA != white {
			t.Errorf("cell %d under a rounded end has bg %+v, want the page so no square shows around the curve", x, bg.RGBA)
		}
	}
	if bg := s.shown.At(4, 2).Bg; bg.RGBA != (color.RGBA{A: 255}) {
		t.Errorf("cell inside the pill has bg %+v, want its fill", bg.RGBA)
	}
}

func TestZedMarksAreWrittenForOneFrameAndWiped(t *testing.T) {
	page := color.RGBA{R: 5, G: 4, B: 6, A: 255}
	mark := color.RGBA{R: 6, G: 4, B: 6, A: 255}
	var asked []color.RGBA
	marking := true
	s, out := screen(terminal.GraphicsNone)
	s.Marks = func(p color.RGBA) []terminal.Mark {
		asked = append(asked, p)
		if !marking {
			return nil
		}
		return []terminal.Mark{{Cell: image.Pt(0, 0), Color: mark}, {Cell: image.Pt(2, 0), Color: mark}}
	}
	frame(t, s, flatPage(page, "a中b"))
	if !bytes.Contains(out.last(), []byte("48;2;6;4;6")) || len(asked) != 1 || asked[0] != page {
		t.Fatalf("marked frame %q, asked %v, want the mark colour written for page %v", out.last(), asked, page)
	}
	for x := range 3 {
		if c := s.shown.At(x, 0); c.Grapheme != " " || x != 1 && c.Bg.RGBA != mark {
			t.Errorf("cell %d %+v, want a blank, marked unless it was the cut wide glyph's head", x, c)
		}
	}
	marking = false
	frame(t, s, flatPage(page, "a中b"))
	if c := s.shown.At(0, 0); c.Grapheme != "a" || c.Bg.RGBA != page || s.shown.At(1, 0).Grapheme != "中" {
		t.Errorf("after the marks: %+v and %q, want the text back on the page", c, s.shown.At(1, 0).Grapheme)
	}
	asked = nil
	frame(t, s, flatPage(color.RGBA{R: 5, G: 4, B: 6, A: 128}, "a"))
	s.Profile = color.ANSI256
	frame(t, s, flatPage(page, "a"))
	if len(asked) != 0 {
		t.Errorf("marks asked for a translucent page or a 256-colour profile: %v", asked)
	}
}

func TestNonGDIBytesUnchanged(t *testing.T) {
	fixture, err := os.ReadFile("testdata/identities.txt")
	if err != nil {
		t.Fatal(err)
	}
	want, got := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(fixture), "\r", "")), "\n"), identities(t)
	if len(got) != len(want) {
		t.Fatalf("%d combinations, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("bytes differ from HEAD: got %q, want %q", got[i], want[i])
		}
	}
}
