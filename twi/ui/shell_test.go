package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func expecter(t *testing.T, d *drive.Driver) func(string, bool) {
	return func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
}

func has(d *drive.Driver, s string) bool { return strings.Contains(d.Frame().Text(), s) }

func TestContextMenuKeysAndPointer(t *testing.T) {
	var (
		menu    *ContextMenu
		tools   *DropdownMenuSub
		chosen  []string
		heard   []rune
		marked  = true
		changes []bool
	)
	d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
		menu = NewContextMenu(rt)
		tools = menu.Sub()
		menu.OnSelect = func(s string) { chosen = append(chosen, s) }
		menu.OnOpenChange = func(open bool) { changes = append(changes, open) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), listen(&heard),
				Button(Outline, SizeDefault, twi.Text("before")),
				twi.Element(twi.Class("flex flex-row pl-10"), menu.Node(twi.Class("w-40 h-7"),
					menu.Trigger(twi.Class("flex-1 items-center justify-center rounded-md border border-dashed"), twi.Text("Right click here")),
					menu.Content(twi.Class("w-26"),
						menu.Item("Back"), menu.Item("Reload"),
						tools.Node(tools.Trigger("More Tools"), tools.Content(tools.Item("Save Page..."), tools.Item("Developer Tools"))),
						menu.CheckboxItem("Show Bookmarks", &marked),
					),
				)),
			)
		}
	})
	expect := expecter(t, d)
	left, top, _ := at(d.Frame(), "╭")
	expect("closed at first", !has(d, "Back") && has(d, "Right click here"))
	hit(d, "tab tab")
	expect("the second tab focuses the area", menu.focused)
	hit(d, "f10 shift+f9 enter space")
	expect("only shift+f10 opens it: f10, shift+f9, enter and space do not", !menu.Open && !has(d, "Back"))
	heard = nil
	hit(d, "shift+f10")
	x, y, _ := at(d.Frame(), "Back")
	expect("shift+f10 opens it on its first item with focus inside", menu.Open && menu.active == 0 && !menu.focused)
	expect("the menu hangs from the area's top left corner", y == top+1 && x == left+4)
	t.Logf("context menu opened by shift+f10 at the area's corner, 80x24:\n%s", d.Frame().Text())
	hit(d, "down down right")
	expect("right on More Tools opens its sub menu", tools.Open && has(d, "Save Page..."))
	hit(d, "left")
	expect("left closes only the sub menu", !tools.Open && menu.Open && !has(d, "Save Page..."))
	hit(d, "escape")
	expect("escape closes it and gives focus back to the area", !menu.Open && menu.focused && !has(d, "Back"))
	hit(d, "shift+f10 down enter")
	expect("enter chooses Reload, closes and gives focus back", slices.Equal(chosen, []string{"Reload"}) && !menu.Open && menu.focused)
	hit(d, "shift+f10 up")
	expect("up on the first item stays", menu.active == 0)
	hit(d, "s enter")
	expect("typeahead reaches Show Bookmarks and enter unchecks it", !marked && slices.Equal(chosen, []string{"Reload", "Show Bookmarks"}))

	bx, by, _ := at(d.Frame(), "Right click here")
	settledClick(d, bx, by)
	expect("a left click on the area does not open it", !menu.Open && !has(d, "Back"))
	hit(d, "shift+f10")
	x, y, _ = at(d.Frame(), "Reload")
	settledClick(d, x, y)
	expect("a click on an item chooses it", slices.Equal(chosen, []string{"Reload", "Show Bookmarks", "Reload"}) && !menu.Open)
	hit(d, "shift+f10")
	x, y, _ = at(d.Frame(), "before")
	settledClick(d, x, y)
	expect("a pointer down outside closes it", !menu.Open && !has(d, "Back"))
	expect("the page heard no key the menu took", len(heard) == 0)
	if want := []bool{true, false, true, false, true, false, true, false, true, false}; !slices.Equal(changes, want) {
		t.Errorf("OnOpenChange calls %v, want %v", changes, want)
	}
}

