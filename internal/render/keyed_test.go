package render_test

import (
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
)

func keyed(key, label string) render.Node {
	n := opening()
	n.Key, n.Children = key, []render.Node{{Text: label}}
	return n
}

func TestKeyedChildrenKeepTheirBoxes(t *testing.T) {
	half := 50 * time.Millisecond
	for _, c := range []struct {
		name  string
		after []render.Node
		label []string
		fresh []bool
	}{
		{"the first removed", []render.Node{keyed("b", "B")}, []string{"B"}, []bool{true}},
		{"the two swapped", []render.Node{keyed("b", "B"), keyed("a", "A")}, []string{"B", "A"}, []bool{true, false}},
		{"the first given a new key", []render.Node{keyed("c", "A"), keyed("b", "B")}, []string{"A", "B"}, []bool{true, true}},
	} {
		r := newMotionRun(t)
		r.at(0, screen(keyed("a", "A")))
		r.at(time.Second, screen(keyed("a", "A"), keyed("b", "B")))
		root := r.at(time.Second+half, screen(c.after...))
		if len(root.Children) < len(c.after) {
			t.Fatalf("%s: %d children, want at least %d", c.name, len(root.Children), len(c.after))
		}
		for i, label := range c.label {
			n := root.Children[i]
			moving := n.Opacity < 1
			t.Logf("%s: child %d %q opacity %.3f", c.name, i, label, n.Opacity)
			if got := n.Children[0].Lines(r.frame.Widths); len(got) == 0 || got[0] != label {
				t.Errorf("%s: child %d reads %v, want %q", c.name, i, got, label)
			}
			if moving != c.fresh[i] {
				t.Errorf("%s: child %d %q at opacity %.3f, want its enter still running %v: the box follows its key, not its index", c.name, i, label, n.Opacity, c.fresh[i])
			}
		}
	}
}
