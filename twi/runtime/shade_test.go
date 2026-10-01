package runtime_test

import (
	"cmp"
	"image"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/present/demo"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/terminal"
)

func TestShadeZedFrameFromTheBackend(t *testing.T) {
	sheet, err := demo.Styles()
	if err != nil {
		t.Fatal(err)
	}
	b := newBackend(pageCols, pageRows)
	b.caps.Identity = terminal.IdentityZed
	rt := runtime.New(runtime.Config{Clock: &clock{}, Sheet: sheet, Profile: color.TrueColor})
	tree := runtime.Tree{Root: render.Node{Classes: []string{"flex", "flex-col", "p-2", "bg-background", "text-foreground"}, Children: []render.Node{
		{Classes: []string{"w-24", "border", "rounded-lg", "shadow-md", "p-1", "bg-card"}, Children: []render.Node{{Text: "A card with shadow-md"}}},
	}}}
	r := run{b: b, done: make(chan error, 1)}
	go func() { r.done <- rt.Run(b, func() runtime.Tree { return tree }) }()
	var m screenModel
	m.write(t, []byte(r.next(t)), image.Point{})
	lines := make([]string, 0, 11)
	for y := range 11 {
		var line strings.Builder
		for x := range 30 {
			line.WriteString(cmp.Or(m.cells[y][x].text, " "))
		}
		lines = append(lines, line.String())
	}
	frame := strings.Join(lines, "\n")
	t.Logf("zed frame:\n%s", frame)
	if !strings.Contains(frame, "▒") || !strings.Contains(frame, "░") {
		t.Errorf("frame has no ▒ and ░ next to the card")
	}
	rt.Quit()
	if err := r.result(t); err != nil {
		t.Fatal(err)
	}
}
