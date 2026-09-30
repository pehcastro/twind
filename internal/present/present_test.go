package present

import (
	"bytes"
	"image"
	"regexp"
	"testing"

	paintkonst "github.com/twind-dev/twind/internal/konst/paint"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

const (
	cols, rows = 90, 28
	kibi       = 1 << 10
)

var wt = image.Pt(10, 20)

type writes struct{ frames [][]byte }

func (w *writes) Write(p []byte) (int, error) {
	w.frames = append(w.frames, bytes.Clone(p))
	return len(p), nil
}

func (w *writes) last() []byte { return w.frames[len(w.frames)-1] }

func tree(t testing.TB, n render.Node) scene.Node {
	t.Helper()
	sheet, err := demo.Styles()
	if err != nil {
		t.Fatal(err)
	}
	root, err := render.Scene(n, render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func screen(g terminal.Graphics) (*Screen, *writes) {
	out := &writes{}
	return &Screen{Out: out, Profile: color.TrueColor, Graphics: g, Cell: wt, Sync: true}, out
}

func (s *Screen) image() *image.RGBA {
	img := image.NewRGBA(s.bounds)
	for y := range s.bounds.Dy() {
		for x := 0; x < s.bounds.Dx(); x += paintkonst.TileColumns * s.Cell.X {
			copy(img.Pix[img.PixOffset(x, y):], s.column(x).line(y))
		}
	}
	return img
}

func frame(t testing.TB, s *Screen, root scene.Node) {
	t.Helper()
	if err := s.Frame(root, cols, rows); err != nil {
		t.Fatal(err)
	}
}

func withText(n render.Node, from, to string) render.Node {
	if n.Text == from {
		n.Text = to
	}
	children := n.Children
	n.Children = make([]render.Node, len(children))
	for i, c := range children {
		n.Children[i] = withText(c, from, to)
	}
	return n
}

func TestTextOnlyFrameSendsNoImage(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.Dialog()))
	rastered := s.rastered
	for i, typed := range []string{"Dashboard  Projects  Settings  Help", "Dashboard  Projects"} {
		frame(t, s, tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", typed)))
		if s.imageBytes != 0 || s.rastered != rastered || len(out.frames) != i+2 {
			t.Errorf("typing %q: %d writes, %d image bytes and %d boxes rastered, want a write, 0 and 0", typed, len(out.frames), s.imageBytes, s.rastered-rastered)
		}
		if bytes.Contains(out.last(), []byte("\x1bP")) {
			t.Errorf("typing %q wrote a Sixel image: %q", typed, out.last())
		}
	}
}

func TestUnchangedBoxNeverRastered(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.List(-1)))
	first := s.rastered
	frame(t, s, tree(t, demo.List(0)))
	if got := s.rastered - first; got != 1 {
		t.Errorf("a hover on one row rastered %d boxes, want 1: the new row fill only", got)
	}
	for hover := 1; hover < 5; hover++ {
		before := s.rastered
		frame(t, s, tree(t, demo.List(hover)))
		if got := s.rastered - before; got != 0 {
			t.Errorf("moving the hover to row %d rastered %d boxes, want 0: the row fill is cached by look", hover, got)
		}
	}
	if first != 2 {
		t.Errorf("the first frame rastered %d boxes, want 2: the page and the card", first)
	}
}

func TestRowHoverSendsOnlyItsTiles(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.List(0)))
	frame(t, s, tree(t, demo.List(1)))
	images := bytes.Count(out.last(), []byte("\x1bP"))
	if images == 0 || images > 6 || s.imageBytes > 16*kibi {
		t.Errorf("moving the hover one row sent %d tiles in %d image bytes, want at most 6 tiles and 16 KB", images, s.imageBytes)
	}
	if !bytes.Contains(out.last(), []byte("Open")) {
		t.Errorf("the rows under the re-sent tiles were not written again: %q", out.last())
	}
}

func TestFirstFrameBudget(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.Dialog()))
	if n := len(out.last()); n > 200*kibi || s.imageBytes == 0 {
		t.Errorf("first frame of the dialog wrote %d bytes, %d of them image, want under 200 KB with images", n, s.imageBytes)
	}
	if len(out.frames) != 1 || !bytes.HasPrefix(out.last(), []byte(termkonst.SyncBegin)) || !bytes.HasSuffix(out.last(), []byte(termkonst.SyncEnd)) {
		t.Errorf("first frame in %d writes, want one synchronized write", len(out.frames))
	}
}

