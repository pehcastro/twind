package runtime_test

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rkonst "github.com/pehcastro/twind/internal/konst/runtime"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/runtime/testdata/relate"
	"github.com/pehcastro/twind/twi/style"
)

type related struct {
	t        *testing.T
	d        *drive.Driver
	sky, red color.RGBA
}

func startRelated(t *testing.T) *related {
	sheet, err := relate.Styles()
	if err != nil {
		t.Fatal(err)
	}
	bg := func(class string) color.RGBA {
		return sheet.ComputeState(style.ComputedStyle{}, []string{class}, style.NodeState{}).Background.RGBA
	}
	r := &related{t: t, d: drive.New(relate.App, drive.Size(40, 8), drive.Styles(sheet)), sky: bg(relate.Lit), red: bg(relate.Pressed)}
	t.Cleanup(func() {
		if err := r.d.Err(); err != nil {
			t.Error(err)
		}
		if err := r.d.Close(); err != nil {
			t.Error(err)
		}
	})
	return r
}

func (r *related) at(word string) (int, int) {
	r.t.Helper()
	target := word
	if word == "row" {
		target = "inner"
	}
	for y, line := range strings.Split(r.d.Frame().Text(), "\n") {
		if x := strings.Index(line, target); x >= 0 {
			if word == "row" {
				return 0, y
			}
			return len([]rune(line[:x])), y
		}
	}
	r.t.Fatalf("no %q in the frame:\n%s", word, r.d.Frame().Text())
	return 0, 0
}

func (r *related) shows(step string, lit ...string) {
	r.t.Helper()
	var got []string
	for _, word := range []string{"name", "badge", "named", "item", "unnamed", "itemed", "before", "peer", "after", "row", "icon", "label", "kid", "plain", "still"} {
		x, y := r.at(word)
		if r.d.Frame().Cells().At(x, y).Bg.RGBA == r.sky {
			got = append(got, word)
		}
	}
	if !slices.Equal(got, lit) {
		r.t.Errorf("%s: lit %q, want %q\n%s", step, got, lit, r.mask())
		return
	}
	r.t.Logf("%s: lit %q\n%s", step, got, r.mask())
}

func (r *related) mask() string {
	frame := r.d.Frame()
	lines := strings.Split(frame.Text(), "\n")
	for y, line := range lines {
		runes := []rune(line)
		for x := range runes {
			if frame.Cells().At(x, y).Bg.RGBA == r.sky {
				runes[x] = '#'
			}
		}
		lines[y] = line + "  " + string(runes)
	}
	return strings.Join(lines, "\n")
}

func TestGroupHoverRedrawsTheDescendant(t *testing.T) {
	r := startRelated(t)
	r.shows("at rest")
	r.d.Move(r.at("name"))
	r.shows("over name in the group row", "badge")
	r.d.Move(r.at("badge"))
	r.shows("over the badge itself", "badge")
	r.d.Move(r.at("plain"))
	r.shows("off the group row")
}

func TestGroupHoverNamed(t *testing.T) {
	r := startRelated(t)
	r.d.Move(r.at("item"))
	r.shows("over the group/item row", "itemed")
	r.d.Move(r.at("named"))
	r.shows("over the unnamed group's group-hover/item cell", "badge")
}

func TestGroupHoverActive(t *testing.T) {
	r := startRelated(t)
	x, y := r.at("name")
	r.d.Down(x, y)
	badge, by := r.at("badge")
	if bg := r.d.Frame().Cells().At(badge, by).Bg.RGBA; bg != r.red {
		t.Errorf("pressed on name: badge %v, want group-active %v:\n%s", bg, r.red, r.mask())
	}
	r.d.Up(x, y)
	r.shows("released over name", "badge")
}

func TestPeerHoverLaterSiblingOnly(t *testing.T) {
	r := startRelated(t)
	r.d.Move(r.at("peer"))
	r.shows("over the peer", "after")
	r.d.Move(r.at("after"))
	r.shows("over the later sibling")
}

func TestRelationalHoverHasAndHands(t *testing.T) {
	r := startRelated(t)
	for _, step := range []struct{ word, lit, name string }{
		{"inner", "row", "over a child of the has-[:hover] row"},
		{"label", "icon", "over the hover:[&>svg] row"},
		{"kid", "kid", "over the *:hover child"},
	} {
		r.d.Move(r.at("plain"))
		r.shows("over plain before " + step.word)
		r.d.Move(r.at(step.word))
		r.shows(step.name, step.lit)
	}
}

func TestRelationalHoverPlainRowWritesNothing(t *testing.T) {
	sheet, err := relate.Styles()
	if err != nil {
		t.Fatal(err)
	}
	probe := startRelated(t)
	spots := map[string][2]int{}
	for _, word := range []string{"plain", "still", "name", "badge"} {
		x, y := probe.at(word)
		spots[word] = [2]int{x, y}
	}
	var views atomic.Int32
	r := launch(newBackend(40, 8), func(rt *twi.Runtime) func() twi.Node {
		view := relate.App(rt)
		return func() twi.Node {
			views.Add(1)
			return view()
		}
	}, twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	r.next(t)
	move := func(word string) {
		r.b.events <- input.MouseEvent{X: spots[word][0], Y: spots[word][1], Action: input.MouseMove, Button: input.MouseNone}
		time.Sleep(2 * rkonst.FrameInterval)
		settled := make(chan struct{})
		r.rt.Dispatch(func() { r.rt.Dispatch(func() { close(settled) }) })
		<-settled
	}
	for _, word := range []string{"plain", "still", "plain"} {
		drawn := views.Load()
		move(word)
		if n := views.Load(); n != drawn {
			t.Fatalf("a move onto %s in a row with no hover or relational rule built the view %d times, want %d", word, n, drawn)
		}
		r.quiet(t)
	}
	move("name")
	if f := r.next(t); !strings.Contains(f, "48;2;0;166;244") {
		t.Errorf("the move onto the group row wrote %q, want the sky-500 badge", f)
	}
	drawn := views.Load()
	move("badge")
	if n := views.Load(); n != drawn {
		t.Errorf("a move within the hovered group onto a cell with no hover rule built the view %d times, want %d", n, drawn)
	}
	r.quiet(t)
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}
