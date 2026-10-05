package render_test

import (
	"image"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/style"
)

func halfSheet(t *testing.T) style.Sheet {
	t.Helper()
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	each := func(first style.Property, l style.Length) []style.Declaration {
		return []style.Declaration{{Property: first, Length: l}, {Property: first + 1, Length: l}, {Property: first + 2, Length: l}, {Property: first + 3, Length: l}}
	}
	fill := style.Declaration{Property: style.PropBackground, Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 30, G: 30, B: 40, A: 255}}}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "page", Decls: []style.Declaration{fill, {Property: style.PropHeight, Length: style.Length{Unit: style.Percent, Value: 100}}}},
		{Class: "col", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Column}, {Property: style.PropWidth, Length: cells(20)}}},
		{Class: "gap-h", Decls: []style.Declaration{{Property: style.PropRowGap, Length: cells(0.5)}}},
		{Class: "gap-1", Decls: []style.Declaration{{Property: style.PropRowGap, Length: cells(1)}}},
		{Class: "pt-h", Decls: []style.Declaration{{Property: style.PropPaddingTop, Length: cells(0.5)}}},
		{Class: "pb-h", Decls: []style.Declaration{{Property: style.PropPaddingBottom, Length: cells(0.5)}}},
		{Class: "-mt-h", Decls: []style.Declaration{{Property: style.PropMarginTop, Length: cells(-0.5)}}},
		{Class: "relative", Decls: []style.Declaration{{Property: style.PropPosition, Position: style.PositionRelative}}},
		{Class: "top-h", Decls: []style.Declaration{{Property: style.PropTop, Length: cells(0.5)}}},
		{Class: "h-1", Decls: []style.Declaration{fill, {Property: style.PropHeight, Length: cells(1)}}},
		{Class: "h-3", Decls: []style.Declaration{{Property: style.PropHeight, Length: cells(3)}, {Property: style.PropOverflowY, Overflow: style.OverflowAuto}}},
		{Class: "w-3", Decls: []style.Declaration{{Property: style.PropWidth, Length: cells(3)}}},
		{Class: "square", Decls: []style.Declaration{fill, {Property: style.PropWidth, Length: cells(4)}, {Property: style.PropAspectRatio, Number: 1}}},
		{Class: "hair", Decls: append(each(style.PropBorderTopWidth, cells(0.5)), fill, style.Declaration{Property: style.PropBorderStyle, BorderStyle: style.BorderSingle})},
		{Class: "line", Decls: append(each(style.PropBorderTopWidth, cells(1)), fill, style.Declaration{Property: style.PropBorderStyle, BorderStyle: style.BorderSingle})},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func halfFrame(sheet style.Sheet, graphics bool) render.Frame {
	return render.Frame{Sheet: sheet, Width: 40, Height: layout.Length{Unit: layout.Cells, Value: 12}, Cell: image.Pt(8, 17), Graphics: graphics}
}

func el(class string, children ...render.Node) render.Node {
	return render.Node{Classes: classes(class), Children: children}
}

func at(root scene.Node, path ...int) scene.Node {
	for _, i := range path {
		root = root.Children[i]
	}
	return root
}

func TestHalfRowsPlaceTextOnWholeRows(t *testing.T) {
	sheet := halfSheet(t)
	tree := el("page", el("col gap-h", text("A"), el("hair", text("B")), text("C")))
	for _, tc := range []struct {
		graphics bool
		rows     [3]int
		box      layout.Rect
		halves   scene.Half
	}{
		{false, [3]int{0, 3, 6}, layout.Rect{X: 0, Y: 2, W: 20, H: 3}, 0},
		{true, [3]int{0, 2, 4}, layout.Rect{X: 0, Y: 1, W: 20, H: 3}, scene.HalfTop | scene.HalfBottom},
	} {
		root, err := render.Scene(tree, halfFrame(sheet, tc.graphics))
		if err != nil {
			t.Fatal(err)
		}
		got := [3]int{at(root, 0, 0).Content.Y, at(root, 0, 1, 0).Content.Y, at(root, 0, 2).Content.Y}
		if got != tc.rows {
			t.Errorf("graphics %v: A, B and C on rows %v, want %v", tc.graphics, got, tc.rows)
		}
		box := at(root, 0, 1)
		if box.Bounds != tc.box || box.Halves.Bounds != tc.halves {
			t.Errorf("graphics %v: the hairline box covers %+v with halves %b, want %+v with %b", tc.graphics, box.Bounds, box.Halves.Bounds, tc.box, tc.halves)
		}
	}
}

