package runtime_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
)

func TestPageKeysScrollThePageFromOutsideIt(t *testing.T) {
	b := newBackend(20, 6)
	rows := make([]render.Node, listRows)
	for i := range rows {
		rows[i] = render.Node{Text: "row " + strconv.Itoa(i)}
	}
	tree := runtime.Tree{
		Root: render.Node{Classes: []string{"col"}, Children: []render.Node{
			{Text: "nav link"},
			{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto"}, Children: rows},
		}},
		Events: runtime.Node{Children: []runtime.Node{{At: []int{0}, Focusable: true}}},
	}
	a := &scrollApp{b: b, grid: blank(b)}
	a.rt = runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None})
	r := run{b: b, done: make(chan error, 1)}
	go func() { r.done <- a.rt.Run(b, func() runtime.Tree { return tree }) }()
	t.Cleanup(func() {
		a.rt.Quit()
		if err := r.result(t); err != nil {
			t.Error(err)
		}
	})
	a.frame(t)
	top := func() string { return strings.TrimSpace(string(a.grid[1][:listWidth-1])) }
	a.send(t, press(input.KeyEnd))
	if got, want := top(), "row "+strconv.Itoa(listRows-listView); got != want {
		t.Fatalf("End with nothing focused: the page starts at %q, want %q:\n%s", got, want, a.grid.text())
	}
	b.events <- press(input.KeyTab)
	a.grid.settle(t, b)
	for _, step := range []struct {
		key  input.Key
		want int
	}{{input.KeyHome, 0}, {input.KeyPageDown, listView}, {input.KeyEnd, listRows - listView}, {input.KeyPageUp, listRows - 2*listView}} {
		a.send(t, press(step.key))
		if got, want := top(), "row "+strconv.Itoa(step.want); got != want {
			t.Fatalf("key %d on the focused nav link: the page starts at %q, want %q:\n%s", step.key, got, want, a.grid.text())
		}
	}
	a.nothing(t, "an arrow on the nav link", press(input.KeyArrowDown))
}
