package paint

import (
	"image"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/paint"
	stylekonst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func everyFeature(t *testing.T) scene.Node {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	styled := func(bs style.BorderStyle, r style.Radius) style.ComputedStyle {
		s := filled(white)
		s.BorderStyle, s.BorderColor, s.Radius = bs, literal(zinc800), r
		return s
	}
	md := style.Shadow{Y: 4, Blur: 6, Spread: -1, Color: literal(color.RGBA{A: 26})}
	inset := styled(style.BorderSingle, style.RadiusNone)
	inset.InsetShadows = []style.Shadow{{Y: 16, Color: literal(shadowMd), Inset: true}}
	pill := filled(blue)
	pill.Radius = style.RadiusFull
	bar := indigoToPink(t, style.ToBottom)
	return page(40, 16, "",
		card(place(1, 1, 8, 4, one), style.RadiusLg, md),
		card(place(11, 1, 8, 4, one), style.RadiusNone, ring(zinc800, 1)),
		scene.New(place(21, 1, 8, 4, one), styled(style.BorderDashed, style.RadiusMd), scene.Text{}),
		scene.New(place(31, 1, 8, 4, one), styled(style.BorderDotted, style.RadiusNone), scene.Text{}),
		scene.New(place(1, 7, 8, 4, one), styled(style.BorderDouble, style.RadiusNone), scene.Text{}),
		scene.New(place(11, 7, 8, 4, one), inset, scene.Text{}),
		ringed(22, 8, 6, focusRing(blue, 128)...),
		scene.New(place(31, 8, 6, 1, layout.Edges{}), pill, scene.Text{}),
		scene.New(place(1, 13, 8, 2, layout.Edges{}), bar, scene.Text{}),
		card(place(11, 13, 8, 1, layout.Edges{Left: 1, Right: 1}), style.RadiusMd),
	)
}

func TestGlyphEveryDrawnGlyphIsInConsoleFonts(t *testing.T) {
	for _, look := range []Look{Composited, Plain, Glyphs} {
		buf := painted(40, 16, everyFeature(t), look)
		for y := range buf.Height() {
			for x, c := range buf.Row(y) {
				if r, n := utf8.DecodeRuneInString(c.Grapheme); n > 1 && !strings.ContainsRune(konst.ConsoleGlyphs, r) {
					t.Errorf("look %d cell %d,%d: %q is missing from Consolas, Cascadia Mono or JetBrains Mono", look, x, y, c.Grapheme)
				}
			}
		}
	}
}

func TestGlyphStandInsAreSafeAndOneCell(t *testing.T) {
	missing, standIns := []rune(konst.ConsoleMissing), []rune(konst.ConsoleStandIns)
	if len(missing) != len(standIns) {
		t.Fatalf("%d missing glyphs, %d stand-ins", len(missing), len(standIns))
	}
	for i, r := range missing {
		if strings.ContainsRune(konst.ConsoleGlyphs, r) {
			t.Errorf("%q is listed as missing and as safe", r)
		}
		if s := standIns[i]; s >= utf8.RuneSelf && !strings.ContainsRune(konst.ConsoleGlyphs, s) {
			t.Errorf("stand-in %q for %q is not in the console fonts", s, r)
		}
	}
}

func TestGlyphStandInOnlyUnderSixteenColours(t *testing.T) {
	text := "⌄ ◐ ⣾ ✕ ok 中"
	for _, c := range []struct {
		profile color.Profile
		want    string
	}{
		{color.ANSI16, "▾ ● | × ok 中 "},
		{color.ANSI256, text + " "},
		{color.TrueColor, text + " "},
	} {
		p := Painter{Profile: c.profile}
		buf := buffer.New(14, 1)
		root := page(14, 1, text)
		p.Paint(buf, &root, Composited)
		if got := strings.Join(rows(buf), ""); got != c.want {
			t.Errorf("profile %d: %q, want %q", c.profile, got, c.want)
		}
	}
}

func surfaceCoverage(got, surface, under color.RGBA) float64 {
	d := [3]float64{float64(surface.R) - float64(under.R), float64(surface.G) - float64(under.G), float64(surface.B) - float64(under.B)}
	g := [3]float64{float64(got.R) - float64(under.R), float64(got.G) - float64(under.G), float64(got.B) - float64(under.B)}
	return min(max((g[0]*d[0]+g[1]*d[1]+g[2]*d[2])/(d[0]*d[0]+d[1]*d[1]+d[2]*d[2]), 0), 1)
}

func pixelLook(t *testing.T, root scene.Node, w, h int) func(x, y int) color.RGBA {
	t.Helper()
	cell := image.Pt(stylekonst.NominalCellX, stylekonst.NominalCellY)
	var f scene.Frame
	f.Record(&root, cell)
	if len(f.Layers) != 1 {
		t.Fatalf("%d layers, want the page alone", len(f.Layers))
	}
	img := image.NewRGBA(image.Rect(0, 0, w*cell.X, h*cell.Y))
	new(raster.Raster).Draw(img, f.Layers[0].Ops, img.Rect)
	return func(x, y int) color.RGBA {
		return raster.Mean(img, image.Rect(x*cell.X, y*cell.Y, (x+1)*cell.X, (y+1)*cell.Y))
	}
}

func cellCoverage(c buffer.Cell, surface, line color.RGBA) float64 {
	on := func(k color.Color) float64 {
		if k.Kind == color.Literal && (k.RGBA == surface || k.RGBA == line) {
			return 1
		}
		return 0
	}
	if r, _ := utf8.DecodeRuneInString(c.Grapheme); strings.ContainsRune(konst.RoundedCorners+konst.SquareCorners+konst.SingleLines, r) && on(c.Bg) == 0 {
		return on(c.Fg) / 2
	}
	return on(c.Bg)
}

func TestCellLookCoversWhatPixelsCover(t *testing.T) {
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	pill := filled(blue)
	pill.Radius = style.RadiusFull
	lined := filled(blue)
	lined.BorderStyle, lined.BorderColor = style.BorderSingle, literal(zinc800)
	rounded := lined
	rounded.Radius = style.RadiusLg
	for _, c := range []struct {
		name string
		node scene.Node
	}{
		{"card", scene.New(place(2, 1, 10, 4, one), rounded, scene.Text{})},
		{"square card", scene.New(place(2, 1, 10, 4, one), lined, scene.Text{})},
		{"bottom border", scene.New(place(2, 1, 10, 3, layout.Edges{Bottom: 1}), lined, scene.Text{})},
		{"pill", scene.New(place(3, 2, 8, 1, layout.Edges{}), pill, scene.Text{})},
		{"one-row box", scene.New(place(2, 2, 10, 1, layout.Edges{Left: 1, Right: 1}), lined, scene.Text{})},
	} {
		root := page(14, 6, "", c.node)
		cells, pixels := painted(14, 6, root, Composited), pixelLook(t, root, 14, 6)
		for y := range 6 {
			for x := range 14 {
				want := surfaceCoverage(pixels(x, y), blue, zinc100)
				if got := cellCoverage(cells.At(x, y), blue, zinc800); math.Abs(got-want) > 0.5 {
					t.Errorf("%s cell %d,%d: the cell look covers %.2f of it, pixels cover %.2f", c.name, x, y, got, want)
				}
			}
		}
		t.Logf("%s, cell look:\n%s", c.name, strings.Join(rows(cells), "\n"))
	}
}

func TestCellLookPillCapsSitInside(t *testing.T) {
	s := filled(blue)
	s.Radius = style.RadiusFull
	box := place(1, 0, 6, 1, layout.Edges{})
	box.ContentBox = layout.Rect{X: 2, W: 4, H: 1}
	buf := painted(8, 1, page(8, 1, "", scene.New(box, s, scene.Sanitize("ab"))), Composited)
	expect(t, buf, "  ab    ")
	for x := range 8 {
		want := literal(blue)
		if x == 0 || x == 7 {
			want = literal(zinc100)
		}
		if c := buf.At(x, 0); c.Bg != want {
			t.Errorf("cell %d: bg %+v, want %+v; the pill fills its box and nothing outside it", x, c.Bg, want)
		}
	}
}

func index16(c color.Color) uint8 { return c.RGBA.ANSI16() }

func TestPalette16KeepsSurfacesAndTextApart(t *testing.T) {
	dark := struct{ page, card, border, text, muted, accent color.RGBA }{
		page:   color.RGBA{R: 9, G: 9, B: 11, A: 255},
		card:   color.RGBA{R: 24, G: 24, B: 27, A: 255},
		border: color.RGBA{R: 39, G: 39, B: 42, A: 255},
		text:   color.RGBA{R: 250, G: 250, B: 250, A: 255},
		muted:  color.RGBA{R: 161, G: 161, B: 170, A: 255},
		accent: color.RGBA{R: 39, G: 39, B: 42, A: 128},
	}
	one := layout.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	s := filled(dark.page)
	s.Color = literal(dark.text)
	card := filled(dark.card)
	card.BorderStyle, card.BorderColor, card.Radius, card.Color = style.BorderSingle, literal(dark.border), style.RadiusLg, literal(dark.muted)
	hover := filled(dark.accent)
	hover.Color = literal(dark.text)
	root := scene.New(place(0, 0, 20, 6, layout.Edges{}), s, scene.Sanitize("page"))
	root.Children = []scene.Node{
		scene.New(place(1, 1, 12, 4, one), card, scene.Sanitize("muted")),
		scene.New(place(14, 2, 5, 1, layout.Edges{}), hover, scene.Sanitize("row")),
	}
	for _, profile := range []color.Profile{color.ANSI16, color.TrueColor} {
		p := Painter{Profile: profile}
		buf := buffer.New(20, 6)
		p.Paint(buf, &root, Composited)
		pageCell, cardCell, borderCell, mutedCell, rowCell := buf.At(0, 0), buf.At(3, 2), buf.At(4, 1), buf.At(2, 2), buf.At(14, 2)
		if profile == color.TrueColor {
			if cardCell.Bg != literal(dark.card) || borderCell.Fg != literal(dark.border) || mutedCell.Fg != literal(dark.muted) {
				t.Errorf("truecolor changed colours: card %+v border %+v muted %+v", cardCell.Bg, borderCell.Fg, mutedCell.Fg)
			}
			continue
		}
		for _, pair := range []struct {
			what string
			a, b color.Color
		}{
			{"card on the page", cardCell.Bg, pageCell.Bg},
			{"border on the card", borderCell.Fg, borderCell.Bg},
			{"muted text on the card", mutedCell.Fg, mutedCell.Bg},
			{"hover row on the page", rowCell.Bg, pageCell.Bg},
			{"text on the hover row", rowCell.Fg, rowCell.Bg},
		} {
			if index16(pair.a) == index16(pair.b) {
				t.Errorf("16 colours: %s both map to index %d (%+v, %+v)", pair.what, index16(pair.a), pair.a.RGBA, pair.b.RGBA)
			}
		}
		if index16(pageCell.Bg) != dark.page.ANSI16() {
			t.Errorf("the page moved to index %d, want %d", index16(pageCell.Bg), dark.page.ANSI16())
		}
		t.Logf("16 colours: page %d card %d border %d muted %d row %d", index16(pageCell.Bg), index16(cardCell.Bg), index16(borderCell.Fg), index16(mutedCell.Fg), index16(rowCell.Bg))
	}
}

func TestPalette16DimsNeverLighten(t *testing.T) {
	black := color.RGBA{R: 9, G: 9, B: 11, A: 255}
	backdrop := filled(color.RGBA{A: 128})
	root := filled(black)
	n := scene.New(place(0, 0, 4, 1, layout.Edges{}), root, scene.Text{})
	n.Children = []scene.Node{scene.New(place(0, 0, 4, 1, layout.Edges{}), backdrop, scene.Text{})}
	p := Painter{Profile: color.ANSI16}
	buf := buffer.New(4, 1)
	p.Paint(buf, &n, Composited)
	if c := buf.At(1, 0); c.Bg.RGBA.ANSI16() != black.ANSI16() {
		t.Errorf("a black backdrop over a black page moved to index %d: a dim must not lighten", c.Bg.RGBA.ANSI16())
	}
}