func TestContextMenuRightClick(t *testing.T) {
	var (
		menu   *ContextMenu
		chosen []string
	)
	d := overlayDriver(t, 80, 24, func(rt *twi.Runtime) func() twi.Node {
		menu = NewContextMenu(rt)
		menu.OnSelect = func(s string) { chosen = append(chosen, s) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				twi.Text("outside"),
				twi.Element(twi.Class("flex flex-row h-6 shrink-0 overflow-hidden"), menu.Node(twi.Class("w-40"),
					menu.Trigger(twi.Class("flex-1 items-center justify-center rounded-md border border-dashed"), twi.Text("Right click here")),
					menu.Content(twi.Class("w-20"), menu.Item("Back"), menu.Item("Forward"), menu.Item("Reload"), menu.Item("Save Page..."), menu.Item("Print...")),
				)),
				twi.Element(twi.Class("flex h-6 shrink-0 bg-secondary"), twi.Text("a later sibling")),
			)
		}
	})
	expect := expecter(t, d)
	right := func(x, y int) {
		d.ClickWith(input.MouseRight, x, y)
		d.Advance(settleTime)
	}
	settledClick(d, 20, 6)
	expect("a left click on the area does not open it", !menu.Open && !has(d, "Back"))
	right(20, 6)
	x, y, _ := at(d.Frame(), "Back")
	expect("a right click opens it with its corner at the pointer", menu.Open && x == 20+4 && y == 6+1)
	t.Logf("right click at 20,6, 80x24:\n%s", d.Frame().Text())
	_, py, _ := at(d.Frame(), "Print...")
	expect("the menu is whole past the bottom of its overflow-hidden parent and over the later sibling", py == 6+5 && has(d, "Print..."))
	right(30, 3)
	x, y, _ = at(d.Frame(), "Back")
	expect("a second right click inside the area moves it to the new point", menu.Open && x == 30+4 && y == 3+1)
	hit(d, "down enter")
	expect("keys work in the menu a right click opened", slices.Equal(chosen, []string{"Forward"}) && !menu.Open)
	right(12, 4)
	x, y, _ = at(d.Frame(), "Reload")
	settledClick(d, x, y)
	expect("a click chooses an item", slices.Equal(chosen, []string{"Forward", "Reload"}) && !menu.Open)
	right(12, 4)
	right(2, 1)
	expect("a right click outside the area closes it", !menu.Open && !has(d, "Back"))
	hit(d, "tab shift+f10")
	x, y, _ = at(d.Frame(), "Back")
	lx, ly, _ := at(d.Frame(), "╭")
	expect("shift+f10 after a right click opens at the area's corner again", menu.Open && x == lx+4 && y == ly+1)
}

