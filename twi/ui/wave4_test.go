package ui

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

func toggleOn(rt *twi.Runtime, r rune, on *bool) twi.NodeOption {
	return twi.OnKey(func(e *twi.Event) {
		if k := e.Key; !k.Release && k.Key == input.KeyRune && k.Rune == r {
			*on = !*on
			rt.Invalidate()
		}
	})
}

func escapes(n *int) twi.NodeOption {
	return twi.OnKeyDown(func(e *twi.Event) {
		if e.Key.Key == input.KeyEscape && !e.Key.Release {
			*n++
		}
	})
}

func TestSpinnerTurnsOnlyWhileShown(t *testing.T) {
	var (
		one, two         *Spinner
		hideOne, hideTwo bool
		renders          int
	)
	d := overlayDriver(t, 20, 3, func(rt *twi.Runtime) func() twi.Node {
		one, two = NewSpinner(rt), NewSpinner(rt)
		return func() twi.Node {
			renders++
			row := []twi.NodeOption{twi.Class("flex flex-row gap-1 h-full bg-background text-foreground"), toggleOn(rt, '1', &hideOne), toggleOn(rt, '2', &hideTwo)}
			if !hideOne {
				row = append(row, one.Node())
			}
			row = append(row, twi.Text("|"))
			if !hideTwo {
				row = append(row, two.Node())
			}
			return twi.Element(row...)
		}
	})
	glyph := func(side int) string {
		line := strings.Split(d.Frame().Text(), "\n")[0]
		parts := strings.Split(line, "|")
		return strings.TrimSpace(parts[side])
	}
	frames := utf8.RuneCountInString(spinnerFrames)
	tick := konst.SpinnerTurn / time.Duration(frames)
	seen := []string{glyph(0)}
	for range frames {
		d.Advance(tick)
		seen = append(seen, glyph(0))
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] == seen[i-1] {
			t.Errorf("tick %d: the glyph did not move, %q", i, seen)
		}
	}
	if seen[0] != seen[len(seen)-1] || len(slices.Compact(slices.Sorted(slices.Values(seen[:len(seen)-1])))) != frames {
		t.Errorf("one turn in %v shows each of the %d frames once and comes back: %q", konst.SpinnerTurn, frames, seen)
	}
	hit(d, "1")
	renders = 0
	before := glyph(1)
	d.Advance(3 * tick)
	if renders != 3 || glyph(1) == before {
		t.Errorf("hiding one spinner stops the other: %d renders in 3 ticks, glyph %q then %q", renders, before, glyph(1))
	}
	hit(d, "2")
	renders = 0
	d.Advance(10 * time.Second)
	if renders != 0 {
		t.Errorf("with both spinners hidden: %d renders in 10 s, want 0", renders)
	}
	hit(d, "1")
	before = glyph(0)
	d.Advance(tick)
	if glyph(0) == before {
		t.Errorf("a spinner shown again does not turn: %q", before)
	}
}

func TestInputGroupButtonInsideTheBorder(t *testing.T) {
	light := zinc(t, theme.Light)
	var (
		in     *Input
		copies int
	)
	d := overlayDriver(t, 40, 6, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		in = NewInput(rt)
		in.Insert("twind.dev")
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				twi.Element(twi.Class("flex flex-row w-30"), in.Group(
					InputGroupAddon(SideLeft, InputGroupText(twi.Text("https://"))),
					InputGroupAddon(SideRight, InputGroupButton(twi.OnClick(func(*twi.Event) { copies++ }), twi.Text("Copy"))),
				)),
				Button(ButtonOutline, ButtonSizeDefault, twi.Text("After")),
			)
		}
	})
	expect := expecter(t, d)
	fx, fy, _ := at(d.Frame(), "twind.dev")
	bx, by, _ := at(d.Frame(), "Copy")
	halo := func() bool { return d.Frame().At(1, fy).Fg.RGBA == light.Tokens[theme.Ring].RGBA }
	expect("the group's border is not ring coloured before any focus", !halo())
	expect("the button sits on the field's row, inside the group's 30 cells", by == fy && bx > fx && bx+len("Copy") <= 1+30)
	hit(d, "tab")
	expect("tab focuses the field", in.focused)
	hit(d, "tab")
	expect("the second tab reaches the button inside the group", !in.focused && copies == 0)
	hit(d, "enter")
	expect("enter presses the button", copies == 1)
	settledClick(d, bx, by)
	hit(d, "type:x")
	expect("a click presses it and leaves the field alone", copies == 2 && in.Value() == "twind.dev")
	expect("a key on the button inside rings the group's border", halo())
	t.Logf("input group with a button, 40x6:\n%s", d.Frame().Text())
}

