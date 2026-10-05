package ui

import (
	"testing"

	"github.com/pehcastro/twind/twi"
)

func shortField(t *testing.T, classes func(in *Input) string) *field {
	t.Helper()
	f := &field{}
	f.Driver = overlayDriver(t, 40, 9, func(rt *twi.Runtime) func() twi.Node {
		in := NewInput(rt)
		in.Insert("Peedro")
		f.state = func() (string, int, int) {
			s, e := in.Selection()
			return in.Value(), s, e
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), in.field(classes(in), []twi.NodeOption{twi.Class("w-16")}))
		}
	})
	return f
}

func TestClickBorderOfAFieldWithNoTextRow(t *testing.T) {
	f := shortField(t, func(in *Input) string { return "h-2 " + fieldBox + in.edge(fieldEdge) })
	x, y, ok := at(f.Frame(), "╭")
	if !ok {
		t.Fatalf("no border:\n%s", f.Frame().Text())
	}
	for _, c := range []struct {
		name       string
		x, y, want int
	}{
		{"bottom border row", x + 5, y + 1, 3},
		{"top border row", x + 4, y, 2},
		{"left border", x, y + 1, 0},
		{"far right", x + 14, y + 1, 6},
	} {
		settledClick(f.Driver, c.x, c.y)
		if _, s, e := f.state(); s != c.want || e != c.want {
			t.Errorf("%s at %d,%d: caret [%d,%d], want %d:\n%s", c.name, c.x, c.y, s, e, c.want, f.Frame().Text())
		}
	}
}
