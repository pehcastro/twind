package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func overlayDriver(t *testing.T, width, height int, app drive.App) *drive.Driver {
	t.Helper()
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	light := zinc(t, theme.Light)
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(light)
		return app(rt)
	}, drive.Size(width, height), drive.Styles(sheet))
	d.Advance(settleTime)
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	return d
}

const (
	settleTime = time.Second
	frameStep  = 20 * time.Millisecond
)

func settledPress(d *drive.Driver, key string) {
	d.Press(key)
	d.Advance(settleTime)
}

func settledClick(d *drive.Driver, x, y int) {
	d.Click(x, y)
	d.Advance(settleTime)
}

func settledMove(d *drive.Driver, x, y int) {
	d.Move(x, y)
	d.Advance(settleTime)
}

func hit(d *drive.Driver, keys string) {
	for k := range strings.FieldsSeq(keys) {
		if typed, ok := strings.CutPrefix(k, "type:"); ok {
			d.Type(typed)
		} else {
			d.Press(k)
		}
		d.Advance(settleTime)
	}
}

func at(f drive.Frame, s string) (x, y int, ok bool) {
	for y, line := range strings.Split(f.Text(), "\n") {
		if i := strings.Index(line, s); i >= 0 {
			return len([]rune(line[:i])), y, true
		}
	}
	return 0, 0, false
}

func listen(heard *[]rune) twi.NodeOption {
	return twi.OnKey(func(k input.KeyEvent) {
		if !k.Release && k.Key == input.KeyRune {
			*heard = append(*heard, k.Rune)
		}
	})
}

func TestDialogKeys(t *testing.T) {
	var (
		before     *Toggle
		dlg        *Dialog
		name, user *Input
		heard      []rune
		changes    []bool
	)
	d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
		before, dlg, name, user = NewToggle(rt), NewDialog(rt), NewInput(rt), NewInput(rt)
		dlg.OnOpenChange = func(open bool) { changes = append(changes, open) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), listen(&heard),
				before.Node(twi.Text("before")),
				dlg.Trigger(Outline, SizeDefault, twi.Text("Edit Profile")),
				dlg.Content(
					dlg.Header(dlg.Title(twi.Text("Edit profile")), dlg.Description(twi.Text("Make changes here."))),
					name.Node(), user.Node(),
					dlg.Footer(dlg.Close(Outline, SizeDefault, twi.Text("Cancel")), dlg.Close(Default, SizeDefault, twi.Text("Save"))),
				),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	corner := func() color.RGBA { return d.Frame().Cells().At(79, 23).Bg.RGBA }
	page := corner()
	hit(d, "tab tab")
	expect("two tabs focus the trigger", dlg.focused && !before.focused)
	closed := d.Frame().Text()
	hit(d, "enter")
	expect("enter opens the dialog and moves focus to its first field", dlg.Open && name.focused && !dlg.focused && strings.Contains(d.Frame().Text(), "Edit profile"))
	expect("the backdrop dims the page to the last cell", corner() != page)
	hit(d, "type:ab")
	expect("typing goes to the field, not the page", name.Value() == "ab" && len(heard) == 0)
	cycle := []func() bool{
		func() bool { return user.focused },
		func() bool { return dlg.focus == 1 },
		func() bool { return dlg.focus == 2 },
		func() bool { return dlg.focus == 3 },
		func() bool { return name.focused },
	}
	for i, focused := range cycle {
		hit(d, "tab")
		expect("tab stays inside the dialog, in order", focused() && !before.focused && !dlg.focused)
		if i == 2 {
			expect("one Close holds the ring", dlg.focus == 2 && strings.Count(d.Frame().Text(), "Cancel") == 1)
		}
	}
	hit(d, "shift+tab")
	expect("shift+tab from the first field wraps to the X close", dlg.focus == 3)
	hit(d, "tab escape")
	expect("escape in a focused field closes the dialog and returns focus to the trigger", !dlg.Open && dlg.focused && dlg.focus == 0)
	expect("the closed dialog leaves the page as it was, trigger ring included", d.Frame().Text() == closed && corner() == page)
	hit(d, "enter tab tab enter")
	expect("enter on Cancel closes and returns focus", !dlg.Open && dlg.focused)
	hit(d, "escape")
	if want := []bool{true, false, true, false}; !slices.Equal(changes, want) {
		t.Errorf("OnOpenChange calls %v, want %v", changes, want)
	}
}