func TestAspectRatioHeightFromWidth(t *testing.T) {
	wide := false
	d := overlayDriver(t, 60, 24, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		return func() twi.Node {
			width := map[bool]string{false: "w-20", true: "w-40"}[wide]
			return twi.Element(twi.Class("flex flex-row items-start gap-1 h-full bg-background text-foreground"), toggleOn(rt, 'w', &wide),
				twi.Element(twi.Class("flex flex-col "+width), AspectRatio(twi.Class("aspect-video bg-muted")), twi.Text("video")),
				twi.Element(twi.Class("flex flex-col w-8"), AspectRatio(twi.Class("bg-primary")), twi.Text("square")),
			)
		}
	})
	rows := func(label string) int {
		_, y, _ := at(d.Frame(), label)
		return y
	}
	if got := rows("video"); got != 6 {
		t.Errorf("aspect-video on 20 cells of 8x16 pixels: %d rows, want 6:\n%s", got, d.Frame().Text())
	}
	if got := rows("square"); got != 4 {
		t.Errorf("the default square on 8 cells: %d rows, want 4:\n%s", got, d.Frame().Text())
	}
	hit(d, "w")
	if got := rows("video"); got != 11 {
		t.Errorf("aspect-video after the width doubles to 40: %d rows, want 11:\n%s", got, d.Frame().Text())
	}
}

func TestComboboxKeysAndPointer(t *testing.T) {
	var (
		box      *Combobox
		changes  []string
		escaped  int
		frames   = [][2]string{{"next", "Next.js"}, {"svelte", "SvelteKit"}, {"nuxt", "Nuxt.js"}, {"remix", "Remix"}, {"astro", "Astro"}}
		filtered string
	)
	d := overlayDriver(t, 60, 16, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		box = NewCombobox(rt)
		box.Placeholder, box.Empty = "Select framework...", "No framework found."
		box.OnChange = func(v string) { changes = append(changes, v) }
		return func() twi.Node {
			var items []twi.NodeOption
			for _, f := range frames {
				items = append(items, box.Item(f[0], f[1]))
			}
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), escapes(&escaped),
				box.Node(twi.Class("w-30"), box.Input(), box.Content(items...)),
				twi.Element(twi.Text("Elsewhere")),
				Button(ButtonOutline, ButtonSizeDefault, twi.Text("After")),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: value %q, open %v, active %d, field %q, changes %v:\n%s", what, box.Value, box.Open, box.active, box.field.Value(), changes, d.Frame().Text())
		}
	}
	shows := func(labels ...string) bool {
		for _, f := range frames {
			if has(d, f[1]+" ") != slices.Contains(labels, f[1]) && has(d, f[1]) != slices.Contains(labels, f[1]) {
				return false
			}
		}
		return true
	}
	expect("closed with the placeholder", !box.Open && has(d, "Select framework...") && !has(d, "Next.js"))
	hit(d, "tab")
	expect("tab focuses the field, still closed", box.field.focused && !box.Open)
	hit(d, "type:n")
	expect("typing opens the list filtered to the items with an n", box.Open && shows("Next.js", "Nuxt.js") && !has(d, "Remix"))
	hit(d, "type:ux")
	expect("nux leaves one match, highlighted", box.Open && shows("Nuxt.js") && !has(d, "Next.js") && box.active == 0)
	filtered = d.Frame().Text()
	hit(d, "enter")
	expect("enter chooses it, closes and puts its label in the field", box.Value == "nuxt" && !box.Open && box.field.Value() == "Nuxt.js" && slices.Equal(changes, []string{"nuxt"}))
	hit(d, "down")
	expect("down opens every item with the highlight on the chosen one, marked", box.Open && shows("Next.js", "SvelteKit", "Nuxt.js", "Remix", "Astro") && box.active == 2 && has(d, "✓"))
	hit(d, "down down down down")
	expect("down stops on the last item", box.active == 4)
	hit(d, "up")
	expect("up moves back", box.active == 3)
	hit(d, "escape")
	expect("escape closes, keeps the value and stays in the combobox", !box.Open && box.Value == "nuxt" && box.field.Value() == "Nuxt.js" && escaped == 0)
	hit(d, "escape")
	expect("escape on a closed combobox reaches the page", escaped == 1)
	hit(d, "type:zzz")
	expect("a filter that matches nothing shows the empty text", box.Open && has(d, "No framework found.") && !has(d, "Remix"))
	hit(d, "enter")
	expect("enter with nothing matched chooses nothing", box.Value == "nuxt" && len(changes) == 1)
	hit(d, "escape")
	expect("escape drops the half-typed filter and restores the label", !box.Open && box.field.Value() == "Nuxt.js")
	cx, cy, _ := at(d.Frame(), "⌄")
	settledClick(d, cx, cy)
	expect("a click on the chevron opens every item", box.Open && shows("Next.js", "SvelteKit", "Nuxt.js", "Remix", "Astro") && box.field.focused)
	rx, ry, _ := at(d.Frame(), "Remix")
	settledMove(d, rx, ry)
	expect("the pointer highlights the item under it", box.active == 3)
	settledClick(d, rx, ry)
	expect("a click chooses the item", box.Value == "remix" && !box.Open && box.field.Value() == "Remix")
	settledClick(d, cx, cy)
	settledClick(d, cx, cy)
	expect("a second click on the chevron closes it", !box.Open)
	settledClick(d, cx, cy)
	ex, ey, _ := at(d.Frame(), "Elsewhere")
	settledClick(d, ex, ey)
	expect("a pointer down outside closes it with no change", !box.Open && box.Value == "remix")
	settledClick(d, cx, cy)
	hit(d, "tab")
	expect("tab closes it and moves on", !box.Open && !box.field.focused)
	expect("items do not pile up across frames", len(box.items) == len(frames))
	t.Logf("combobox filtered to one match, 60x16:\n%s", filtered)
}