func TestMenubarKeysAndPointer(t *testing.T) {
	var (
		bar              *Menubar
		file, edit, view *MenubarMenu
		share            *DropdownMenuSub
		chosen           []string
		heard            []rune
		before, after    *Toggle
	)
	d := overlayDriver(t, 100, 24, func(rt *twi.Runtime) func() twi.Node {
		bar, before, after = NewMenubar(rt), NewToggle(rt), NewToggle(rt)
		file, edit, view = bar.Menu(), bar.Menu(), bar.Menu()
		share = file.Sub()
		for _, m := range []*MenubarMenu{file, edit, view} {
			m.OnSelect = func(s string) { chosen = append(chosen, s) }
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"), listen(&heard),
				before.Node(twi.Text("before")),
				bar.Node(twi.Class("self-start"),
					file.Node(file.Trigger(twi.Text("File")), file.Content(
						file.Item("New Tab"),
						share.Node(share.Trigger("Share"), share.Content(share.Item("Email link"), share.Item("Notes"))),
						file.Item("Print..."),
					)),
					edit.Node(edit.Trigger(twi.Text("Edit")), edit.Content(edit.Item("Undo"), edit.Item("Redo"))),
					view.Node(view.Trigger(twi.Text("View")), view.Content(view.Item("Reload"))),
				),
				after.Node(twi.Text("after")),
			)
		}
	})
	expect := expecter(t, d)
	accent := zinc(t, theme.Light).Tokens[theme.Accent].RGBA
	lit := func(s string) bool {
		x, y, ok := at(d.Frame(), s)
		return ok && d.Frame().Cells().At(x, y).Bg.RGBA == accent
	}
	opened := func() []string {
		var out []string
		for i, m := range []*MenubarMenu{file, edit, view} {
			if m.Open {
				out = append(out, []string{"File", "Edit", "View"}[i])
			}
		}
		return out
	}

	hit(d, "tab tab")
	expect("the second tab lands on File, lit, nothing open", bar.within && bar.active == 0 && lit("File") && !lit("Edit") && len(opened()) == 0)
	hit(d, "tab")
	expect("one tab leaves the bar", after.focused && !bar.within && !lit("File"))
	hit(d, "shift+tab")
	expect("shift+tab from after lands on the last trigger", bar.within && bar.active == 2 && lit("View"))
	hit(d, "shift+tab")
	expect("one more shift+tab leaves the bar", before.focused && !bar.within)
	hit(d, "tab")
	expect("tab from before enters on File", bar.within && bar.active == 0)
	hit(d, "left")
	expect("left from File wraps to View", bar.active == 2 && lit("View") && !lit("File"))
	hit(d, "right right")
	expect("right twice reaches Edit", bar.active == 1 && lit("Edit"))
	hit(d, "down")
	expect("down opens Edit on its first item", slices.Equal(opened(), []string{"Edit"}) && edit.active == 0 && has(d, "Undo"))
	hit(d, "right")
	expect("right moves the open menu to View", slices.Equal(opened(), []string{"View"}) && bar.active == 2 && has(d, "Reload") && !has(d, "Undo"))
	hit(d, "right")
	expect("right from the last menu wraps to File", slices.Equal(opened(), []string{"File"}) && has(d, "New Tab"))
	hit(d, "down right")
	expect("right on Share opens its sub menu, File stays open", share.Open && file.Open && has(d, "Email link"))
	t.Logf("File open, Share's sub menu open, 100x24:\n%s", d.Frame().Text())
	hit(d, "left")
	expect("left closes the sub menu only", !share.Open && file.Open && file.active == 1)
	hit(d, "left")
	expect("left in a menu moves to the previous menu", slices.Equal(opened(), []string{"View"}))
	hit(d, "escape")
	expect("escape closes it and View keeps the highlight", len(opened()) == 0 && bar.active == 2 && lit("View") && !has(d, "Reload"))
	hit(d, "up")
	expect("up opens View on its last item", view.Open && view.active == 0)
	hit(d, "escape left left up")
	expect("up opens File on its last item", file.Open && file.active == 2)
	hit(d, "enter")
	expect("enter chooses Print... and closes", slices.Equal(chosen, []string{"Print..."}) && len(opened()) == 0)
	hit(d, "enter tab")
	expect("tab closes the menu and leaves the bar", len(opened()) == 0 && after.focused)
	expect("the page heard no key the bar took", len(heard) == 0)

	x, y, _ := at(d.Frame(), "File")
	settledClick(d, x, y)
	expect("a click on File opens it", slices.Equal(opened(), []string{"File"}))
	x, y, _ = at(d.Frame(), "Edit")
	d.Move(x, y)
	d.Advance(settleTime)
	expect("hovering Edit while File is open switches", slices.Equal(opened(), []string{"Edit"}) && bar.active == 1)
	x, y, _ = at(d.Frame(), "View")
	d.Move(x, y+3)
	d.Advance(settleTime)
	expect("leaving the bar keeps Edit open", slices.Equal(opened(), []string{"Edit"}))
	x, y, _ = at(d.Frame(), "Edit")
	settledClick(d, x, y)
	expect("a click on the open trigger closes it", len(opened()) == 0)
	x, y, _ = at(d.Frame(), "View")
	d.Move(x, y)
	d.Advance(settleTime)
	expect("hovering with nothing open opens nothing", len(opened()) == 0)
	settledClick(d, x, y)
	x, y, _ = at(d.Frame(), "Reload")
	settledClick(d, x, y)
	expect("a click on an item chooses it", slices.Equal(chosen, []string{"Print...", "Reload"}) && len(opened()) == 0)
	x, y, _ = at(d.Frame(), "File")
	settledClick(d, x, y)
	x, y, _ = at(d.Frame(), "before")
	settledClick(d, x, y)
	expect("a click outside closes it", len(opened()) == 0)
}

