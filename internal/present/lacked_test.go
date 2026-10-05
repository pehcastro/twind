package present

import (
	"bytes"
	"image"
	"testing"

	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/terminal"
)

func TestLackedLettersArePixelGlyphsUnderGDI(t *testing.T) {
	s, out, p := gdiScreen()
	s.Covers = func(cluster string) bool { return cluster != "ש" && cluster != "ל" }
	page, fg := ink(13, 13, 18, 255), ink(0, 0, 0, 255)
	paint := func(root scene.Node) (tiles int, glyphs []terminal.Glyph) {
		from := len(p.calls)
		frame(t, s, root)
		for _, call := range p.calls[from:] {
			for _, tile := range call.Tiles {
				if len(tile.Glyphs) > 0 && tile.Pix == nil {
					t.Fatalf("tile %v carries glyphs and no pixels", tile.Cells)
				}
				tiles, glyphs = tiles+1, append(glyphs, tile.Glyphs...)
			}
		}
		return tiles, glyphs
	}
	lamed, shin := terminal.Glyph{Cell: image.Pt(1, 0), Cluster: "ל", Fg: fg}, terminal.Glyph{Cell: image.Pt(2, 0), Cluster: "ש", Fg: fg}
	if _, got := paint(flatPage(page, "aשל")); len(got) != 2 || got[0] != lamed || got[1] != shin {
		t.Fatalf("glyphs %+v, want %+v and %+v, right to left", got, lamed, shin)
	}
	shin.Cell.X = 1
	if b := out.last(); bytes.Contains(b, []byte("ש")) || bytes.Contains(b, []byte("ל")) || !bytes.Contains(b, []byte("a")) {
		t.Errorf("console bytes %q, want the covered a and no lacked letter", b)
	}
	if tiles, got := paint(flatPage(page, "aשל")); tiles != 0 || len(got) != 0 {
		t.Errorf("an unchanged frame sent %d tiles with glyphs %+v", tiles, got)
	}
	if _, got := paint(flatPage(page, "aש")); len(got) != 1 || got[0] != shin {
		t.Errorf("after the ל went away: glyphs %+v, want the tile again with only the ש", got)
	}
	if tiles, _ := paint(flatPage(page, "a")); tiles == 0 {
		t.Errorf("the last glyph going away sent no tile, its pixels stay on screen")
	}
}
