package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
)

func TestToastCloseSitsOnTheTitleRow(t *testing.T) {
	d, toaster, _, _ := toastDriver(t)
	settledPress(d, "t")
	t.Logf("a toast with an action and a close:\n%s", d.Frame().Text())
	title := line(d, "Event 1 has been created")
	_, y, _ := at(d.Frame(), "Event 1 has been created")
	above := strings.Split(d.Frame().Text(), "\n")[y-1]
	undo, close := strings.Index(title, "Undo"), strings.Index(title, "✕")
	edge := strings.LastIndex(title, "│")
	if strings.Contains(above, "✕") || undo < 0 || close < undo || edge < close {
		t.Errorf("want the close after the action on the title row, inside the border; above %q, title %q", above, title)
	}
	x, y, _ := at(d.Frame(), "✕")
	settledClick(d, x, y)
	if len(toaster.toasts) != 0 {
		t.Errorf("the close on the title row did not dismiss the toast:\n%s", d.Frame().Text())
	}
}

func TestToastCloseInsideTheToast(t *testing.T) {
	for _, width := range []int{100, 50, 30} {
		var toaster *Toaster
		d := overlayDriver(t, width, 20, func(rt *twi.Runtime) func() twi.Node {
			toaster = NewToaster(rt)
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"), twi.Text("page"), toaster.Node())
			}
		})
		toaster.Show("Your event has been created and shared with everyone", "Sunday, December 03, 2023 at 9:00 AM in the main hall", ToastAction{Label: "Undo"})
		d.Advance(settleTime)
		lines := strings.Split(d.Frame().Text(), "\n")
		_, top, _ := at(d.Frame(), "╭")
		_, y, _ := at(d.Frame(), "✕")
		row := []rune(lines[y])
		left, right, undo := slices.Index(row, '│'), -1, -1
		for i, r := range row {
			if r == '│' {
				right = i
			}
			if strings.HasPrefix(string(row[i:]), "Undo") {
				undo = i
			}
		}
		if close := slices.Index(row, '✕'); left < 0 || right <= close || undo <= left || close <= undo || y != top+2 {
			t.Errorf("%d columns: want the action then the close between the borders on the first body row (row %d), got row %d:\n%s", width, top+2, y, d.Frame().Text())
		}
		t.Logf("%d columns:\n%s", width, strings.Join(lines[top:], "\n"))
	}
}

func TestSidebarHoverIsLighterThanActive(t *testing.T) {
	d := overlayDriver(t, 60, 12, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		s := NewSidebar(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"), s.Provider(s.Node(SidebarContent(SidebarGroup(SidebarMenu(
				SidebarMenuItem(SidebarMenuButton(SidebarMenuButtonSizeDefault, twi.Text("Home"))),
				SidebarMenuItem(SidebarMenuButton(SidebarMenuButtonSizeDefault, Active(true), twi.Text("Inbox"))),
				SidebarMenuItem(SidebarMenuButton(SidebarMenuButtonSizeDefault, twi.Text("Drafts"))),
			))))))
		}
	})
	look := func(word string) (color.Color, color.Color) {
		x, y, _ := at(d.Frame(), word)
		c := d.Frame().At(x, y)
		return c.Bg, c.Fg
	}
	x, y, _ := at(d.Frame(), "Home")
	idle, _ := look("Home")
	activeBg, activeFg := look("Inbox")
	settledMove(d, x, y)
	hoverBg, hoverFg := look("Home")
	if hoverBg == idle || hoverBg == activeBg || hoverFg == activeFg {
		t.Errorf("hover bg %v fg %v, active bg %v fg %v, idle bg %v: want hover visible and lighter than active, not a second active row", hoverBg.RGBA, hoverFg.RGBA, activeBg.RGBA, activeFg.RGBA, idle.RGBA)
	}
	x, y, _ = at(d.Frame(), "Inbox")
	settledMove(d, x, y)
	if bg, fg := look("Inbox"); bg != activeBg || fg != activeFg {
		t.Errorf("hover on the active item changed its look to bg %v fg %v, want the active bg %v fg %v", bg.RGBA, fg.RGBA, activeBg.RGBA, activeFg.RGBA)
	}
}

func TestComboboxIconInsideTheField(t *testing.T) {
	var (
		box     *Combobox
		changes []string
	)
	d := overlayDriver(t, 60, 16, func(rt *twi.Runtime) func() twi.Node {
		box = NewCombobox(rt)
		box.Placeholder, box.ShowClear = "Pick one", true
		box.OnChange = func(v string) { changes = append(changes, v) }
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"),
				box.Node(twi.Class("w-30"), box.Input(), box.Content(box.Item("next", "Next.js"), box.Item("nuxt", "Nuxt.js"))),
			)
		}
	})
	inside := func(step, text, glyph string) {
		t.Helper()
		lines := strings.Split(d.Frame().Text(), "\n")
		_, y, ok := at(d.Frame(), text)
		row := []rune(lines[y])
		left, right := slices.Index(row, '│'), strings.LastIndex(string(row), "│")
		right = len([]rune(string(row)[:max(right, 0)]))
		g := slices.Index(row, []rune(glyph)[0])
		if !ok || y < 1 || !strings.Contains(lines[y-1], "╭") || !strings.Contains(lines[y+1], "╰") || g <= left || g >= right || strings.TrimSpace(string(row[g+1:right])) != "" {
			t.Errorf("%s: want %s on %q's row, the last thing inside the border, with border rows above and below:\n%s", step, glyph, text, d.Frame().Text())
		}
	}
	inside("closed, empty", "Pick one", "⌄")
	hit(d, "tab type:nu enter")
	inside("a value chosen, ShowClear", "Nuxt.js", "✕")
	x, y, _ := at(d.Frame(), "✕")
	settledClick(d, x, y)
	if box.Value != "" || box.field.Value() != "" || !slices.Equal(changes, []string{"nuxt", ""}) {
		t.Errorf("the clear button left value %q, field %q, changes %v", box.Value, box.field.Value(), changes)
	}
	inside("cleared", "Pick one", "⌄")
	t.Logf("combobox after clear:\n%s", d.Frame().Text())
}

func TestPaletteHasNoCloseButton(t *testing.T) {
	d, _, _, _ := commandApp(t, 40, new(int))
	settledPress(d, "ctrl+k")
	if strings.Contains(d.Frame().Text(), "✕") {
		t.Errorf("the palette draws a close button over its search row, shadcn's CommandDialog has none:\n%s", d.Frame().Text())
	}
}