func TestDialogFooterFollowsSm(t *testing.T) {
	for _, width := range []int{70, 120} {
		d := overlayDriver(t, width, 16, func(rt *twi.Runtime) func() twi.Node {
			rt.SetTheme(zinc(t, theme.Dark))
			dlg := NewDialog(rt)
			dlg.Open = true
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
					dlg.Content(
						dlg.Header(dlg.Title(twi.Text("Edit profile")), dlg.Description(twi.Text("Make changes here."))),
						dlg.Footer(dlg.Close(Outline, SizeDefault, twi.Text("Cancel")), dlg.Close(Default, SizeDefault, twi.Text("Save"))),
					),
				)
			}
		})
		f := d.Frame()
		t.Logf("%d columns:\n%s", width, f.Text())
		cancelX, cancelY, _ := at(f, "Cancel")
		saveX, saveY, _ := at(f, "Save")
		titleX, _, _ := at(f, "Edit profile")
		descriptionX, _, _ := at(f, "Make changes here.")
		if wide := width >= 80; wide != (cancelY == saveY && cancelX < saveX) || !wide != (saveY < cancelY) {
			t.Errorf("%d columns: Cancel at %d,%d, Save at %d,%d; want a column with Save first below sm, a row with Save last from sm", width, cancelX, cancelY, saveX, saveY)
		}
		if wide := width >= 80; wide != (titleX == descriptionX) {
			t.Errorf("%d columns: title at %d, description at %d; want centred below sm, left from sm", width, titleX, descriptionX)
		}
	}
}

func TestDialogKinds(t *testing.T) {
	for _, c := range []struct {
		name  string
		make  func(*twi.Runtime) *Dialog
		where func(x, y int) bool
		x     bool
	}{
		{"dialog, 64 columns centred with an X", NewDialog, func(x, y int) bool { return x == (80-64)/2+1+3 && y > 4 && y < 16 }, true},
		{"alert dialog, centred, no X", NewAlertDialog, func(x, y int) bool { return x == (80-64)/2+1+3 && y > 4 && y < 16 }, false},
		{"sheet from the right", func(rt *twi.Runtime) *Dialog { return NewSheet(rt, Right) }, func(x, _ int) bool { return x > 30 }, true},
		{"sheet from the left", func(rt *twi.Runtime) *Dialog { return NewSheet(rt, Left) }, func(x, _ int) bool { return x < 20 }, true},
		{"sheet from the top", func(rt *twi.Runtime) *Dialog { return NewSheet(rt, Top) }, func(_, y int) bool { return y < 6 }, true},
		{"drawer from the bottom", func(rt *twi.Runtime) *Dialog { return NewDrawer(rt, Bottom) }, func(_, y int) bool { return y >= 12 }, false},
	} {
		var dlg *Dialog
		d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
			dlg = c.make(rt)
			dlg.Open = true
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
					dlg.Trigger(Ghost, SizeDefault, twi.Text("open")),
					dlg.Content(dlg.Header(dlg.Title(twi.Text("Title"))), dlg.Footer(dlg.Close(Ghost, SizeDefault, twi.Text("Done")))),
				)
			}
		})
		f := d.Frame()
		x, y, ok := at(f, "Title")
		if !ok || !c.where(x, y) {
			t.Errorf("%s: the title is at %d,%d:\n%s", c.name, x, y, f.Text())
		}
		if got := strings.Contains(f.Text(), "✕"); got != c.x {
			t.Errorf("%s: an X close shown %v, want %v", c.name, got, c.x)
		}
		hit(d, "escape")
		if dlg.Open || strings.Contains(d.Frame().Text(), "Title") {
			t.Errorf("%s: escape did not close it:\n%s", c.name, d.Frame().Text())
		}
	}
}

