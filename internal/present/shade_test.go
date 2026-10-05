package present

import (
	"bytes"
	"testing"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/terminal"
)

func TestShadeZedScreen(t *testing.T) {
	root := tree(t, render.Node{Classes: []string{"flex", "flex-col", "p-4", "bg-background"}, Children: []render.Node{
		{Classes: []string{"w-24", "border", "rounded-lg", "shadow-md", "p-1", "bg-card"}, Children: []render.Node{{Text: "card"}}},
	}})
	for _, id := range []terminal.Identity{terminal.IdentityZed, terminal.IdentityOther} {
		s, out := screen(terminal.GraphicsNone)
		s.Identity = id
		frame(t, s, root)
		got := out.last()
		if shades := bytes.Contains(got, []byte("▒")) && bytes.Contains(got, []byte("░")); shades != (id == terminal.IdentityZed) {
			t.Errorf("identity %d: shades written %v, want %v", id, shades, id == terminal.IdentityZed)
		}
	}
}