func TestCalendarKeysAndPointer(t *testing.T) {
	var (
		cal      *Calendar
		selected []string
		turned   string
	)
	ymd := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
	d := overlayDriver(t, 40, 20, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		cal = NewCalendar(rt)
		cal.Month, cal.Today = ymd(2026, time.September, 1), ymd(2026, time.September, 30)
		cal.OnSelect = func(t time.Time) { selected = append(selected, t.Format(time.DateOnly)) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), cal.Node())
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: month %s, focus %s, selected %v:\n%s", what, cal.Month.Format("2006-01"), cal.focus.Format(time.DateOnly), selected, d.Frame().Text())
		}
	}
	focus := func(y int, m time.Month, day int) bool { return cal.focus.Equal(ymd(y, m, day)) }
	row := func(n int) []string {
		_, y, _ := at(d.Frame(), "Su")
		return strings.Fields(strings.Split(d.Frame().Text(), "\n")[y+2*n])
	}
	expect("September 2026 starts on a Tuesday after two outside days", has(d, "September 2026") && slices.Equal(row(1), []string{"30", "31", "1", "2", "3", "4", "5"}))
	expect("the weekdays run Sunday to Saturday", slices.Equal(row(0), []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}))
	hit(d, "tab tab tab")
	expect("the third tab reaches the grid, on today", cal.focused && focus(2026, time.September, 30))
	hit(d, "pagedown")
	expect("pagedown turns a month on the same day", focus(2026, time.October, 30) && has(d, "October 2026"))
	hit(d, "right right down")
	expect("arrows step by day and by week, and turn the month at its edge", focus(2026, time.November, 8) && has(d, "November 2026"))
	turned = d.Frame().Text()
	hit(d, "enter")
	expect("enter selects the focused day", cal.Selected.Equal(ymd(2026, time.November, 8)) && slices.Equal(selected, []string{"2026-11-08"}))
	hit(d, "space")
	expect("space on the selected day changes nothing", len(selected) == 1)
	hit(d, "pageup end")
	expect("end goes to the week's Saturday", focus(2026, time.October, 10))
	hit(d, "home")
	expect("home goes to the week's Sunday", focus(2026, time.October, 4))
	hit(d, "end down down down pagedown")
	expect("pagedown from the 31st lands on November's last day", focus(2026, time.November, 30))
	hit(d, "shift+pagedown")
	expect("shift+pagedown turns a year", focus(2027, time.November, 30) && has(d, "November 2027"))
	hit(d, "shift+pageup up")
	expect("shift+pageup comes back and up steps a week", focus(2026, time.November, 23))
	px, py, _ := at(d.Frame(), "‹")
	settledClick(d, px, py)
	expect("the previous button turns back a month", has(d, "October 2026") && focus(2026, time.October, 23))
	x, y, _ := at(d.Frame(), "15")
	settledClick(d, x, y)
	expect("a click selects the day", cal.Selected.Equal(ymd(2026, time.October, 15)) && cal.focused && slices.Equal(selected, []string{"2026-11-08", "2026-10-15"}))
	expect("October 2026 opens with four outside days", slices.Equal(row(1), []string{"27", "28", "29", "30", "1", "2", "3"}))
	x, y, _ = at(d.Frame(), "27")
	settledClick(d, x, y)
	expect("a click on an outside day selects it and turns to its month", cal.Selected.Equal(ymd(2026, time.September, 27)) && has(d, "September 2026"))
	t.Logf("calendar after pagedown and three arrows, 40x20:\n%s", turned)
}