func TestDrawerHandleDragsItClosed(t *testing.T) {
	var dlg *Dialog
	d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
		dlg = NewDrawer(rt, Bottom)
		dlg.Open = true
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"),
				dlg.Trigger(Ghost, SizeDefault, twi.Text("open")),
				dlg.Content(dlg.Header(dlg.Title(twi.Text("Title")), dlg.Description(twi.Text("Words"))), dlg.Footer(dlg.Close(Ghost, SizeDefault, twi.Text("Done")))),
			)
		}
	})
	expect := expecter(t, d)
	muted := zinc(t, theme.Light).Tokens[theme.Muted]
	_, top, _ := at(d.Frame(), "Title")
	hx, hy := 40, top
	for hy > 0 && d.Frame().Cells().At(hx, hy).Bg != muted {
		hy--
	}
	expect("the handle sits above the title", hy > 0 && hy < top)
	title := func() int {
		_, y, _ := at(d.Frame(), "Title")
		return y
	}
	wx, wy, _ := at(d.Frame(), "Words")
	d.Down(wx, wy)
	d.Move(wx, wy+3)
	d.Advance(frameStep)
	expect("a drag that does not start on the handle moves nothing", title() == top)
	d.Up(wx, wy+3)
	d.Advance(settleTime)
	d.Down(hx, hy)
	d.Move(hx, hy+1)
	d.Advance(frameStep)
	expect("the panel follows the handle down one row while held", dlg.Open && title() == top+1)
	t.Logf("drawer dragged one row down, 80x24:\n%s", d.Frame().Text())
	d.Up(hx, hy+1)
	d.Advance(settleTime)
	expect("a release under a quarter of its height puts it back", dlg.Open && title() == top)
	d.Down(hx, hy)
	d.Move(hx, hy+2)
	d.Move(hx, hy+4)
	d.Advance(frameStep)
	expect("the panel follows the handle four rows down", dlg.Open && title() == top+4)
	d.Up(hx, hy+4)
	d.Advance(settleTime)
	expect("a release past a quarter of its height closes it", !dlg.Open && !has(d, "Title"))
}

