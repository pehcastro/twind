package present

import (
	"image"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/terminal"
)

func TestTextOnAThinLineThroughItsRowStaysVisible(t *testing.T) {
	const line, label = 4, "186"
	gridded := func(dst *image.RGBA, cell image.Point) {
		for x := range dst.Rect.Dx() {
			copy(dst.Pix[dst.PixOffset(x, (line-2)*cell.Y+cell.Y/2):], []uint8{60, 60, 70, 255})
		}
	}
	blanks := make([]render.Node, 6)
	for i := range blanks {
		blanks[i] = render.Node{Text: strings.Repeat(" ", 30)}
	}
	plot := render.Node{Extra: &render.Extra{Canvas: &render.Canvas{Key: 1, Paint: gridded}}, Children: blanks}
	root := render.Node{Classes: strings.Fields("flex flex-col gap-1 p-2 bg-background text-foreground"), Children: []render.Node{
		plot,
		{Extra: &render.Extra{Placement: render.Placement{Positioned: true, At: image.Pt(6, line)}}, Children: []render.Node{{Text: label}}},
	}}
	sheet, err := demo.Styles()
	if err != nil {
		t.Fatal(err)
	}
	scene, err := render.Scene(root, render.Frame{Sheet: sheet, Width: cols, Height: layout.Length{Unit: layout.Cells, Value: rows}, Cell: wt, Graphics: true})
	if err != nil {
		t.Fatal(err)
	}
	s, out := screen(terminal.GraphicsSixel)
	m := &term{}
	frame(t, s, scene)
	m.write(t, out.last())
	at := -1
	for x := range s.cols - len(label) {
		if s.text.At(x, line).Grapheme == label[:1] && s.text.At(x+1, line).Grapheme == label[1:2] {
			at = x
		}
	}
	if at < 0 {
		t.Fatalf("no %q on row %d, the row of the line", label, line)
	}
	inset := s.inset()
	for x := at; x < at+len(label); x++ {
		covered := 0
		for py := line*wt.Y + inset; py < (line+1)*wt.Y-inset; py++ {
			for px := x * wt.X; px < (x+1)*wt.X; px++ {
				if m.layer.RGBAAt(px, py).A != 0 {
					covered++
				}
			}
		}
		if covered > 0 {
			t.Errorf("cell %d,%d holds %q and an image covers %d of its glyph pixels: the text is hidden", x, line, s.text.At(x, line).Grapheme, covered)
		}
	}
}
