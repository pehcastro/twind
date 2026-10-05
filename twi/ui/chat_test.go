package ui

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

func counted(renders *int, app func() twi.Node) func() twi.Node {
	return func() twi.Node {
		*renders++
		return app()
	}
}

func idle(t *testing.T, d *drive.Driver, renders *int, what string) {
	t.Helper()
	*renders = 0
	d.Advance(10 * time.Second)
	if *renders != 0 {
		t.Errorf("%s: %d renders in 10 s with nothing moving, want 0", what, *renders)
	}
}

func TestMessageScrollerFollowsAndStops(t *testing.T) {
	var (
		s       *MessageScroller
		sent    = 10
		renders int
	)
	d := overlayDriver(t, 40, 14, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		s = NewMessageScroller(rt)
		return counted(&renders, func() twi.Node {
			var items []twi.NodeOption
			for i := 1; i <= sent; i++ {
				a := map[bool]Alignment{true: End, false: Start}[i%2 == 0]
				items = append(items, s.Item(Message(a, MessageContent(Bubble(Secondary, a, BubbleContent(twi.Text("message "+strconv.Itoa(i))))))))
			}
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				twi.OnKey(func(e *twi.Event) {
					if k := e.Key; !k.Release && k.Key == input.KeyRune && k.Rune == 'n' {
						sent++
						rt.Invalidate()
					}
				}),
				s.Node(twi.Class("h-8 w-30"), s.Viewport(items...), s.Button()),
			)
		})
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: following %v, sent %d:\n%s", what, s.following, sent, d.Frame().Text())
		}
	}
	shown := func() []int {
		var seen []int
		for _, line := range strings.Split(d.Frame().Text(), "\n") {
			if _, after, ok := strings.Cut(line, "message "); ok {
				n, _ := strconv.Atoi(strings.TrimRightFunc(strings.Fields(after)[0], func(r rune) bool { return r < '0' || r > '9' }))
				seen = append(seen, n)
			}
		}
		return seen
	}
	shows := func(n int) bool { return slices.Contains(shown(), n) }
	expect("it opens on the latest message, the first scrolled away", shows(10) && !shows(1) && s.following && !has(d, "↓"))
	idle(t, d, &renders, "following, settled")
	hit(d, "n")
	expect("a new message at the bottom is followed", shows(11) && s.following)
	ten := d.Frame().Text()
	d.Wheel(5, 5, -1)
	d.Advance(settleTime)
	expect("a wheel up stops following and shows the jump button", !s.following && has(d, "↓") && !shows(11))
	before, visible := d.Frame().Text(), shown()
	hit(d, "n")
	expect("a message while scrolled up leaves the view where it was", !shows(12) && slices.Equal(shown(), visible))
	idle(t, d, &renders, "scrolled up, settled")
	bx, by, _ := at(d.Frame(), "↓")
	settledClick(d, bx, by)
	expect("the jump button goes to the latest and hides", shows(12) && s.following && !has(d, "↓"))
	d.Wheel(5, 5, -2)
	d.Advance(settleTime)
	expect("two notches up stop following", !s.following)
	d.Wheel(5, 5, 5)
	d.Advance(settleTime)
	expect("scrolling back to the end follows again", s.following && !has(d, "↓"))
	hit(d, "n")
	expect("and the next message is followed", shows(13))
	hit(d, "tab pageup")
	expect("pageup on the focused viewport stops following", !s.following && !shows(13))
	hit(d, "end")
	expect("end on the viewport follows again", s.following && shows(13))
	d.Resize(40, 10)
	d.Advance(settleTime)
	expect("a shorter terminal keeps the latest message in view", shows(13))
	t.Logf("ten messages, following, 40x14:\n%s", ten)
	t.Logf("scrolled up with the jump button:\n%s", before)
}