func TestSidebarCollapse(t *testing.T) {
	for _, c := range []struct{ width, expanded int }{{150, 30}, {120, 26}} {
		var (
			side    *Sidebar
			group   *Collapsible
			field   *Input
			changes []bool
		)
		d := overlayDriver(t, c.width, 20, func(rt *twi.Runtime) func() twi.Node {
			side, group, field = NewSidebar(rt), NewCollapsible(rt), NewInput(rt)
			group.Open = true
			side.OnOpenChange = func(open bool) { changes = append(changes, open) }
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"), side.Provider(
					side.Node(
						SidebarHeader(SidebarMenu(SidebarMenuItem(SidebarMenuButton(SizeDefault, false, twi.Text("▣"), twi.Text("Acme Inc"))))),
						SidebarContent(SidebarGroup(SidebarGroupLabel(twi.Text("Platform")), SidebarMenu(
							group.Node(SidebarMenuItem(
								SidebarMenuButton(SizeDefault, true, group.AsTrigger(), twi.Text("▸"), twi.Text("Playground")),
								group.Content(SidebarMenuSub(SidebarMenuSubItem(SidebarMenuSubButton(false, twi.Text("History"))))),
							)),
							SidebarMenuItem(SidebarMenuButton(SizeDefault, false, twi.Text("◔"), twi.Text("Sales")), SidebarMenuBadge(twi.Text("12"))),
						))),
					),
					SidebarInset(twi.Element(twi.Class("flex flex-row gap-1 px-2"), side.Trigger(), field.Node())),
				))
			}
		})
		expect := expecter(t, d)
		edge := func(s string) int {
			_, y, ok := at(d.Frame(), s)
			if !ok {
				return -1
			}
			return slices.Index([]rune(strings.Split(d.Frame().Text(), "\n")[y]), '▏')
		}
		iconX, iconY, _ := at(d.Frame(), "▸")
		expect("expanded: labels, the sub menu and the badge show", has(d, "Platform") && has(d, "Playground") && has(d, "History") && has(d, "12") && has(d, "Acme Inc"))
		expect("expanded width follows the breakpoint", edge("▸") == c.expanded-1)
		t.Logf("%d columns, expanded:\n%s", c.width, d.Frame().Text())
		hit(d, "ctrl+b")
		x, y, _ := at(d.Frame(), "▸")
		expect("ctrl+b collapses to the icons", !side.Open && !has(d, "Platform") && !has(d, "Playground") && !has(d, "History") && !has(d, "12") && !has(d, "Sales"))
		expect("collapsed width is 6 at every breakpoint", edge("▸") == 5)
		expect("the icon keeps its column and moves up the group label's row", x == iconX && y == iconY-1)
		t.Logf("%d columns, collapsed:\n%s", c.width, d.Frame().Text())
		hit(d, "ctrl+b")
		expect("ctrl+b again expands", side.Open && has(d, "Playground") && edge("▸") == c.expanded-1)
		tx, ty, _ := at(d.Frame(), "◧")
		settledClick(d, tx, ty)
		expect("a click on the trigger collapses once", !side.Open)
		hit(d, "enter")
		expect("enter on the focused trigger toggles once", side.Open)
		hit(d, "tab ctrl+b")
		expect("ctrl+b in a focused input still toggles and types nothing", field.focused && !side.Open && field.Value() == "")
		hit(d, "ctrl+b")
		px, py, _ := at(d.Frame(), "Playground")
		settledClick(d, px, py)
		expect("a click on the group's button closes its sub menu", !group.Open && !has(d, "History"))
		hit(d, "enter")
		expect("enter on the focused group button opens it again", group.Open && has(d, "History"))
		if want := []bool{false, true, false, true, false, true}; !slices.Equal(changes, want) {
			t.Errorf("OnOpenChange calls %v, want %v", changes, want)
		}
	}
}

