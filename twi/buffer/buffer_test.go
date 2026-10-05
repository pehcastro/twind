package buffer

import (
	"slices"
	"testing"

	"github.com/pehcastro/twind/twi/color"
)

var red = color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 0xff, A: 0xff}}

func assertRow(t *testing.T, b *Buffer, y int, want ...string) {
	t.Helper()
	var got []string
	for _, c := range b.Row(y) {
		if c.Width == Continuation {
			got = append(got, "~")
			continue
		}
		got = append(got, c.Grapheme)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("row %d = %q, want %q", y, got, want)
	}
}

func TestWideOverwriteContinuation(t *testing.T) {
	b := New(6, 1)
	b.Set(3, 0, Cell{Grapheme: "中", Width: Wide, Bg: red})
	assertRow(t, b, 0, " ", " ", " ", "中", "~", " ")
	b.Set(4, 0, Cell{Grapheme: "a"})
	assertRow(t, b, 0, " ", " ", " ", " ", "a", " ")
	if b.At(3, 0).Bg != red {
		t.Fatalf("cleared head lost its background: %+v", b.At(3, 0))
	}
}

func TestWideOverwriteHead(t *testing.T) {
	b := New(6, 1)
	b.Set(1, 0, Cell{Grapheme: "中", Width: Wide})
	b.Set(1, 0, Cell{Grapheme: "a"})
	assertRow(t, b, 0, " ", "a", " ", " ", " ", " ")
}

func TestWideOverWide(t *testing.T) {
	b := New(6, 1)
	b.Set(0, 0, Cell{Grapheme: "中", Width: Wide})
	b.Set(2, 0, Cell{Grapheme: "日", Width: Wide})
	b.Set(1, 0, Cell{Grapheme: "本", Width: Wide})
	assertRow(t, b, 0, " ", "本", "~", " ", " ", " ")
}

func TestWideLastColumnClipped(t *testing.T) {
	b := New(4, 2)
	b.Set(3, 0, Cell{Grapheme: "中", Width: Wide, Bg: red, Attr: Bold})
	assertRow(t, b, 0, " ", " ", " ", " ")
	assertRow(t, b, 1, " ", " ", " ", " ")
	if got := b.At(3, 0); got.Width != Narrow || got.Bg != red || got.Attr != Bold {
		t.Fatalf("clipped cell = %+v, want a narrow space with the glyph's style", got)
	}
}

func TestWideOutside(t *testing.T) {
	b := New(3, 2)
	for _, p := range [][2]int{{-1, 0}, {3, 0}, {0, -1}, {0, 2}, {-1, 1}} {
		b.Set(p[0], p[1], Cell{Grapheme: "中", Width: Wide})
	}
	assertRow(t, b, 0, " ", " ", " ")
	assertRow(t, b, 1, " ", " ", " ")
}

func TestWideFill(t *testing.T) {
	b := New(5, 2)
	b.Fill(Rect{X: -1, Y: 1, W: 9, H: 4}, Cell{Grapheme: "中", Width: Wide})
	assertRow(t, b, 0, " ", " ", " ", " ", " ")
	assertRow(t, b, 1, "中", "~", "中", "~", " ")
}

func TestNarrowFillCutsWideGlyphsAtItsEdges(t *testing.T) {
	b := New(7, 2)
	for y := range 2 {
		b.Set(0, y, Cell{Grapheme: "中", Width: Wide, Bg: red})
		b.Set(4, y, Cell{Grapheme: "日", Width: Wide, Bg: red})
	}
	b.Fill(Rect{X: 1, Y: 1, W: 4, H: 1}, Cell{Grapheme: "a", Attr: Bold})
	assertRow(t, b, 0, "中", "~", " ", " ", "日", "~", " ")
	assertRow(t, b, 1, " ", "a", "a", "a", "a", " ", " ")
	for _, x := range []int{0, 5} {
		if got := b.At(x, 1); got.Width != Narrow || got.Bg != red {
			t.Fatalf("cell %d cut from its glyph = %+v, want a narrow space keeping the glyph's background", x, got)
		}
	}
	if got := b.At(2, 1); got != (Cell{Grapheme: "a", Attr: Bold}) {
		t.Fatalf("filled cell = %+v, want the fill cell", got)
	}
}