func TestSixelTextTakesTheRegisterColour(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.Dialog()))
	img := s.image()
	percent := func(v uint8) uint8 {
		sent := (int(v)*100 + 127) / 255
		return uint8((sent*255 + 50) / 100)
	}
	found := 0
	for y := range rows {
		for x, c := range s.shown.Row(y) {
			if blank(s.text.At(x, y)) && c != (buffer.Cell{}) {
				t.Fatalf("blank cell %d,%d written as %+v, want it left to the image", x, y, c)
			}
			if c == (buffer.Cell{}) || c.Width == buffer.Continuation {
				continue
			}
			found++
			m := img.RGBAAt(x*wt.X+wt.X/2, y*wt.Y+wt.Y/2)
			want := color.RGBA{R: percent(m.R), G: percent(m.G), B: percent(m.B), A: 255}
			if c.Bg.RGBA != want && s.flat(x, y) {
				t.Fatalf("text %q at %d,%d has bg %+v, want the register colour %+v of the flat surface under it", c.Grapheme, x, y, c.Bg.RGBA, want)
			}
		}
	}
	if found < 100 {
		t.Errorf("found %d glyph cells, want the whole dialog text", found)
	}
}

func TestANSI256SurfaceTakesThePaletteColour(t *testing.T) {
	cube := map[uint8]bool{0: true, 95: true, 135: true, 175: true, 215: true, 255: true}
	palette := func(c color.RGBA) bool {
		grey := c.R == c.G && c.G == c.B && c.R >= 8 && c.R <= 238 && (c.R-8)%10 == 0
		return grey || cube[c.R] && cube[c.G] && cube[c.B]
	}
	full, _ := screen(terminal.GraphicsSixel)
	frame(t, full, tree(t, demo.Dialog()))
	s, _ := screen(terminal.GraphicsSixel)
	s.Profile = color.ANSI256
	frame(t, s, tree(t, demo.Dialog()))
	img := s.image()
	for i := 0; i < len(img.Pix); i += 4 {
		p := img.Pix[i : i+4]
		if c := (color.RGBA{R: p[0], G: p[1], B: p[2], A: p[3]}); c.A == 255 && !palette(c) {
			t.Fatalf("opaque surface pixel %+v at byte %d under ANSI256, want an xterm palette colour", c, i)
		}
	}
	for y := range rows {
		for x, c := range s.shown.Row(y) {
			if c == (buffer.Cell{}) || c.Width == buffer.Continuation || !s.flat(x, y) {
				continue
			}
			m := img.RGBAAt(x*wt.X, y*wt.Y)
			if under := (color.RGBA{R: m.R, G: m.G, B: m.B, A: 255}).ANSI256(); c.Bg.RGBA.ANSI256() != under {
				t.Fatalf("text %q at %d,%d has bg index %d over a flat surface of index %d", c.Grapheme, x, y, c.Bg.RGBA.ANSI256(), under)
			}
		}
	}
	if s.imageBytes > full.imageBytes {
		t.Errorf("ANSI256 surface sent %d image bytes, TrueColor %d: quantising must not grow the image", s.imageBytes, full.imageBytes)
	}
}

func TestCursorForgottenAfterImage(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, flatPage(white, "a"))
	tinted := flatPage(white, "a         x")
	tint := flatPage(color.RGBA{A: 26}, "")
	tint.Bounds = layout.Rect{X: 8, W: 8, H: 1}
	tinted.Children = []scene.Node{tint}
	frame(t, s, tinted)
	last := out.last()
	after := last[bytes.LastIndex(last, []byte("\x1b\\"))+2:]
	if !regexp.MustCompile(`^(\x1b\[[0-9;]*m)*\x1b\[\d+;\d+H`).Match(after) {
		t.Errorf("text after the last image starts with %q, want an absolute cursor move", after[:min(len(after), 24)])
	}
}

func TestBottomTilesFitWholeBands(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, tree(t, demo.Dialog()))
	for _, tile := range s.tiles {
		if tile.Max.Y == rows && tile.Dy()*wt.Y%6 != 0 {
			t.Errorf("tile %v on the bottom row is %d px tall, want a multiple of 6 so the terminal does not scroll", tile, tile.Dy()*wt.Y)
		}
	}
}

func flatPage(bg color.RGBA, text string) scene.Node {
	box := &layout.Box{BorderBox: layout.Rect{W: cols, H: rows}, PaddingBox: layout.Rect{W: cols, H: rows}, ContentBox: layout.Rect{W: cols, H: rows}, Clip: layout.Rect{W: cols, H: rows}}
	s := style.ComputedStyle{Opacity: 1, Background: color.Color{Kind: color.Literal, RGBA: bg}, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{A: 255}}}
	return scene.New(box, s, scene.Sanitize(text))
}

func TestGlyphTurnedBlank(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	s, out := screen(terminal.GraphicsSixel)
	frame(t, s, flatPage(white, "abc"))
	frame(t, s, flatPage(white, "ab"))
	if c := s.shown.At(2, 0); s.imageBytes != 0 || len(out.frames) != 2 || c.Grapheme != " " || c.Bg.RGBA != white {
		t.Errorf("erasing a glyph on a flat surface: %d image bytes, cell %+v, want 0 and a white space written", s.imageBytes, c)
	}
	ramp := flatPage(white, "abc")
	ramp.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight},
		From:         style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, A: 255}}},
		To:           style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{B: 255, A: 255}}, Position: 1},
	}
	s, _ = screen(terminal.GraphicsSixel)
	frame(t, s, ramp)
	shorter := flatPage(white, "ab")
	shorter.Gradient = ramp.Gradient
	frame(t, s, shorter)
	if s.imageBytes == 0 {
		t.Errorf("erasing a glyph on a gradient sent no image, want its tile again")
	}
}

