package present

import (
	"bytes"
	"image"
	"regexp"
	"slices"
	"testing"

	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

const listRows = 200

func scrolled(t testing.TB, offsets ...int) []scene.Node {
	t.Helper()
	sheet, err := demo.Styles()
	if err != nil {
		t.Fatal(err)
	}
	f := render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}}
	var tree render.Tree
	var out []scene.Node
	for i, offset := range append([]int{0}, offsets...) {
		tree.ScrollTo(demo.ScrollerPath, 0, offset)
		root, err := tree.Scene(demo.Scroller(listRows), f)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			out = append(out, root)
		}
	}
	return out
}

func view(root scene.Node) layout.Rect { return root.Children[0].Children[1].Padding }

func TestScrollMatchesAFreshFrame(t *testing.T) {
	offsets := []int{0, 1, 2, 3, 2, 1, 0, 10, 9, 30, 29, 180, 179, 178, 0}
	trees := scrolled(t, offsets...)
	for _, margins := range []bool{true, false} {
		s, out := screen(terminal.GraphicsSixel)
		s.Margins = margins
		for i, root := range trees {
			frame(t, s, root)
			fresh, _ := screen(terminal.GraphicsSixel)
			frame(t, fresh, root)
			img := s.image()
			if !bytes.Equal(img.Pix, fresh.image().Pix) {
				t.Errorf("margins %v, offset %d: the stepped surface differs from a fresh frame", margins, offsets[i])
			}
			v := view(root)
			if above, below := img.RGBAAt(v.X*wt.X+wt.X/2, v.Y*wt.Y-wt.Y/2), img.RGBAAt(v.X*wt.X+wt.X/2, (v.Y+v.H)*wt.Y+wt.Y/2); above != below {
				t.Errorf("margins %v, offset %d: the card under the view is %v, above it %v: a row painted outside the view", margins, offsets[i], below, above)
			}
			for y := range rows {
				if !slices.Equal(onScreen(s.shown.Row(y)), onScreen(fresh.shown.Row(y))) {
					t.Errorf("margins %v, offset %d: row %d on screen is %q, a fresh frame shows %q", margins, offsets[i], y, cells(s.shown.Row(y)), cells(fresh.shown.Row(y)))
				}
			}
			for tile, sent := range s.sent {
				if !s.moved[tile] && sent != s.hash(tile) {
					t.Errorf("margins %v, offset %d: tile %v is believed sent as other pixels than the surface", margins, offsets[i], s.tiles[tile])
				}
			}
			step := 0
			if i > 0 {
				step = offsets[i] - offsets[i-1]
			}
			if scrolledByRegion := bytes.Contains(out.last(), []byte("\x1b[1S")) || bytes.Contains(out.last(), []byte("\x1b[1T")); scrolledByRegion != (step == 1 || step == -1) {
				t.Errorf("margins %v, offset %d after a step of %d: region scroll %v", margins, offsets[i], step, scrolledByRegion)
			}
		}
	}
}

func onScreen(row []buffer.Cell) []buffer.Cell {
	out := slices.Clone(row)
	for i, c := range out {
		if c == (buffer.Cell{Grapheme: " ", Bg: c.Bg, Width: buffer.Narrow}) {
			out[i] = buffer.Cell{Bg: c.Bg}
		}
	}
	return out
}

func overlaid(offset int) scene.Node {
	page := flatPage(color.RGBA{R: 255, G: 255, B: 255, A: 255}, "")
	page.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToBottom},
		From:         style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, A: 255}}},
		To:           style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{B: 255, A: 255}}, Position: 1},
	}
	view := layout.Rect{X: 4, Y: 3, W: 30, H: 12}
	list := flatPage(color.RGBA{}, "")
	list.Bounds, list.Padding, list.Content = view, view, view
	list.Scroll, list.ScrollContent = true, layout.Rect{X: view.X, Y: view.Y - offset, W: view.W, H: 60}
	for i := 0; i < 60; i += 2 {
		row := flatPage(color.RGBA{R: uint8(i * 4), G: 200, B: 90, A: 255}, "")
		row.Bounds = layout.Rect{X: view.X, Y: view.Y + i - offset, W: view.W, H: 1}
		row.Clip = view
		list.Children = append(list.Children, row)
	}
	popover := flatPage(color.RGBA{R: 40, G: 40, B: 40, A: 255}, "")
	popover.Bounds, popover.Position = layout.Rect{X: 10, Y: 6, W: 12, H: 4}, layout.PositionFixed
	page.Children = []scene.Node{list, popover}
	return page
}