func TestFreshCellsAreSpacesOrTheFillCell(t *testing.T) {
	b := New(3, 1)
	b.Set(0, 0, Cell{Grapheme: "a", Fg: red, Attr: Bold})
	b.Resize(4, 2)
	want := [][]Cell{{{Grapheme: "a", Fg: red, Attr: Bold}, {Grapheme: " "}, {Grapheme: " "}, {Grapheme: " "}}, {{Grapheme: " "}, {Grapheme: " "}, {Grapheme: " "}, {Grapheme: " "}}}
	for y := range want {
		if got := b.Row(y); !slices.Equal(got, want[y]) {
			t.Fatalf("row %d after a resize = %+v, want %+v", y, got, want[y])
		}
	}
	for _, c := range []Cell{{}, {Grapheme: "\x00"}, {Grapheme: "x", Bg: red, Attr: Bold}} {
		f := Filled(5, 3, c)
		for y := range 3 {
			if got := f.Row(y); slices.ContainsFunc(got, func(g Cell) bool { return g != c }) {
				t.Fatalf("Filled with %+v: row %d = %+v", c, y, got)
			}
		}
	}
}

func TestWideResizeNarrower(t *testing.T) {
	b := New(5, 1)
	b.Set(0, 0, Cell{Grapheme: "a"})
	b.Set(2, 0, Cell{Grapheme: "中", Width: Wide})
	b.Resize(3, 2)
	if b.Width() != 3 || b.Height() != 2 {
		t.Fatalf("size %dx%d, want 3x2", b.Width(), b.Height())
	}
	assertRow(t, b, 0, "a", " ", " ")
	assertRow(t, b, 1, " ", " ", " ")
}

func TestDiffEqual(t *testing.T) {
	prev, cur := New(80, 24), New(80, 24)
	cur.Set(10, 5, Cell{Grapheme: "中", Width: Wide})
	prev.Set(10, 5, Cell{Grapheme: "中", Width: Wide})
	if runs := Diff(nil, prev, cur); len(runs) != 0 {
		t.Fatalf("runs = %+v, want none", runs)
	}
}

func TestDiffOneCell(t *testing.T) {
	prev, cur := New(80, 24), New(80, 24)
	cur.Set(79, 23, Cell{Grapheme: "a"})
	runs := Diff(nil, prev, cur)
	if len(runs) != 1 || runs[0] != (Run{X: 79, Y: 23, Len: 1}) {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestDiffWideGlyph(t *testing.T) {
	prev, cur := New(80, 24), New(80, 24)
	prev.Set(10, 5, Cell{Grapheme: "中", Width: Wide})
	cur.Set(10, 5, Cell{Grapheme: "日", Width: Wide})
	runs := Diff(nil, prev, cur)
	if len(runs) != 1 || runs[0] != (Run{X: 10, Y: 5, Len: 2}) {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestDiffNarrowOverWide(t *testing.T) {
	prev, cur := New(80, 24), New(80, 24)
	prev.Set(10, 5, Cell{Grapheme: "中", Width: Wide})
	cur.Set(10, 5, Cell{Grapheme: "a"})
	runs := Diff(nil, prev, cur)
	if len(runs) != 1 || runs[0] != (Run{X: 10, Y: 5, Len: 2}) {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestDiffSizeMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("no panic on buffers of different sizes")
		}
	}()
	Diff(nil, New(81, 24), New(80, 24))
}

func BenchmarkDiffUnchanged(b *testing.B) {
	prev, cur := New(200, 60), New(200, 60)
	prev.Fill(Rect{W: 200, H: 60}, Cell{Grapheme: "x", Fg: red})
	cur.Fill(Rect{W: 200, H: 60}, Cell{Grapheme: "x", Fg: red})
	var runs []Run
	b.ReportAllocs()
	for b.Loop() {
		runs = Diff(runs[:0], prev, cur)
	}
	if len(runs) != 0 {
		b.Fatalf("runs = %d", len(runs))
	}
}
