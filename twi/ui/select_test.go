package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/theme"
)

func TestSelectKeysAndPointer(t *testing.T) {
	var (
		fruit, size *Select
		changes     []string
	)
	d := overlayDriver(t, 80, 20, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		fruit, size = NewSelect(rt), NewSelect(rt)
		fruit.Placeholder = "Select a fruit"
		fruit.OnChange = func(v string) { changes = append(changes, v) }
		size.Value = "md"
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				fruit.Node(fruit.Trigger(), fruit.Content(
					SelectLabel(twi.Text("Fruits")),
					fruit.Item("apple", "Apple"), fruit.Item("banana", "Banana"), fruit.Item("blueberry", "Blueberry"),
					SelectSeparator(),
					fruit.Item("grapes", "Grapes"), fruit.Item("pineapple", "Pineapple"),
				)),
				twi.Element(twi.Class("font-medium"), twi.Text("A later sibling under the select")),
				size.Node(size.Trigger(), size.Content(size.Item("sm", "Small"), size.Item("md", "Medium"))),
				twi.Element(twi.Text("Elsewhere")),
			)
		}
	})
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: value %q, open %v, active %d, changes %v:\n%s", what, fruit.Value, fruit.Open, fruit.active, changes, d.Frame().Text())
		}
	}
	expect("the placeholder shows while nothing is chosen", has("Select a fruit") && !has("Apple"))
	expect("a preset value shows its label from the first frame", has("Medium") && !has("md"))
	hit(d, "tab enter")
	expect("enter opens on the first item when nothing is chosen", fruit.Open && fruit.active == 0 && !fruit.focused && has("Apple") && has("Pineapple"))
	expect("the content covers the later sibling instead of sitting under it", !has("A later sibling"))
	hit(d, "up")
	expect("up on the first item stays", fruit.active == 0)
	hit(d, "down down")
	expect("down moves the highlight", fruit.active == 2)
	hit(d, "enter")
	expect("enter chooses, closes and returns focus to the trigger showing the label", fruit.Value == "blueberry" && !fruit.Open && fruit.focused && has("Blueberry") && has("A later sibling"))
	hit(d, "down")
	expect("down opens on the chosen item, marked", fruit.Open && fruit.active == 2 && has("Blueberry ✓"))
	hit(d, "end down")
	expect("end goes to the last item and down stays there", fruit.active == 4)
	hit(d, "home")
	expect("home goes to the first", fruit.active == 0)
	hit(d, "escape")
	expect("escape closes without a change", !fruit.Open && fruit.Value == "blueberry" && fruit.focused)
	hit(d, "up g")
	expect("up opens, g moves to Grapes", fruit.Open && fruit.active == 3)
	hit(d, "b B")
	expect("typeahead wraps and ignores case: b to Banana, B to Blueberry", fruit.active == 2)
	hit(d, "space")
	expect("space on the chosen item closes with no change", !fruit.Open && slices.Equal(changes, []string{"blueberry"}))
	hit(d, "p")
	expect("a letter on the closed trigger chooses the next match", !fruit.Open && fruit.Value == "pineapple" && has("Pineapple"))
	x, y, _ := at(d.Frame(), "Pineapple")
	settledClick(d, x, y)
	expect("a click on the trigger opens it", fruit.Open)
	settledClick(d, x, y)
	expect("a second click closes it", !fruit.Open)
	settledClick(d, x, y)
	gx, gy, _ := at(d.Frame(), "Grapes")
	settledMove(d, gx, gy)
	expect("the pointer highlights the item under it", fruit.active == 3)
	settledClick(d, gx, gy)
	expect("a click chooses the item and closes", fruit.Value == "grapes" && !fruit.Open)
	settledClick(d, x, y)
	ex, ey, _ := at(d.Frame(), "Elsewhere")
	settledClick(d, ex, ey)
	expect("a pointer down outside closes without a change", !fruit.Open && fruit.Value == "grapes")
	if want := []string{"blueberry", "pineapple", "grapes"}; !slices.Equal(changes, want) {
		t.Errorf("OnChange calls %v, want %v", changes, want)
	}
	expect("items do not pile up across frames", len(fruit.items) == 5 && len(size.items) == 2)
}