func TestScrollOverStaticLayers(t *testing.T) {
	s, _ := screen(terminal.GraphicsSixel)
	s.Margins = true
	for _, offset := range []int{0, 1, 2, 3, 2, 1} {
		root := overlaid(offset)
		frame(t, s, root)
		fresh, _ := screen(terminal.GraphicsSixel)
		frame(t, fresh, root)
		if !bytes.Equal(s.image().Pix, fresh.image().Pix) {
			t.Errorf("offset %d over a gradient under a popover: the stepped surface differs from a fresh frame", offset)
		}
	}
}

func cells(row []buffer.Cell) string {
	var b []byte
	for _, c := range row {
		switch c.Grapheme {
		case "":
			b = append(b, '_')
		case "\x00":
			b = append(b, '?')
		default:
			b = append(b, c.Grapheme...)
		}
	}
	return string(b)
}

func TestScrollSendsTheExposedRowAndTheThumb(t *testing.T) {
	trees := scrolled(t, 40, 41, 40)
	s, out := screen(terminal.GraphicsSixel)
	s.Margins = true
	frame(t, s, trees[0])
	v := view(trees[0])
	for i, exposed := range []int{v.Y + v.H - 1, v.Y} {
		frame(t, s, trees[i+1])
		row := image.Rect(v.X, exposed, v.X+v.W, exposed+1)
		bar := image.Rect(v.X+v.W-1, v.Y, v.X+v.W, v.Y+v.H)
		for tile, sent := range s.send {
			if sent && !s.tiles[tile].Overlaps(row) && !s.tiles[tile].Overlaps(bar) {
				t.Errorf("scroll %d sent tile %v, outside the exposed row %v and the scrollbar %v", i, s.tiles[tile], row, bar)
			}
		}
		if text := len(out.last()) - s.imageBytes; text > 320 {
			t.Errorf("scroll %d wrote %d bytes besides images, want at most 320: one row of text, the region and the sync", i, text)
		}
		if want := "\x1b[0m\x1b[?69h\x1b[5;24r\x1b[4;39s\x1b[1" + [...]string{"S", "T"}[i] + "\x1b[?69l\x1b[r"; !bytes.Contains(out.last(), []byte(want)) {
			t.Errorf("scroll %d wrote %q, want the region scroll %q", i, out.last()[:min(len(out.last()), 80)], want)
		}
	}
}

func TestScrollWithoutRegion(t *testing.T) {
	trees := scrolled(t, 0, 25, 26)
	for _, c := range []struct {
		name     string
		graphics terminal.Graphics
		identity terminal.Identity
		from, to int
	}{
		{"a jump past the view", terminal.GraphicsSixel, terminal.IdentityOther, 0, 1},
		{"kitty", terminal.GraphicsKitty, terminal.IdentityOther, 1, 2},
		{"cells", terminal.GraphicsNone, terminal.IdentityOther, 1, 2},
	} {
		s, out := screen(c.graphics)
		s.Margins, s.Identity = true, c.identity
		frame(t, s, trees[c.from])
		frame(t, s, trees[c.to])
		if regexp.MustCompile(`\x1b\[\d+;\d+r`).Match(out.last()) {
			t.Errorf("%s: wrote a scroll region: %q", c.name, out.last())
		}
	}
}

func TestResizeClear(t *testing.T) {
	for _, c := range []struct {
		identity    terminal.Identity
		want, never string
	}{
		{terminal.IdentityOther, "\x1b[0m\x1b[2J", "\x1b[H\x1b[J"},
		{terminal.IdentityVSCode, "\x1b[0m\x1b[H\x1b[J", "\x1b[2J"},
	} {
		s, out := screen(terminal.GraphicsSixel)
		s.Identity = c.identity
		page := flatPage(color.RGBA{R: 5, G: 4, B: 6, A: 255}, "a")
		frame(t, s, page)
		if err := s.Frame(page, cols, rows-1); err != nil {
			t.Fatal(err)
		}
		if got := out.last(); !bytes.Contains(got, []byte(c.want)) || bytes.Contains(got, []byte(c.never)) {
			t.Errorf("identity %d: the frame after a resize starts %q, want %q and never %q", c.identity, got[:min(len(got), 40)], c.want, c.never)
		}
	}
}

func TestScrollbarInCells(t *testing.T) {
	trees := scrolled(t, 0, 5, 180)
	s, _ := screen(terminal.GraphicsNone)
	v := view(trees[0])
	x := v.X + v.W - 1
	for i, want := range [][]string{
		{"█", "█", " "},
		{"▄", "█", "▄!"},
		{" ", "█", "█"},
	} {
		frame(t, s, trees[i])
		top := v.Y
		if i == 2 {
			top = v.Y + v.H - 3
		}
		for k, glyph := range want {
			c := s.shown.At(x, top+k)
			got := c.Grapheme
			if c.Attr&buffer.Inverse != 0 {
				got += "!"
			}
			if got != glyph {
				t.Errorf("frame %d, row %d: scrollbar cell %q, want %q", i, top+k, got, glyph)
			}
		}
	}
}
