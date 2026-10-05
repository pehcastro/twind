package render_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/paint"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/style"
	twitext "github.com/pehcastro/twind/twi/text"
)

func widthSheet(t *testing.T) style.Sheet {
	t.Helper()
	one := style.Length{Unit: style.Cells, Value: 1}
	var border []style.Declaration
	for _, p := range []style.Property{style.PropBorderTopWidth, style.PropBorderRightWidth, style.PropBorderBottomWidth, style.PropBorderLeftWidth} {
		border = append(border, style.Declaration{Property: p, Length: one})
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "card", Decls: append(border,
			style.Declaration{Property: style.PropBorderStyle, BorderStyle: style.BorderSingle},
			style.Declaration{Property: style.PropWidth, Length: style.Length{Unit: style.FitContent}})},
		{Class: "w-10", Decls: []style.Declaration{{Property: style.PropWidth, Length: style.Length{Unit: style.Cells, Value: 10}}}},
		{Class: "flex", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}}},
		{Class: "shrink-0", Decls: []style.Declaration{{Property: style.PropShrink, Number: 0}}},
		{Class: "whitespace-nowrap", Decls: []style.Declaration{{Property: style.PropWhiteSpace, WhiteSpace: style.WhiteSpaceNowrap}}},
		{Class: "overflow-hidden", Decls: []style.Declaration{
			{Property: style.PropOverflowX, Overflow: style.OverflowHidden},
			{Property: style.PropOverflowY, Overflow: style.OverflowHidden},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sheet
}

func cellRows(buf *buffer.Buffer) []string {
	var out []string
	for y := range buf.Height() {
		var row []string
		for _, c := range buf.Row(y) {
			switch {
			case c.Width == buffer.Continuation:
				row = append(row, "~")
			case c.Grapheme != " ":
				row = append(row, c.Grapheme)
			}
		}
		out = append(out, strings.Join(row, " "))
	}
	return out
}

func TestWidthsCardAroundAFlag(t *testing.T) {
	root := node("", node("card", text("🇧🇷ok")))
	frame := func(w twitext.Widths) render.Frame {
		return render.Frame{Sheet: widthSheet(t), Width: 8, Height: layout.Length{Unit: layout.Cells, Value: 3}, Look: paint.Plain, Widths: w}
	}
	one := []string{"┌ ─ ─ ─ ┐", "│ 🇧🇷 o k │", "└ ─ ─ ─ ┘"}
	two := []string{"┌ ─ ─ ─ ─ ┐", "│ 🇧🇷 ~ o k │", "└ ─ ─ ─ ─ ┘"}
	for _, c := range []struct {
		name   string
		widths twitext.Widths
		want   []string
		right  int
	}{
		{"flag 1", twitext.Widths{twitext.Flag: 1}, one, 4},
		{"default", twitext.Widths{}, two, 5},
		{"flag 2", twitext.Widths{twitext.Flag: 2}, two, 5},
	} {
		buf, err := render.Render(root, frame(c.widths))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s:\n%s", c.name, strings.Join(cellRows(buf), "\n"))
		if got := cellRows(buf); !slices.Equal(got, c.want) {
			t.Errorf("%s: frame\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
		for y := range 3 {
			if g := buf.At(c.right, y).Grapheme; !strings.ContainsAny(g, "┐│┘") {
				t.Errorf("%s: row %d column %d is %q, want the right border", c.name, y, c.right, g)
			}
		}
	}
	var tree render.Tree
	for _, w := range []twitext.Widths{{}, {twitext.Flag: 1}, {}} {
		n, err := tree.Scene(root, frame(w))
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := render.Scene(root, frame(w))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := n.Children[0].Bounds, fresh.Children[0].Bounds; got != want {
			t.Errorf("retained card after switching to %v: %+v, want %+v", w, got, want)
		}
	}
}

func TestWidthsMinContent(t *testing.T) {
	root := node("flex", node("card", text("🇧🇷🇧🇷")), node("w-10 shrink-0"))
	for _, c := range []struct {
		widths twitext.Widths
		want   []string
	}{
		{twitext.Widths{twitext.Flag: 1}, []string{"┌ ─ ┐", "│ 🇧🇷 │", "│ 🇧🇷 │", "└ ─ ┘"}},
		{twitext.Widths{}, []string{"┌ ─ ─ ┐", "│ 🇧🇷 ~ │", "│ 🇧🇷 ~ │", "└ ─ ─ ┘"}},
	} {
		f := render.Frame{Sheet: widthSheet(t), Width: 11, Height: layout.Length{Unit: layout.Cells, Value: 4}, Look: paint.Plain, Widths: c.widths}
		buf, err := render.Render(root, f)
		if err != nil {
			t.Fatal(err)
		}
		got := cellRows(buf)
		t.Logf("%v:\n%s", c.widths, strings.Join(got, "\n"))
		if !slices.Equal(got, c.want) {
			t.Errorf("%v: frame\n%s\nwant\n%s", c.widths, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
	}
}

func TestNoWrapClipsOneLine(t *testing.T) {
	own := render.Node{Classes: strings.Fields("w-10 whitespace-nowrap overflow-hidden"), Text: "abc defghijklmn"}
	buf, err := render.Render(node("", own), render.Frame{Sheet: widthSheet(t), Width: 20, Height: layout.Length{Unit: layout.Cells, Value: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rowsOf(buf), []string{"abc defghi", ""}; !slices.Equal(got, want) {
		t.Errorf("frame %q, want %q", got, want)
	}
}
