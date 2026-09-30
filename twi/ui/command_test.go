package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/ui"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func TestCommandScore(t *testing.T) {
	for _, c := range []struct {
		value, search string
		want          bool
	}{
		{"Settings", "", true},
		{"Settings", "set", true},
		{"Settings", "SET", true},
		{"Settings", "tin", true},
		{"Settings", "stg", true},
		{"Settings", "gts", false},
		{"Settings", "settingsx", false},
		{"Émoji", "émo", true},
		{"Émoji", "ÉMO", true},
		{"", "a", false},
	} {
		if got := score(c.value, c.search) > 0; got != c.want {
			t.Errorf("score(%q, %q) matches %v, want %v", c.value, c.search, got, c.want)
		}
	}
	ranked := []string{"Profile", "Calendar", "Billing", "Plain", "Help pl", "Pale"}
	slices.SortStableFunc(ranked, func(a, b string) int { return score(b, "pl") - score(a, "pl") })
	if want := []string{"Plain", "Help pl", "Pale", "Profile"}; !slices.Equal(ranked[:4], want) {
		t.Errorf("ranked for pl %v, want a prefix first, then an earlier substring, then subsequences by span, %v", ranked, want)
	}
	if score("Calendar", "pl") != 0 || score("Billing", "pl") != 0 {
		t.Error("items without every letter in order match")
	}
	if n := testing.AllocsPerRun(100, func() { score("Search Emoji", "emj") }); n != 0 {
		t.Errorf("score allocates %v times", n)
	}
}

func commandApp(t *testing.T, height int, extra *int) (*drive.Driver, *CommandDialog, *[]string, *twi.Runtime) {
	var (
		palette *CommandDialog
		chosen  []string
		runtime *twi.Runtime
	)
	d := overlayDriver(t, 100, height, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		runtime, palette = rt, NewCommandDialog(rt)
		palette.OnSelect = func(v string) { chosen = append(chosen, v) }
		return func() twi.Node {
			var many []CommandItem
			for i := 40 - *extra; i < 40; i++ {
				many = append(many, palette.Item("Item "+string(rune('A'+i/26))+string(rune('a'+i%26))))
			}
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"),
				twi.Text("Press Ctrl K"),
				palette.Node(
					palette.Input("Type a command or search..."),
					palette.List(
						palette.Group("Suggestions", palette.Item("Calendar"), palette.Item("Search Emoji"), palette.Item("Calculator")),
						palette.Separator(),
						palette.Group("Settings",
							palette.Item("Profile", twi.Text("Profile"), CommandShortcut(twi.Text("⌘P"))),
							palette.Item("Billing", twi.Text("Billing"), CommandShortcut(twi.Text("⌘B"))),
							palette.Item("Settings", twi.Text("Settings"), CommandShortcut(twi.Text("⌘S"))),
						),
						palette.Group("More", many...),
					),
				),
			)
		}
	})
	return d, palette, &chosen, runtime
}

