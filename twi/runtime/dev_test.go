package runtime_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pehcastro/twind/internal/dev/snapshot"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/events"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/runtime"
)

type devApp struct {
	rt       *runtime.Runtime
	b        *backend
	grid     grid
	page     string
	focused  string
	restores int
	done     chan error
}

func startDev(t *testing.T, state string) *devApp {
	t.Helper()
	a := &devApp{b: newBackend(30, 7), page: "intro", done: make(chan error, 1)}
	a.grid = blank(a.b)
	a.rt = runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None, DevState: state})
	go func() { a.done <- a.rt.Run(a.b, a.tree) }()
	return a
}

func (a *devApp) tree() runtime.Tree {
	a.rt.Remember("page", func() string { return a.page }, func(v string) {
		a.page = v
		a.restores++
	})
	rows := make([]render.Node, listRows)
	elems := make([]runtime.Node, listRows)
	for i := range rows {
		rows[i] = render.Node{Text: "row " + strconv.Itoa(i)}
		elems[i] = runtime.Node{Key: "row-" + strconv.Itoa(i), At: []int{i}, Focusable: true, Listeners: []events.Listener[*runtime.Elem]{{Type: events.Focus, Handle: func(*events.Event[*runtime.Elem]) {
			a.focused = "row-" + strconv.Itoa(i)
		}}}}
	}
	return runtime.Tree{
		Root: render.Node{Classes: []string{"row"}, Children: []render.Node{
			{Classes: []string{"col", "h-5", "w-12", "shrink-0", "overflow-y-auto"}, Children: rows},
			{Classes: []string{"col"}, Children: []render.Node{{Text: a.page}}},
		}},
		Events: runtime.Node{Children: []runtime.Node{{Key: "list", At: []int{0}, Children: elems}}},
	}
}

func (a *devApp) frame(t *testing.T) {
	t.Helper()
	a.grid.apply(t, run{b: a.b}.next(t))
}

func (a *devApp) quit(t *testing.T) {
	t.Helper()
	a.rt.Quit()
	if err := (run{done: a.done}).result(t); err != nil {
		t.Fatal(err)
	}
}

func TestDevStateSurvivesARestart(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	first := startDev(t, state)
	first.frame(t)
	for range 3 {
		first.b.events <- wheel(1, 1, input.MouseWheelDown)
		first.frame(t)
	}
	scrolled := first.grid.first()
	if scrolled == 0 {
		t.Fatalf("the wheel did not scroll the list\n%s", first.grid.text())
	}
	first.rt.Dispatch(func() {
		first.page = "input"
		first.rt.Focus("row-" + strconv.Itoa(scrolled+1))
	})
	first.frame(t)
	first.quit(t)
	saved, err := snapshot.Read(state)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Focus != "row-"+strconv.Itoa(scrolled+1) || saved.Scroll["list"].Y != scrolled || saved.Values["page"] != "input" {
		t.Fatalf("saved %+v, want focus row-%d, list at %d, page input", saved, scrolled+1, scrolled)
	}

	second := startDev(t, state)
	second.frame(t)
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Errorf("the snapshot is still there after the start: %v", err)
	}
	if second.page != "input" || second.restores != 1 {
		t.Errorf("page %q after %d restores, want input after 1", second.page, second.restores)
	}
	if got := second.grid.first(); got != scrolled {
		t.Errorf("list starts at row %d, want %d\n%s", got, scrolled, second.grid.text())
	}
	if side := second.grid.side(); side[:len("input")] != "input" {
		t.Errorf("the first frame shows %q, want the restored page", side)
	}
	if second.focused != saved.Focus {
		t.Errorf("focus %q after the restart, want %q", second.focused, saved.Focus)
	}
	second.quit(t)
}

func TestDevStateUnreadableStartsFresh(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(state, []byte(`{"version":1,"foc`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := startDev(t, state)
	a.frame(t)
	if a.page != "intro" || a.grid.first() != 0 {
		t.Errorf("page %q, list at %d; want a fresh start", a.page, a.grid.first())
	}
	a.quit(t)
}