func TestMenuKeys(t *testing.T) {
	var (
		m, empty      *DropdownMenu
		invite, share *DropdownMenuSub
		status        bool
		panel         = "Top"
		chosen        []string
		changes       []bool
		heard         []rune
	)
	d := overlayDriver(t, 80, 30, func(rt *twi.Runtime) func() twi.Node {
		m, empty = NewDropdownMenu(rt), NewDropdownMenu(rt)
		invite, share = m.Sub(), m.Sub()
		m.Align = Start
		m.OnSelect = func(s string) { chosen = append(chosen, s) }
		m.OnOpenChange = func(open bool) { changes = append(changes, open) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-row gap-4 p-1 h-full bg-background text-foreground"), listen(&heard),
				m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Open")), m.Content(
					DropdownMenuLabel(twi.Text("My Account")),
					m.Item("Profile", DropdownMenuShortcut(twi.Text("⇧⌘P"))),
					m.Item("Billing"),
					invite.Node(invite.Trigger("Invite users"), invite.Content(invite.Item("Email"), invite.Item("Message"), DropdownMenuSeparator(), invite.Item("More..."))),
					share.Node(share.Trigger("Share"), share.Content(share.Item("Link"), share.Item("QR code"))),
					DropdownMenuSeparator(),
					m.CheckboxItem("Status Bar", &status),
					m.RadioItem("Top", &panel), m.RadioItem("Bottom", &panel),
					m.Item("Log out"),
				)),
				empty.Node(empty.Trigger(Ghost, SizeDefault, twi.Text("Empty")), empty.Content()),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	line := func(s string) string {
		_, y, ok := at(d.Frame(), s)
		if !ok {
			return ""
		}
		return strings.Split(d.Frame().Text(), "\n")[y]
	}
	hit(d, "tab")
	expect("tab focuses the trigger, the menu is closed", m.focused && !strings.Contains(d.Frame().Text(), "Profile"))
	hit(d, "down")
	expect("down opens the menu on its first item", m.Open && m.active == 0 && !m.focused && strings.Contains(d.Frame().Text(), "My Account") && strings.Contains(d.Frame().Text(), "Profile"))
	hit(d, "down down")
	expect("two downs reach the sub trigger", m.active == 2)
	hit(d, "right")
	expect("right opens the sub menu on its first item", invite.Open && invite.active == 0 && strings.Contains(d.Frame().Text(), "Email"))
	ex, ey, _ := at(d.Frame(), "Email")
	ix, iy, _ := at(d.Frame(), "Invite users")
	expect("the sub menu sits right of its trigger, first item on the trigger's row", ex > ix+len("Invite users") && ey == iy)
	hit(d, "down")
	expect("down moves in the sub menu, not the root", invite.active == 1 && m.active == 2)
	hit(d, "right")
	expect("right on a plain item keeps the sub menu as it was", invite.Open && invite.active == 1)
	hit(d, "left")
	expect("left closes the sub menu only, the root keeps its item", !invite.Open && m.Open && m.active == 2 && !strings.Contains(d.Frame().Text(), "Email"))
	hit(d, "left")
	expect("left in the root menu does nothing", m.Open)
	share.Open = true
	hit(d, "right")
	expect("opening a sub menu closes its sibling", invite.Open && !share.Open)
	hit(d, "down enter")
	expect("enter in the sub menu selects, closes every level and returns focus", slices.Equal(chosen, []string{"Message"}) && !m.Open && !invite.Open && m.focused)
	hit(d, "enter")
	expect("reopening starts on the first item with no sub menu", m.Open && m.active == 0 && !invite.Open && !strings.Contains(d.Frame().Text(), "Email"))
	hit(d, "end")
	expect("end goes to the last item", m.active == 7)
	hit(d, "down")
	expect("down on the last item stays", m.active == 7)
	hit(d, "home up")
	expect("home, then up on the first item stays", m.active == 0)
	hit(d, "b")
	expect("b moves to Billing", m.active == 1)
	hit(d, "b")
	expect("b again moves to the next b, Bottom", m.active == 6)
	hit(d, "B")
	expect("typeahead ignores case and wraps", m.active == 1)
	hit(d, "tab")
	expect("tab keeps focus in the menu", m.Open && !m.focused)
	hit(d, "s s enter")
	expect("s twice reaches Status Bar and enter checks it", status && !m.Open && slices.Equal(chosen, []string{"Message", "Status Bar"}))
	hit(d, "enter")
	expect("the checked item shows its mark", strings.Contains(line("Status Bar"), "✓"))
	hit(d, "b b enter enter")
	expect("the radio item moves to Bottom", panel == "Bottom" && strings.Contains(line("Bottom"), "•") && !strings.Contains(line("Top"), "•"))
	hit(d, "down down right escape")
	expect("escape in the sub menu closes every level and returns focus", !m.Open && !invite.Open && m.focused)
	expect("items do not pile up across frames", len(m.items) == 8 && len(invite.items) == 3)
	expect("the page heard no key the menu took", len(heard) == 0)
	if want := []bool{true, false, true, false, true, false, true, false}; !slices.Equal(changes, want) {
		t.Errorf("OnOpenChange calls %v, want %v", changes, want)
	}
	hit(d, "tab enter down up home end enter x left right")
	expect("an empty menu opens and takes every key", empty.Open)
	hit(d, "escape")
	expect("and closes", !empty.Open && empty.focused)
}

func TestOverlayKeys(t *testing.T) {
	var (
		tip          *Tooltip
		pop          *Popover
		card         *HoverCard
		width        *Input
		tips, popped []bool
	)
	d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
		tip, pop, card, width = NewTooltip(rt), NewPopover(rt), NewHoverCard(rt), NewInput(rt)
		tip.OnOpenChange = func(open bool) { tips = append(tips, open) }
		pop.OnOpenChange = func(open bool) { popped = append(popped, open) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col items-center gap-6 pt-4 h-full bg-background text-foreground"),
				tip.Node(tip.Trigger(Outline, SizeDefault, twi.Text("Hover")), tip.Content(twi.Text("Add to library"))),
				pop.Node(pop.Trigger(Outline, SizeDefault, twi.Text("Open popover")), pop.Content(twi.Text("Dimensions"), width.Node())),
				card.Node(card.Trigger(Link, SizeDefault, twi.Text("@nextjs")), card.Content(twi.Text("The React Framework"))),
			)
		}
	})
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	below := func(content, trigger string) bool {
		_, cy, ok := at(d.Frame(), content)
		_, ty, _ := at(d.Frame(), trigger)
		return ok && cy > ty
	}
	hit(d, "tab")
	_, hy, _ := at(d.Frame(), "Hover")
	_, ty, shown := at(d.Frame(), "Add to library")
	expect("focus on the trigger shows the tooltip above it and keeps focus", tip.Open && tip.focused && shown && ty < hy)
	hit(d, "escape")
	expect("escape hides the tooltip, the trigger keeps focus", !tip.Open && tip.focused && !strings.Contains(d.Frame().Text(), "Add to library"))
	hit(d, "tab")
	expect("leaving the trigger keeps the tooltip hidden", !tip.Open && pop.focused)
	hit(d, "enter")
	expect("enter opens the popover below its trigger with focus in it", pop.Open && width.focused && below("Dimensions", "Open popover"))
	cx, _, _ := at(d.Frame(), "Dimensions")
	px, _, _ := at(d.Frame(), "Open popover")
	expect("the centred popover is not squeezed to its trigger's width", cx < px)
	hit(d, "type:x tab")
	expect("the popover keeps focus inside", width.focused && width.Value() == "x")
	hit(d, "escape")
	expect("escape closes the popover and returns focus", !pop.Open && pop.focused)
	hit(d, "tab")
	expect("focus shows the hover card below its trigger", card.Open && below("The React Framework", "@nextjs"))
	hit(d, "tab")
	expect("leaving the hover card's trigger hides it, the tooltip shows again", !card.Open && tip.Open)
	if !slices.Equal(tips, []bool{true, false, true}) || !slices.Equal(popped, []bool{true, false}) {
		t.Errorf("OnOpenChange: tooltip %v, popover %v", tips, popped)
	}
}