func TestHalfRowCellsLayoutIsTheWholeCellLayout(t *testing.T) {
	sheet := halfSheet(t)
	half, err := render.Scene(el("page", el("col gap-h", text("A"), el("hair", text("B")), text("C"))), halfFrame(sheet, false))
	if err != nil {
		t.Fatal(err)
	}
	whole, err := render.Scene(el("page", el("col gap-1", text("A"), el("line", text("B")), text("C"))), halfFrame(sheet, false))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range [][]int{{0}, {0, 0}, {0, 1}, {0, 1, 0}, {0, 2}} {
		h, w := at(half, path...), at(whole, path...)
		if h.Bounds != w.Bounds || h.Content != w.Content || h.Halves != w.Halves {
			t.Errorf("without pixels, node %v: half steps give %+v %+v %+v, whole steps %+v %+v %+v", path, h.Bounds, h.Content, h.Halves, w.Bounds, w.Content, w.Halves)
		}
	}
}

func TestHalfRowBoxesMoveTheirTextOntoRows(t *testing.T) {
	sheet := halfSheet(t)
	root, err := render.Scene(el("page", el("col pt-h", text("A"), text("B")), el("h-1"), el("relative top-h", text("C"))), halfFrame(sheet, true))
	if err != nil {
		t.Fatal(err)
	}
	if a, b := at(root, 0, 0).Content, at(root, 0, 1).Content; a.Y != 1 || b.Y != 2 || a.H != 1 || b.H != 1 {
		t.Errorf("text below a half-row padding: A %+v, B %+v, want rows 1 and 2, one row each", a, b)
	}
	if col := at(root, 0); col.Bounds.Y != 0 || col.Bounds.H != 3 || col.Halves.Bounds != scene.HalfTop {
		t.Errorf("the column moved half a row down so its text sits on rows: %+v halves %b, want rows 0 to 3, its top half a row in", col.Bounds, col.Halves.Bounds)
	}
	if line := at(root, 1); line.Bounds.Y != 3 || line.Bounds.H != 1 || line.Halves.Bounds != 0 {
		t.Errorf("an empty one-row box after the column: %+v halves %b, want row 3", line.Bounds, line.Halves.Bounds)
	}
	if shifted := at(root, 2); shifted.Bounds.Y != 4 || shifted.Bounds.H != 2 || shifted.Halves.Bounds != scene.HalfTop|scene.HalfBottom || at(root, 2, 0).Content.Y != 5 {
		t.Errorf("text shifted half a row by top-0.5: box %+v halves %b, text on row %d, want rows 4 and 5 with half edges and the text snapped down to row 5", shifted.Bounds, shifted.Halves.Bounds, at(root, 2, 0).Content.Y)
	}
}

func TestHalfRowsAboveTheViewFloorAndCeil(t *testing.T) {
	sheet := halfSheet(t)
	root, err := render.Scene(el("page", el("-mt-h h-1"), text("A")), halfFrame(sheet, true))
	if err != nil {
		t.Fatal(err)
	}
	if b := at(root, 0); b.Bounds.Y != -1 || b.Bounds.H != 2 || b.Halves.Bounds != scene.HalfTop|scene.HalfBottom {
		t.Errorf("a one-row box half a row above the view: %+v halves %b, want rows -1 and 0, both edges half a row in", b.Bounds, b.Halves.Bounds)
	}
	if a := at(root, 1); a.Content.Y != 1 {
		t.Errorf("text below it on row %d, want 1", a.Content.Y)
	}
}

