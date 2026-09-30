package render_test

import (
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/style"
)

func TestOverflowHiddenKeepsStart(t *testing.T) {
	one := style.Length{Unit: style.Cells, Value: 1}
	edges := func(props ...style.Property) []style.Declaration {
		var out []style.Declaration
		for _, p := range props {
			out = append(out, style.Declaration{Property: p, Length: one})
		}
		return out
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "w-24", Decls: []style.Declaration{{Property: style.PropWidth, Length: style.Length{Unit: style.Cells, Value: 24}}}},
		{Class: "h-4", Decls: []style.Declaration{{Property: style.PropHeight, Length: style.Length{Unit: style.Cells, Value: 4}}}},
		{Class: "overflow-hidden", Decls: []style.Declaration{
			{Property: style.PropOverflowX, Overflow: style.OverflowHidden},
			{Property: style.PropOverflowY, Overflow: style.OverflowHidden},
		}},
		{Class: "px-1", Decls: edges(style.PropPaddingLeft, style.PropPaddingRight)},
		{Class: "border", Decls: append(edges(style.PropBorderTopWidth, style.PropBorderRightWidth, style.PropBorderBottomWidth, style.PropBorderLeftWidth),
			style.Declaration{Property: style.PropBorderStyle, BorderStyle: style.BorderSingle})},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		classes string
		want    []string
	}{
		{"w-24 h-4 overflow-hidden", []string{"title", "row 1", "row 2", "row 3", "", "", "", ""}},
		{"w-24 h-4 overflow-hidden px-1 border", []string{" ▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁▁", "▕ title                ▏", "▕ row 1                ▏", " ▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔", "", "", "", ""}},
	} {
		app := func(*twi.Runtime) func() twi.Node {
			return func() twi.Node {
				return twi.Element(twi.Class(c.classes), twi.Text("title"), twi.Text("row 1"), twi.Text("row 2"), twi.Text("row 3"), twi.Text("row 4"))
			}
		}
		d := drive.New(app, drive.Size(30, 8), drive.Styles(sheet))
		if got, want := d.Frame().Text(), strings.Join(c.want, "\n")+"\n"; got != want {
			t.Errorf("%s: frame\n%s\nwant\n%s", c.classes, got, want)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}
}
