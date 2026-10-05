package present

import (
	"testing"

	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

func TestTilesSeenBeforeAreNeitherComposedNorEncodedAgain(t *testing.T) {
	for _, id := range []terminal.Identity{terminal.IdentityOther, terminal.IdentityVSCode} {
		md := newModelled(id)
		for _, hover := range []int{1, 2, 1} {
			md.frame(t, tree(t, demo.List(hover)))
		}
		for i := range 4 {
			root := tree(t, demo.List(2-i%2))
			composed, encoded := md.s.composed.Load(), md.s.encoded.Load()
			images := md.frame(t, root)
			fresh := newModelled(id)
			fresh.frame(t, root)
			sameScreen(t, "hover back", md.m, fresh.m)
			if images == 0 {
				t.Errorf("identity %d, hover %d: no image sent", id, i)
			}
			if n := md.s.composed.Load() - composed; n != 0 {
				t.Errorf("identity %d, hover %d: %d tiles composed again, want 0: each was composed two frames before", id, i, n)
			}
			if n := md.s.encoded.Load() - encoded; id == terminal.IdentityOther && n != 0 {
				t.Errorf("identity %d, hover %d: %d images encoded again, want 0", id, i, n)
			}
		}
	}
}

func TestStoredTilesNeverCrossAPageColour(t *testing.T) {
	steps := []color.RGBA{{R: 250, G: 250, B: 250, A: 255}, {R: 10, G: 10, B: 30, A: 255}, {R: 250, G: 250, B: 250, A: 255}}
	md := newModelled(terminal.IdentityOther)
	for _, page := range steps {
		root := pageWithCard("stored")
		root.Background = color.Color{Kind: color.Literal, RGBA: page}
		md.frame(t, root)
		fresh := newModelled(terminal.IdentityOther)
		fresh.frame(t, root)
		sameScreen(t, "page colour", md.m, fresh.m)
	}
}

func TestStoreStaysBounded(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	for i := range 400 {
		root := flatPage(color.RGBA{R: 250, G: 250, B: 250, A: 255}, "")
		root.Children = []scene.Node{card(i%(cols-14), i%(rows-6), color.RGBA{R: uint8(i), G: uint8(3 * i), B: 90, A: 200})}
		frame(t, s, root)
	}
	if n := len(s.stored.tiles) + len(s.stored.images); n > 4*len(s.tiles) {
		t.Errorf("the store holds %d entries after 400 distinct frames, want at most %d", n, 4*len(s.tiles))
	}
}