func TestMessageScrollerShortContentFollows(t *testing.T) {
	var s *MessageScroller
	d := overlayDriver(t, 40, 12, func(rt *twi.Runtime) func() twi.Node {
		s = NewMessageScroller(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"),
				s.Node(twi.Class("h-8 w-30"), s.Viewport(s.Item(Message(Start, MessageContent(twi.Text("only one"))))), s.Button()))
		}
	})
	d.Wheel(5, 3, -3)
	d.Advance(settleTime)
	if !s.following || has(d, "↓") {
		t.Errorf("content shorter than the viewport is always at the end, following %v:\n%s", s.following, d.Frame().Text())
	}
}

func TestResizableHandleByKeysAndDrag(t *testing.T) {
	var (
		r       *Resizable
		resized [][]int
		renders int
	)
	d := overlayDriver(t, 60, 8, func(rt *twi.Runtime) func() twi.Node {
		r = NewResizable(rt)
		r.OnResize = func(s []int) { resized = append(resized, s) }
		return counted(&renders, func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
				r.Node(twi.Class("rounded-lg border"),
					r.Panel(twi.Class("items-center justify-center"), twi.Text("One")),
					r.Handle(true),
					r.Panel(twi.Class("items-center justify-center"), twi.Text("Two")),
				))
		})
	})
	handle := func() int { return r.panels[0].Bounds().Max.X }
	start := func() int { return r.panels[0].Bounds().Min.X }
	inner := func() int { return r.panels[1].Bounds().Max.X - start() }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: sizes %v, handle at %d, panels from %d over %d:\n%s", what, r.Sizes, handle(), start(), inner(), d.Frame().Text())
		}
	}
	expect("no sizes given: two even panels", slices.Equal(r.Sizes, []int{50, 50}) && len(resized) == 0)
	idle(t, d, &renders, "even panels, settled")
	expect("the handle sits after half the group", handle()-start() == inner()/2)
	hit(d, "tab right")
	expect("right on the focused handle grows the first panel by a step", r.Sizes[0] == 50+konst.PanelStep && handle()-start() == inner()*r.Sizes[0]/konst.PercentWhole)
	hit(d, "left left")
	expect("left shrinks it", r.Sizes[0] == 50-konst.PanelStep)
	hit(d, "home")
	expect("home goes to the minimum", r.Sizes[0] == konst.PanelMin && r.Sizes[1] == konst.PercentWhole-konst.PanelMin)
	hit(d, "left")
	expect("left stops at the minimum", r.Sizes[0] == konst.PanelMin)
	hit(d, "end")
	expect("end goes to the maximum", r.Sizes[0] == konst.PercentWhole-konst.PanelMin)
	hit(d, "up down")
	expect("up and down do nothing on a side-by-side group", r.Sizes[0] == konst.PercentWhole-konst.PanelMin)
	hit(d, "home")
	from, y := handle(), r.panels[0].Bounds().Min.Y+1
	d.Down(from, y)
	d.Advance(settleTime)
	for x := from + 1; x <= from+12; x++ {
		d.Move(x, y)
		d.Advance(20 * time.Millisecond)
	}
	d.Advance(settleTime)
	expect("a drag of twelve columns puts the handle under the pointer", handle() == from+12)
	d.Up(from+12, y)
	d.Advance(settleTime)
	moved := r.Sizes[0]
	settledMove(d, from+20, y)
	expect("the release ends the drag and a later move does not resize", !r.dragging && r.Sizes[0] == moved && handle() == from+12)
	d.Down(handle(), y)
	d.Move(1, y)
	d.Advance(settleTime)
	d.Up(1, y)
	d.Advance(settleTime)
	expect("a drag past the edge stops at the minimum", r.Sizes[0] == konst.PanelMin)
	if len(resized) == 0 || !slices.Equal(resized[len(resized)-1], r.Sizes) {
		t.Errorf("OnResize reported %v, want the last sizes %v", resized, r.Sizes)
	}
	hit(d, "end")
	narrow := d.Frame().Text()
	d.Resize(100, 8)
	d.Advance(settleTime)
	expect("at a wider terminal the first panel keeps its share", inner() > 90 && handle()-start() == inner()*r.Sizes[0]/konst.PercentWhole)
	t.Logf("resizable at 60 columns:\n%s", narrow)
	t.Logf("resizable at 100 columns:\n%s", d.Frame().Text())
}