func TestCommandFilterAndChoose(t *testing.T) {
	d, palette, chosen, _ := commandApp(t, 30, new(int))
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: open %v, selected %q, visible %v, chosen %v:\n%s", what, palette.Open, palette.selected, palette.visible, *chosen, d.Frame().Text())
		}
	}
	selectedRow := func() string { return line(d, palette.selected) }
	expect("closed at first", !palette.Open && !has("Suggestions"))
	settledPress(d, "ctrl+k")
	t.Logf("open, 100x30, zinc light:\n%s", d.Frame().Text())
	expect("ctrl+k opens the dialog with both groups, the separator and the shortcuts", palette.Open && has("Suggestions") && has("Settings") && has("⌘P") && has("Type a command or search..."))
	expect("the first item is selected", palette.selected == "Calendar")
	settledPress(d, "down")
	settledPress(d, "down")
	settledPress(d, "down")
	expect("down moves across the group boundary", palette.selected == "Profile" && strings.Contains(selectedRow(), "⌘P"))
	settledPress(d, "up")
	expect("up moves back", palette.selected == "Calculator")
	settledPress(d, "up")
	settledPress(d, "up")
	settledPress(d, "up")
	expect("up stops at the first item", palette.selected == "Calendar")
	separator := strings.Count(d.Frame().Text(), "─")
	d.Type("R")
	t.Logf("typed R:\n%s", d.Frame().Text())
	expect("typing filters ignoring case and selects the best match", palette.selected == "Profile" && slices.Equal(palette.visible, []string{"Profile", "Search Emoji", "Calendar", "Calculator"}) && !has("Billing"))
	_, settings, _ := at(d.Frame(), "Settings")
	_, suggestions, _ := at(d.Frame(), "Suggestions")
	_, emoji, _ := at(d.Frame(), "Search Emoji")
	_, calendar, _ := at(d.Frame(), "Calendar")
	expect("the group holding the better match moves up, and an earlier match ranks first within a group", settings < suggestions && emoji < calendar)
	expect("separators hide while searching", strings.Count(d.Frame().Text(), "─") < separator)
	settledPress(d, "backspace")
	d.Type("stg")
	expect("a subsequence matches: stg finds Settings", palette.selected == "Settings" && slices.Equal(palette.visible, []string{"Settings"}))
	d.Type("zz")
	expect("no match shows the empty state and hides every group", has("No results found.") && !has("Suggestions") && !has("Settings "))
	settledPress(d, "enter")
	expect("enter with no match chooses nothing", len(*chosen) == 0 && palette.Open)
	for range 5 {
		settledPress(d, "backspace")
	}
	d.Type("bil")
	settledPress(d, "enter")
	expect("enter chooses the selected match once and closes the dialog", slices.Equal(*chosen, []string{"Billing"}) && !palette.Open)
	settledPress(d, "ctrl+k")
	expect("a reopened dialog starts with an empty search", palette.input.Value() == "" && has("Calendar") && palette.selected == "Calendar")
	x, y, _ := at(d.Frame(), "Search Emoji")
	settledMove(d, x, y)
	expect("hovering an item selects it", palette.selected == "Search Emoji")
	settledClick(d, x, y)
	expect("a click chooses it and closes", slices.Equal(*chosen, []string{"Billing", "Search Emoji"}) && !palette.Open)
	settledPress(d, "ctrl+k")
	settledPress(d, "escape")
	expect("escape closes it without choosing", !palette.Open && len(*chosen) == 2)
}

func TestCommandWindow(t *testing.T) {
	extra := 40
	d, palette, _, rt := commandApp(t, 40, &extra)
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: selected %q, offset %d:\n%s", what, palette.selected, palette.offset, d.Frame().Text())
		}
	}
	visible := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	settledPress(d, "ctrl+k")
	expect("the list shows a window of rows, the later items are below it", visible("Suggestions") && !visible("Item Ab"))
	for range 7 {
		settledPress(d, "down")
	}
	expect("the window follows the selection down", visible("Item Ab") && palette.selected == "Item Ab" && !visible("Suggestions"))
	for range 60 {
		settledPress(d, "down")
	}
	expect("it stops at the last item and never runs past the end", palette.selected == "Item Bn" && visible("Item Bn") && palette.offset == len(palette.visible)+4-konst.CommandRows)
	rt.Dispatch(func() {
		extra = 30
		rt.Invalidate()
	})
	d.Advance(20 * time.Millisecond)
	expect("when the app drops ten items above the window, it stays full instead of running past the end", palette.selected == "Item Bn" && visible("Item Bf") && palette.offset == len(palette.visible)+4-konst.CommandRows)
	for range 60 {
		settledPress(d, "up")
	}
	expect("back at the top the heading of the first group shows again", palette.selected == "Calendar" && visible("Suggestions") && palette.offset == 0)
	d.Type("item b")
	expect("typing resets the window to the first match", palette.selected == "Item Ba" && visible("Item Ba") && palette.offset == 0)
}
