package render_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func TestGridCardHeaderDriven(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	app := func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class(sheet.Page),
				twi.Element(twi.Class(sheet.Card),
					twi.Element(twi.Class(sheet.CardHeader),
						twi.Element(twi.Class(sheet.CardTitle), twi.Text("Card Title")),
						twi.Element(twi.Class(sheet.CardDescription), twi.Text("Card Description")),
						twi.Element(twi.Class(sheet.CardAction), twi.Element(twi.Class(sheet.CardButton), twi.Text("Button"))),
					),
					twi.Element(twi.Class(sheet.CardContent), twi.Text("Content")),
				),
			)
		}
	}
	d := drive.New(app, drive.Size(60, 12), drive.With(twi.Styles(styles)))
	rows := strings.Split(d.Frame().Text(), "\n")
	t.Logf("frame 60x12:\n%s", d.Frame().Text())
	at := func(y int, word string) int {
		before, _, found := strings.Cut(rows[y], word)
		if !found {
			return -1
		}
		return utf8.RuneCountInString(before)
	}
	for _, c := range []struct {
		word string
		x, y int
	}{
		{"Card Title", 7, 2},
		{"Button", 43, 3},
		{"Card Description", 7, 5},
		{"Content", 7, 7},
	} {
		if got := at(c.y, c.word); got != c.x {
			t.Errorf("%q on row %d at column %d, want %d", c.word, c.y, got, c.x)
		}
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestGridRetainedRelayout(t *testing.T) {
	f := cssFrame(t, 40)
	node := func(first string) render.Node {
		return render.Node{Classes: strings.Fields(sheet.Cells), Children: []render.Node{
			{Classes: strings.Fields(first), Text: "a"}, {Text: "b"}, {Text: "c"},
		}}
	}
	var tree render.Tree
	for _, c := range []struct {
		first string
		want  []layout.Rect
	}{
		{"", []layout.Rect{{X: 0, Y: 0, W: 12, H: 1}, {X: 14, Y: 0, W: 12, H: 1}, {X: 28, Y: 0, W: 12, H: 1}}},
		{sheet.Wide, []layout.Rect{{X: 0, Y: 0, W: 26, H: 1}, {X: 28, Y: 0, W: 12, H: 1}, {X: 0, Y: 3, W: 12, H: 1}}},
		{"", []layout.Rect{{X: 0, Y: 0, W: 12, H: 1}, {X: 14, Y: 0, W: 12, H: 1}, {X: 28, Y: 0, W: 12, H: 1}}},
	} {
		root, err := tree.Scene(node(c.first), f)
		if err != nil {
			t.Fatal(err)
		}
		for i, child := range root.Children {
			if child.Bounds != c.want[i] {
				t.Errorf("first %q: child %d at %+v, want %+v", c.first, i, child.Bounds, c.want[i])
			}
		}
	}
}