func TestHalfRowsScrollWholeRows(t *testing.T) {
	sheet := halfSheet(t)
	var lines []render.Node
	for range 7 {
		lines = append(lines, text("line"))
	}
	view := el("h-3 col pb-h", lines...)
	tree := el("page", view)
	r := new(render.Tree)
	offset := func() int {
		root, err := r.Scene(tree, halfFrame(sheet, true))
		if err != nil {
			t.Fatal(err)
		}
		v := at(root, 0)
		return v.Padding.Y - v.ScrollContent.Y
	}
	offset()
	if !r.ScrollBy([]int{0}, 0, 1) {
		t.Fatal("one row down did not scroll")
	}
	if got := offset(); got != 1 {
		t.Errorf("one row down scrolled %d rows, want 1", got)
	}
	r.ScrollTo([]int{0}, 0, 99)
	if got := offset(); got != 5 {
		t.Errorf("to the end of 7.5 rows in a 3-row view: %d rows, want 5, a whole row past the half", got)
	}
	root, _ := r.Scene(tree, halfFrame(sheet, true))
	for i, line := range at(root, 0).Children {
		if want := i - 5; line.Content.Y != want {
			t.Errorf("line %d on row %d at the end, want %d", i, line.Content.Y, want)
		}
	}
}

func TestHalfRowsFollowGraphicsAfterTheFirstFrame(t *testing.T) {
	sheet := halfSheet(t)
	tree := el("page", el("col gap-h", text("A"), el("hair", text("B")), text("C")))
	r := new(render.Tree)
	if _, err := r.Scene(tree, halfFrame(sheet, false)); err != nil {
		t.Fatal(err)
	}
	flipped, err := r.Scene(tree, halfFrame(sheet, true))
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := render.Scene(tree, halfFrame(sheet, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range [][]int{{0, 0}, {0, 1}, {0, 1, 0}, {0, 2}} {
		if f, g := at(flipped, path...), at(fresh, path...); f.Bounds != g.Bounds || f.Content != g.Content || f.Halves != g.Halves {
			t.Errorf("node %v after pixels arrived: %+v %+v %+v, a fresh tree with pixels %+v %+v %+v", path, f.Bounds, f.Content, f.Halves, g.Bounds, g.Content, g.Halves)
		}
	}
}

func TestHalfRowsWrapMeasuresWholeLines(t *testing.T) {
	sheet := halfSheet(t)
	root, err := render.Scene(el("page", el("col", el("w-3", text("aaa bbb ccc")), text("D"))), halfFrame(sheet, true))
	if err != nil {
		t.Fatal(err)
	}
	if p, d := at(root, 0, 0, 0).Content, at(root, 0, 1).Content; p.Y != 0 || p.H != 3 || d.Y != 3 {
		t.Errorf("three wrapped lines %+v then D on row %d, want rows 0 to 2 and D on row 3", p, d.Y)
	}
}

func TestHalfRowsKeepShapesAndPlaces(t *testing.T) {
	sheet := halfSheet(t)
	placed := el("h-1")
	placed.At = &image.Point{X: 3, Y: 5}
	for _, graphics := range []bool{false, true} {
		root, err := render.Scene(el("page", el("square"), placed), halfFrame(sheet, graphics))
		if err != nil {
			t.Fatal(err)
		}
		if page := at(root); page.Bounds.H != 12 {
			t.Errorf("graphics %v: a full-height page is %d rows, want 12", graphics, page.Bounds.H)
		}
		if sq := at(root, 0); sq.Bounds.H != 2 || sq.Halves.Bounds != 0 {
			t.Errorf("graphics %v: a square four columns wide is %+v halves %b, want two rows of 8x17 cells", graphics, sq.Bounds, sq.Halves.Bounds)
		}
		if p := at(root, 1); p.Bounds.X != 3 || p.Bounds.Y != 5 || p.Bounds.H != 1 {
			t.Errorf("graphics %v: placed at 3,5 lands on %+v", graphics, p.Bounds)
		}
	}
}
