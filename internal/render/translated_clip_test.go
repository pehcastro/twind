package render_test

import (
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/style"
)

func TestTranslatedClip(t *testing.T) {
	cells := func(p style.Property, unit style.Unit, n float64) style.Declaration {
		return style.Declaration{Property: p, Length: style.Length{Unit: unit, Value: n}}
	}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "row", Decls: []style.Declaration{{Property: style.PropDisplay, Display: style.DisplayFlex}, {Property: style.PropDirection, Direction: style.Row}}},
		{Class: "w-5", Decls: []style.Declaration{cells(style.PropWidth, style.Cells, 5)}},
		{Class: "w-10", Decls: []style.Declaration{cells(style.PropWidth, style.Cells, 10)}},
		{Class: "shrink-0", Decls: []style.Declaration{{Property: style.PropShrink}}},
		{Class: "overflow-hidden", Decls: []style.Declaration{
			{Property: style.PropOverflowX, Overflow: style.OverflowHidden},
			{Property: style.PropOverflowY, Overflow: style.OverflowHidden},
		}},
		{Class: "-translate-x-full", Decls: []style.Declaration{cells(style.PropTranslateX, style.Percent, -100)}},
		{Class: "translate-x-5", Decls: []style.Declaration{cells(style.PropTranslateX, style.Cells, 5)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	slide := func(text string) twi.Node { return twi.Element(twi.Class("w-10 shrink-0"), twi.Text(text)) }
	for _, c := range []struct {
		name string
		root twi.Node
		want string
	}{
		{
			"a track moved by its full width inside an overflow-hidden viewport shows the next slide and nothing past the viewport",
			twi.Element(twi.Class("row w-10 overflow-hidden"), twi.Element(twi.Class("row w-10 shrink-0 -translate-x-full"), slide("one"), slide("two"), slide("three"))),
			"two",
		},
		{
			"a translated overflow-hidden box clips its children where it moved to",
			twi.Element(twi.Class("row"), twi.Element(twi.Class("row w-5 shrink-0 overflow-hidden translate-x-5"), slide("abcdefghij"))),
			"     abcde",
		},
		{
			"a translate nested in a translate inside the viewport",
			twi.Element(twi.Class("row w-10 overflow-hidden"), twi.Element(twi.Class("row w-10 shrink-0 -translate-x-full"),
				twi.Element(twi.Class("row w-10 shrink-0 -translate-x-full"), slide("one"), slide("two"), slide("three")))),
			"three",
		},
	} {
		d := drive.New(func(*twi.Runtime) func() twi.Node { return func() twi.Node { return c.root } }, drive.Size(30, 2), drive.Styles(sheet))
		if got := strings.Split(d.Frame().Text(), "\n")[0]; got != c.want {
			t.Errorf("%s: first row %q, want %q:\n%s", c.name, got, c.want, d.Frame().Text())
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}
}
