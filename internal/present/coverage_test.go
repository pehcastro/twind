package present

import (
	"bytes"
	"image"
	"testing"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/terminal"
)

func TestCoverageFontChangeRepaints(t *testing.T) {
	lacked := ""
	s, out := screen(terminal.GraphicsNone)
	s.Font, s.Covers = terminal.Font{Face: "Cascadia Mono", Size: image.Pt(8, 16)}, func(cluster string) bool { return cluster != lacked }
	root := tree(t, render.Node{Text: "moon ☾"})
	frame(t, s, root)
	if !bytes.Contains(out.last(), []byte("☾")) {
		t.Fatalf("first frame lacks the moon: %q", out.last())
	}
	lacked = "☾"
	s.Font = terminal.Font{Face: "Lucida Console", Size: image.Pt(8, 16)}
	frame(t, s, root)
	if got := out.last(); bytes.Contains(got, []byte("☾")) || !bytes.Contains(got, []byte("●")) {
		t.Errorf("after the font change: %q, want the moon's stand-in", got)
	}
}