func TestScrollArea(t *testing.T) {
	d := overlayDriver(t, 40, 14, func(*twi.Runtime) func() twi.Node {
		rows := []twi.NodeOption{twi.Class("h-8 w-16 rounded-md border"), twi.Element(twi.Class("h-1 w-full shrink-0 bg-primary"))}
		for i := range 20 {
			rows = append(rows, twi.Text("row "+string(rune('a'+i))))
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-row gap-2 p-1 h-full bg-background text-foreground"),
				ScrollArea(rows...),
				ScrollArea(twi.Class("h-8 w-16 rounded-md border"), twi.Text("short")),
			)
		}
	})
	expect := expecter(t, d)
	lines := func() []string { return strings.Split(d.Frame().Text(), "\n") }
	ax, ay, _ := at(d.Frame(), "row a")
	thumb := func(col int) int {
		n := 0
		for _, l := range lines()[ay-1 : ay+5] {
			if r := []rune(l); col < len(r) && strings.ContainsRune("▁▂▃▄▅▆▇█", r[col]) {
				n++
			}
		}
		return n
	}
	bar := ax + 16 - 3
	primary := zinc(t, theme.Light).Tokens[theme.Primary].RGBA
	expect("the first rows show, the rest are clipped", has(d, "row a") && has(d, "row e") && !has(d, "row f"))
	expect("a thumb runs down its own column", thumb(bar) > 0 && thumb(bar) < 6)
	expect("the content stops one column short of the thumb", d.Frame().Cells().At(bar-1, ay-1).Bg.RGBA == primary && d.Frame().Cells().At(bar, ay-1).Bg.RGBA != primary)
	sx, _, _ := at(d.Frame(), "short")
	expect("an area that fits draws no thumb", thumb(sx+16-3) == 0)
	t.Logf("scroll areas, 40x14:\n%s", d.Frame().Text())
	d.Wheel(ax, ay, 1)
	d.Advance(settleTime)
	expect("the wheel scrolls three rows", !has(d, "row b") && has(d, "row c") && has(d, "row h"))
	hit(d, "tab down")
	expect("with focus, down scrolls a row", !has(d, "row c") && has(d, "row i"))
	hit(d, "end")
	expect("end reaches the last row", has(d, "row t") && !has(d, "row n"))
	t.Logf("scrolled to the end:\n%s", d.Frame().Text())
}

func TestOverlayClosingFrame(t *testing.T) {
	var (
		pop     *Popover
		focuses *twi.Signal[int]
		states  []string
	)
	d := overlayDriver(t, 60, 12, func(rt *twi.Runtime) func() twi.Node {
		pop, focuses = NewPopover(rt), twi.NewSignal(rt, 0)
		return func() twi.Node {
			refocus := twi.OnFocus(func() { focuses.Set(focuses.Get() + 1) })
			n := twi.Element(twi.Class("flex flex-col py-1 pl-20 h-full bg-background text-foreground"),
				pop.Node(pop.Trigger(Outline, SizeDefault, refocus, twi.Text("open")), pop.Content(twi.Element(twi.Text("Dimensions")))))
			state := "removed"
			if reflect.ValueOf(n).FieldByName("tree").FieldByName("Children").Index(0).FieldByName("Children").Index(1).FieldByName("Children").Len() == 1 {
				state = dataState(n, []int{0, 1, 0})
			}
			if len(states) == 0 || states[len(states)-1] != state {
				states = append(states, state)
			}
			return n
		}
	})
	expect := expecter(t, d)
	hit(d, "tab enter")
	expect("open: the content carries data-state=open", pop.Open && has(d, "Dimensions") && slices.Equal(states, []string{"removed", "open"}))
	before := focuses.Get()
	d.Press("escape")
	expect("the closing frame is painted although the focus returning to the trigger sets a signal in the same frame: render holds the content through its exit",
		!pop.Open && pop.focused && focuses.Get() == before+1 && has(d, "Dimensions"))
	d.Advance(settleTime)
	expect("closing builds the content once more with data-state=closed, then removes it from its parent", !has(d, "Dimensions") &&
		slices.Equal(states, []string{"removed", "open", "closed", "removed"}))
	d.Press("enter")
	d.Press("escape")
	d.Press("enter")
	d.Advance(settleTime)
	expect("reopened while closing, it is open", pop.Open && states[len(states)-1] == "open" && has(d, "Dimensions"))
	t.Logf("data-state per build: %v", states)
}