func TestResizableVerticalUsesUpAndDown(t *testing.T) {
	var r *Resizable
	d := overlayDriver(t, 30, 20, func(rt *twi.Runtime) func() twi.Node {
		r = NewResizable(rt)
		r.Orientation, r.Sizes = Vertical, []int{30, 70}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
				r.Node(r.Panel(twi.Text("Header")), r.Handle(false), r.Panel(twi.Text("Body"))))
		}
	})
	hit(d, "tab left right")
	if r.Sizes[0] != 30 {
		t.Errorf("left and right moved a stacked handle: %v", r.Sizes)
	}
	hit(d, "down")
	if inner := r.panels[1].Bounds().Max.Y - r.panels[0].Bounds().Min.Y; r.Sizes[0] != 30+konst.PanelStep || r.panels[0].Bounds().Dy() != inner*r.Sizes[0]/konst.PercentWhole {
		t.Errorf("down grows the top panel: sizes %v, top %v of %d rows:\n%s", r.Sizes, r.panels[0].Bounds(), inner, d.Frame().Text())
	}
}

func TestCarouselWrapsByKeys(t *testing.T) {
	var (
		c       *Carousel
		slides  = 3
		renders int
		changes []int
	)
	d := overlayDriver(t, 40, 8, func(rt *twi.Runtime) func() twi.Node {
		c = NewCarousel(rt)
		c.OnChange = func(i int) { changes = append(changes, i) }
		return counted(&renders, func() twi.Node {
			var items []twi.NodeOption
			for i := 1; i <= slides; i++ {
				items = append(items, c.Item(Card(twi.Class("h-4 items-center justify-center"), twi.Text("Slide "+strconv.Itoa(i)))))
			}
			return twi.Element(twi.Class("flex flex-col items-center justify-center h-full bg-background text-foreground"),
				c.Node(twi.Class("w-20 h-4"), c.Content(items...), c.Previous(), c.Next()))
		})
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: index %d, changes %v:\n%s", what, c.Index, changes, d.Frame().Text())
		}
	}
	shows := func(n int) bool { return has(d, "Slide "+strconv.Itoa(n)) }
	expect("the first slide shows alone", shows(1) && !shows(2) && c.Index == 0)
	hit(d, "tab")
	expect("tab focuses the carousel", c.focused)
	settled := d.Frame().Text()
	d.Press("right")
	d.Advance(100 * time.Millisecond)
	moving := d.Frame().Text()
	expect("a frame in the slide differs from where it started and where it ends", moving != settled && c.Index == 1)
	d.Advance(settleTime)
	expect("right shows the second slide", shows(2) && !shows(1) && moving != d.Frame().Text())
	idle(t, d, &renders, "after the slide")
	hit(d, "right")
	hit(d, "right")
	expect("right on the last slide wraps to the first", c.Index == 0 && shows(1))
	hit(d, "left")
	expect("left on the first slide wraps to the last", c.Index == 2 && shows(3))
	nx, ny, _ := at(d.Frame(), "→")
	settledClick(d, nx, ny)
	expect("a click on next wraps too", c.Index == 0 && shows(1))
	px, py, _ := at(d.Frame(), "←")
	settledClick(d, px, py)
	expect("a click on previous goes back", c.Index == 2)
	hit(d, "enter")
	expect("enter on the focused previous button steps back", c.Index == 1)
	slides = 1
	c.rt.Invalidate()
	d.Advance(settleTime)
	expect("fewer slides than the index clamps to the last", c.Index == 0 && shows(1))
	if !slices.Equal(changes, []int{1, 2, 0, 2, 0, 2, 1}) {
		t.Errorf("OnChange saw %v", changes)
	}
	t.Logf("carousel 100 ms into a slide, 40x8:\n%s", moving)
}