func TestNavigationMenuKeysAndPointer(t *testing.T) {
	var (
		nav          *NavigationMenu
		started, kit *NavigationMenuItem
		chosen       []string
		escaped      int
		panel        string
	)
	d := overlayDriver(t, 80, 16, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		nav = NewNavigationMenu(rt)
		started, kit = nav.Item("started"), nav.Item("components")
		nav.OnSelect = func(href string) { chosen = append(chosen, href) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), escapes(&escaped),
				nav.Node(nav.List(
					started.Node(started.Trigger(twi.Text("Getting started")), started.Content(twi.Class("w-30"),
						started.Link("/docs", twi.Text("Introduction")), started.Link("/docs/installation", twi.Text("Installation")), started.Link("/docs/typography", twi.Text("Typography")))),
					kit.Node(kit.Trigger(twi.Text("Components")), kit.Content(twi.Class("w-30"),
						kit.Link("/docs/alert-dialog", twi.Text("Alert Dialog")), kit.Link("/docs/hover-card", twi.Text("Hover Card")), kit.Link("/docs/progress", twi.Text("Progress")))),
				)),
				twi.Element(twi.Class("pt-10"), twi.Text("Elsewhere")),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: value %q, link %d, chosen %v, escaped %d:\n%s", what, nav.Value, nav.link, chosen, escaped, d.Frame().Text())
		}
	}
	expect("closed at first", nav.Value == "" && !has(d, "Introduction"))
	kx, ky, _ := at(d.Frame(), "Components")
	d.Move(kx, ky)
	d.Advance(konst.HoverOpen / 2)
	expect("hover waits before opening", nav.Value == "")
	d.Advance(settleTime)
	expect("hover on a trigger opens its panel", nav.Value == "components" && has(d, "Hover Card"))
	gx, gy, _ := at(d.Frame(), "Getting started")
	d.Move(gx, gy)
	d.Advance(50 * time.Millisecond)
	expect("the next trigger opens at once while the last panel fades out", nav.Value == "started" && has(d, "Introduction") && has(d, "Progress"))
	d.Advance(settleTime)
	expect("the old panel is gone after its exit", !has(d, "Progress"))
	ix, iy, _ := at(d.Frame(), "Installation")
	for y := gy + 1; y <= iy; y++ {
		d.Move(ix, y)
		d.Advance(20 * time.Millisecond)
	}
	d.Advance(settleTime)
	expect("the pointer crosses into the panel and highlights a link", nav.Value == "started" && nav.link == 1)
	panel = d.Frame().Text()
	settledClick(d, ix, iy)
	expect("a click on a link selects it and closes", nav.Value == "" && slices.Equal(chosen, []string{"/docs/installation"}))
	settledMove(d, gx, gy)
	expect("hover opens it again", nav.Value == "started")
	ex, ey, _ := at(d.Frame(), "Elsewhere")
	d.Move(ex, ey)
	d.Advance(konst.HoverShut / 2)
	expect("leaving the menu waits a moment", nav.Value == "started")
	d.Advance(settleTime)
	expect("then closes it", nav.Value == "" && !has(d, "Introduction"))
	hit(d, "tab right enter")
	expect("tab, right and enter open the second panel", nav.Value == "components" && nav.link == -1)
	hit(d, "down down enter")
	expect("down walks into the panel and enter selects the link", nav.Value == "" && slices.Equal(chosen, []string{"/docs/installation", "/docs/hover-card"}))
	hit(d, "down left")
	expect("down opens, left carries the open panel to the previous trigger", nav.Value == "started" && has(d, "Introduction"))
	hit(d, "down up")
	expect("up from the first link goes back to the trigger", nav.link == -1 && nav.Value == "started")
	hit(d, "escape")
	expect("escape closes it and stays in the menu", nav.Value == "" && escaped == 0 && nav.focused)
	hit(d, "escape")
	expect("escape on a closed menu reaches the page", escaped == 1)
	hit(d, "enter")
	settledClick(d, ex, ey)
	expect("a pointer down outside closes it", nav.Value == "")
	t.Logf("navigation menu with a panel open, 80x16:\n%s", panel)
}