func TestOverlayPlacement(t *testing.T) {
	spot := func(side Side, align Alignment) (ax, ay, cx, cy int) {
		var pop *Popover
		d := overlayDriver(t, 100, 30, func(rt *twi.Runtime) func() twi.Node {
			pop = NewPopover(rt)
			pop.Side, pop.Align, pop.Open = side, align, true
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col items-center justify-center h-full"),
					pop.Node(pop.Trigger(Ghost, SizeXS, twi.Text("anchor")), pop.Content(twi.Text("CONTENT"))))
			}
		})
		ax, ay, _ = at(d.Frame(), "anchor")
		cx, cy, _ = at(d.Frame(), "CONTENT")
		return ax, ay, cx, cy
	}
	ax, ay, sx, sy := spot(Bottom, Start)
	_, _, mx, my := spot(Bottom, Center)
	_, _, ex, ey := spot(Bottom, End)
	if sy <= ay || my != sy || ey != sy {
		t.Errorf("bottom: content rows %d %d %d, anchor row %d", sy, my, ey, ay)
	}
	content, anchor := 36, len("anchor")+2
	if sx-mx != (content-anchor)/2 || sx-ex != content-anchor {
		t.Errorf("bottom start, centre, end at columns %d %d %d; want shifts of %d and %d", sx, mx, ex, (content-anchor)/2, content-anchor)
	}
	if _, _, _, y := spot(Top, Center); y >= ay {
		t.Errorf("top: content row %d, anchor row %d", y, ay)
	}
	if _, _, x, _ := spot(Right, Start); x <= ax+len("anchor") {
		t.Errorf("right: content column %d, anchor ends at %d", x, ax+len("anchor"))
	}
	if _, _, x, _ := spot(Left, End); x+len("CONTENT") >= ax {
		t.Errorf("left: content ends at %d, anchor starts at %d", x+len("CONTENT"), ax)
	}
}

