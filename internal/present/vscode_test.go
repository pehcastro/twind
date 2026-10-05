package present

import (
	"bytes"
	"image"
	"regexp"
	"strconv"
	"testing"

	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
)

type vscodeStep struct {
	name string
	root scene.Node
}

func swatches(shift int) scene.Node {
	p := flatPage(ink(5, 4, 6, 255), "")
	for y := range 6 {
		for x := range cols {
			p.Children = append(p.Children, box(x, y+2, 1, 1, ink(uint8(x*255/cols), uint8(y*40+shift), uint8((x*7+y*13+shift)%256), 255)))
		}
	}
	return p
}

func vscodeSteps(t *testing.T) []vscodeStep {
	steps := []vscodeStep{{"dialog", tree(t, demo.Dialog())}, {"typed", tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", "Dashboard"))}}
	for hover := range 5 {
		steps = append(steps, vscodeStep{"hover " + strconv.Itoa(hover), tree(t, demo.List(hover))})
	}
	steps = append(steps, vscodeStep{"page", tree(t, demo.Page())}, vscodeStep{"cards", tree(t, demo.Cards("flex-row", 6))})
	steps = append(steps, vscodeStep{"swatches", swatches(0)}, vscodeStep{"swatches moved", swatches(9)}, vscodeStep{"dialog again", tree(t, demo.Dialog())})
	return steps
}

type modelled struct {
	s   *Screen
	out *writes
	m   *xterm
}

func newModelled(id terminal.Identity) *modelled {
	s, out := screen(terminal.GraphicsSixel)
	s.Identity = id
	return &modelled{s, out, newXterm(wt)}
}

func (md *modelled) frame(t *testing.T, root scene.Node) (images int) {
	t.Helper()
	before, added := len(md.out.frames), md.m.added
	frame(t, md.s, root)
	if len(md.out.frames) > before {
		md.m.write(t, md.out.last())
	}
	return md.m.added - added
}

func TestVSCodeShowsWhatOneImagePerTileShows(t *testing.T) {
	other, vscode := newModelled(terminal.IdentityOther), newModelled(terminal.IdentityVSCode)
	var fewer bool
	for _, step := range vscodeSteps(t) {
		tiles, images := other.frame(t, step.root), vscode.frame(t, step.root)
		sameScreen(t, step.name, vscode.m, other.m)
		fresh := newModelled(terminal.IdentityOther)
		fresh.frame(t, step.root)
		sameScreen(t, step.name+", against a fresh frame", vscode.m, fresh.m)
		if images > tiles {
			t.Errorf("%s: VS Code got %d images, one per tile is %d", step.name, images, tiles)
		}
		fewer = fewer || images < tiles
		if bytes.Count(vscode.out.last(), []byte("\x1bP")) != images {
			t.Errorf("%s: the model counted %d images, the stream has %d", step.name, images, bytes.Count(vscode.out.last(), []byte("\x1bP")))
		}
	}
	if !fewer {
		t.Errorf("VS Code never got fewer images than tiles")
	}
}

func TestVSCodeScrollKeepsEveryImage(t *testing.T) {
	offsets := []int{0, 1, 2, 3, 4, 5, 6, 7, 10, 9, 8, 7, 6, 5, 30, 29, 28, 27, 180, 179, 178, 177, 176, 0}
	region := regexp.MustCompile(`\x1b\[\d+;\d+r`)
	vscode := newModelled(terminal.IdentityVSCode)
	for i, root := range scrolled(t, offsets...) {
		name := "offset " + strconv.Itoa(offsets[i])
		vscode.frame(t, root)
		if i%2 == 1 {
			vscode.s.compact()
		}
		fresh := newModelled(terminal.IdentityOther)
		fresh.frame(t, root)
		sameScreen(t, name, vscode.m, fresh.m)
		step := 0
		if i > 0 {
			step = offsets[i] - offsets[i-1]
		}
		if scrolled := region.Match(vscode.out.last()); scrolled && max(step, -step) >= 25 || !scrolled && (step == 1 || step == -1) {
			t.Errorf("%s after a step of %d: region scroll %v", name, step, scrolled)
		}
	}
}

func TestScrollRepaintsWhatMovedBesideIt(t *testing.T) {
	offsets := []int{0, 1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1}
	for _, c := range []struct {
		id      terminal.Identity
		margins bool
		x       int
	}{
		{terminal.IdentityOther, true, 10},
		{terminal.IdentityOther, false, 60},
		{terminal.IdentityVSCode, false, 60},
		{terminal.IdentityVSCode, false, 10},
	} {
		s, out := screen(terminal.GraphicsSixel)
		s.Identity, s.Margins = c.id, c.margins
		m := newXterm(wt)
		for i, root := range scrolled(t, offsets...) {
			root.Children = append(root.Children, box(c.x, 6+3*i%12, 3, 4, ink(200, 60, 40, 255)))
			frame(t, s, root)
			fresh := newModelled(terminal.IdentityVSCode)
			fresh.frame(t, root)
			if !bytes.Equal(s.image().Pix, fresh.s.image().Pix) {
				t.Errorf("identity %d, margins %v, box at column %d, offset %d: the surface differs from a fresh frame", c.id, c.margins, c.x, offsets[i])
			}
			if c.id == terminal.IdentityVSCode {
				m.write(t, out.last())
				sameScreen(t, "offset "+strconv.Itoa(offsets[i]), m, fresh.m)
			}
		}
	}
}

func TestVSCodeHoverIsOneImage(t *testing.T) {
	md := newModelled(terminal.IdentityVSCode)
	md.frame(t, tree(t, demo.List(0)))
	for hover := 1; hover < 5; hover++ {
		images := md.frame(t, tree(t, demo.List(hover)))
		var sent, box image.Rectangle
		for tile, ok := range md.s.send {
			if ok {
				box, sent = box.Union(md.s.tiles[tile]), sent.Union(image.Rect(tile%len(md.s.columns), tile/len(md.s.columns), tile%len(md.s.columns)+1, tile/len(md.s.columns)+1))
			}
		}
		count := 0
		for _, ok := range md.s.send {
			if ok {
				count++
			}
		}
		if count != sent.Dx()*sent.Dy() {
			t.Fatalf("hover %d: the %d changed tiles do not fill their box %v", hover, count, sent)
		}
		if images != 1 {
			t.Errorf("hover %d: %d images for the %d changed tiles over cells %v, want one", hover, images, count, box)
		}
	}
}