func TestSamePixelsAreNotSentAgain(t *testing.T) {
	grey, black := color.RGBA{R: 200, G: 200, B: 200, A: 255}, color.RGBA{A: 255}
	pair := func(first, second int) scene.Node {
		page := flatPage(grey, "")
		for _, x := range []int{first, second} {
			b := flatPage(black, "")
			b.Bounds = layout.Rect{X: x, Y: 1, W: 4, H: 2}
			page.Children = append(page.Children, b)
		}
		return page
	}
	s, _ := screen(terminal.GraphicsSixel)
	frame(t, s, pair(0, 20))
	frame(t, s, pair(20, 0))
	if s.imageBytes != 0 {
		t.Errorf("swapping two identical boxes sent %d image bytes, want 0: the tiles hash the same", s.imageBytes)
	}
}

func TestResizeSendsEverything(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	root := tree(t, demo.Hello())
	frame(t, s, root)
	if err := s.Frame(root, cols-10, rows); err != nil {
		t.Fatal(err)
	}
	if s.imageBytes == 0 || s.image().Rect.Dx() != (cols-10)*wt.X {
		t.Errorf("after a resize: %d image bytes, surface %v, want a full frame at the new width", s.imageBytes, s.bounds)
	}
}

func TestKittyKeepsTextOverImagesTransparent(t *testing.T) {
	s, out := screen(terminal.GraphicsKitty)
	frame(t, s, tree(t, demo.Dialog()))
	if !bytes.Contains(out.last(), []byte("\x1b_G")) {
		t.Fatalf("no Kitty image in the first frame")
	}
	for y := range rows {
		for x, c := range s.shown.Row(y) {
			if c.Bg.Kind == color.Literal && s.sent[s.tileOf[y*cols+x]] != 0 {
				t.Fatalf("cell %d,%d has bg %+v over a Kitty image, want the default so the image shows", x, y, c.Bg)
			}
		}
	}
}

func TestNoGraphicsIsTheCellPaint(t *testing.T) {
	s, out := screen(terminal.GraphicsNone)
	root := tree(t, demo.Dialog())
	frame(t, s, root)
	want := buffer.New(cols, rows)
	(&paint.Painter{}).Paint(want, &root, paint.Composited)
	var expected bytes.Buffer
	never := buffer.New(cols, rows)
	never.Fill(buffer.Rect{W: cols, H: rows}, buffer.Cell{Grapheme: "\x00"})
	w := terminal.Writer{Out: &expected, Profile: color.TrueColor, Sync: true}
	if err := w.Diff(never, want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.last(), expected.Bytes()) {
		t.Errorf("graphics none wrote %d bytes, want the %d bytes of today's cell paint", len(out.last()), expected.Len())
	}
}

func TestTileTurnedTransparentIsErased(t *testing.T) {
	s, out := screen(terminal.GraphicsSixel)
	card := flatPage(color.RGBA{R: 255, A: 255}, "")
	card.Bounds, card.Padding, card.Content = layout.Rect{X: 2, Y: 2, W: 4, H: 2}, layout.Rect{X: 2, Y: 2, W: 4, H: 2}, layout.Rect{X: 2, Y: 2, W: 4, H: 2}
	bare := flatPage(color.RGBA{}, "")
	withCard := bare
	withCard.Children = []scene.Node{card}
	frame(t, s, withCard)
	frame(t, s, bare)
	if !bytes.Contains(out.last(), []byte("\x1b[3;1H\x1b[8X")) {
		t.Errorf("closing a box over the default background wrote %q, want its tile's cells erased", out.last())
	}
}

func TestOpacityGroupDoesNotDarkenOverlap(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	page := flatPage(color.RGBA{R: 255, G: 255, B: 255, A: 255}, "")
	group := flatPage(color.RGBA{}, "")
	group.Opacity = 0.5
	group.Bounds = layout.Rect{X: 0, Y: 0, W: 10, H: 4}
	a, b := flatPage(color.RGBA{A: 255}, ""), flatPage(color.RGBA{A: 255}, "")
	a.Bounds, b.Bounds = layout.Rect{X: 0, Y: 0, W: 6, H: 2}, layout.Rect{X: 3, Y: 1, W: 6, H: 2}
	group.Children = []scene.Node{a, b}
	page.Children = []scene.Node{group}
	frame(t, s, page)
	img := s.image()
	one, both := img.RGBAAt(15, 10), img.RGBAAt(45, 30)
	if one != both || one.R < 120 || one.R > 135 {
		t.Errorf("a black pair in an opacity-50 group over white: %+v alone, %+v where they overlap, want both mid grey", one, both)
	}
}