func TestOverlayStates(t *testing.T) {
	light := zinc(t, theme.Light)
	rt := twi.New(twi.ColorProfile(color.None))
	shadowed := func(s style.ComputedStyle) bool { return len(s.Shadows) == 2 && s.Shadows[0].Blur > 0 }
	dialog := func(make func(*twi.Runtime) *Dialog, open bool) twi.Node {
		d := make(rt)
		d.Open = open
		return d.Content(d.Header(d.Title(twi.Text("T"))))
	}
	sheet := func(rt *twi.Runtime) *Dialog { return NewSheet(rt, Right) }
	menu := func(active int, sub bool) twi.Node {
		m := NewDropdownMenu(rt)
		s := m.Sub()
		m.Open, m.active, s.Open = true, active, sub
		a, b := m.Item("A"), s.Node(s.Trigger("B"), s.Content(s.Item("C")))
		return m.Node(m.Trigger(Ghost, SizeDefault, twi.Text("open")), m.Content(a, b))
	}
	trigger := NewPopover(rt).Trigger(Outline, SizeDefault, twi.Text("open"))
	closes := func() twi.Node {
		d := NewDialog(rt)
		d.Open = true
		return d.Content(d.Footer(d.Close(Outline, SizeDefault, twi.Text("Cancel")), d.Close(Default, SizeDefault, twi.Text("Save"))))
	}
	tip := func() twi.Node {
		tt := NewTooltip(rt)
		tt.Open = true
		return tt.Node(tt.Trigger(Ghost, SizeDefault, twi.Text("t")), tt.Content(twi.Text("tip")))
	}
	checkParts(t, []partCase{
		{"dialog layer: fixed inset-0 z-50, no colour of its own", light, dialog(NewDialog, true), []int{0}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionFixed && s.ZIndex == 50 && s.Inset.Top == cells(0) && s.Inset.Left == cells(0) && s.Inset.Right == cells(0) && s.Inset.Bottom == cells(0) &&
				s.Background.Kind == color.Unset
		}},
		{"dialog overlay: bg-black/50 over the whole layer, beside the panel", light, dialog(NewDialog, true), []int{0, 0}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionAbsolute && s.Inset.Top == cells(0) && s.Inset.Bottom == cells(0) && s.Background.RGBA.R == 0 && s.Background.RGBA.A >= 127 && s.Background.RGBA.A <= 128 &&
				s.Animation.Keyframes == style.KeyframesEnter && s.Animation.Enter.Opacity == 0 && s.Animation.Enter.Scale == 1
		}},
		{"dialog centring wrapper: no colour, not animated", light, dialog(NewDialog, true), []int{0, 1}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionAbsolute && s.Background.Kind == color.Unset && s.Animation.Keyframes == style.KeyframesNone
		}},
		{"dialog content: bg-background rounded-lg border shadow-lg, 64 cells at most, fades and zooms in over 100 ms", light, dialog(NewDialog, true), []int{0, 1, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Background] && s.Radius == style.RadiusLg && s.BorderWidth.Top == cells(1) && shadowed(s) && s.MaxWidth == cells(64) &&
				s.Animation.Keyframes == style.KeyframesEnter && s.Animation.Enter.Opacity == 0 && s.Animation.Enter.Scale == 0.95 && s.Animation.Duration == 100*time.Millisecond
		}},
		{"sheet from the right: full height, three quarters wide, border on the left only", light, dialog(sheet, true), []int{0, 1, 0}, func(s style.ComputedStyle) bool {
			return s.Height == percent(100) && s.Width == percent(75) && s.BorderWidth.Left == cells(1) && s.BorderWidth.Right == cells(0) && shadowed(s)
		}},
		{"menu positioner: fixed z-50, out of every clip", light, menu(0, false), []int{1}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionFixed && s.ZIndex == 50
		}},
		{"menu content: bg-popover rounded-md border shadow-md, 16 cells at least", light, menu(0, false), []int{1, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Popover] && s.Radius == style.RadiusMd && s.BorderWidth.Top == cells(1) && shadowed(s) && s.MinWidth == cells(16)
		}},
		{"highlighted item: bg-accent", light, menu(0, false), []int{1, 0, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent] && s.Color == light.Tokens[theme.AccentForeground]
		}},
		{"item not highlighted: no background", light, menu(1, false), []int{1, 0, 0, 0}, func(s style.ComputedStyle) bool { return s.Background.Kind == color.Unset }},
		{"open sub trigger: bg-accent while the root highlight is elsewhere", light, menu(0, true), []int{1, 0, 0, 1, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent]
		}},
		{"root highlight gives way to the open sub trigger", light, menu(0, true), []int{1, 0, 0, 0}, func(s style.ComputedStyle) bool { return s.Background.Kind == color.Unset }},
		{"sub content positioner: fixed z-50, out of every clip", light, menu(0, true), []int{1, 0, 0, 1, 1}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionFixed && s.ZIndex == 50
		}},
		{"sub content: bg-popover shadow-lg", light, menu(0, true), []int{1, 0, 0, 1, 1, 0, 0}, func(s style.ComputedStyle) bool {
			return shadowed(s) && s.Background == light.Tokens[theme.Popover]
		}},
		{"outline trigger idle: the button's border ring", light, trigger, nil, func(s style.ComputedStyle) bool { return ring(s, light.Tokens[theme.Border]) }},
		{"tooltip: bg-foreground text-background above the trigger", light, tip(), []int{1, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Foreground] && s.Color == light.Tokens[theme.Background]
		}},
	})
	if n := rendered(dialog(NewDialog, false)).FieldByName("Children").Len(); n != 0 {
		t.Errorf("a closed dialog's holder has %d children, want none", n)
	}
	checkFocused(t, []focusCase{
		{"outline trigger focused: the focus ring replaces the border ring on the button itself", light, trigger, []int{}, nil, func(s style.ComputedStyle) bool {
			return halo(s, light.Tokens[theme.Ring], scaled(light, theme.Ring, 0.5)) && s.Background == light.Tokens[theme.Background]
		}},
		{"focused Close: the focus ring on its button", light, closes(), []int{0, 1, 0, 0, 0}, []int{0, 1, 0, 0, 0}, func(s style.ComputedStyle) bool {
			return halo(s, light.Tokens[theme.Ring], scaled(light, theme.Ring, 0.5))
		}},
		{"the other Close keeps its own look", light, closes(), []int{0, 1, 0, 0, 0}, []int{0, 1, 0, 0, 1}, func(s style.ComputedStyle) bool {
			return len(s.Shadows) == 0 && s.Background == light.Tokens[theme.Primary]
		}},
		{"Close not focused: the outline button's border ring", light, closes(), []int{0, 1, 0, 0, 1}, []int{0, 1, 0, 0, 0}, func(s style.ComputedStyle) bool { return ring(s, light.Tokens[theme.Border]) }},
	})
}
