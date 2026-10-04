package ui

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
)

func toastDriver(t *testing.T) (*drive.Driver, *Toaster, *Dialog, *int) {
	var (
		toaster *Toaster
		dlg     *Dialog
		undone  int
		made    int
	)
	d := overlayDriver(t, 100, 30, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		toaster, dlg = NewToaster(rt), NewDialog(rt)
		keys := twi.OnKey(func(k input.KeyEvent) {
			if k.Release || k.Key != input.KeyRune {
				return
			}
			switch k.Rune {
			case 't':
				made++
				toaster.Show("Event "+string(rune('0'+made))+" has been created", "Sunday, December 03, 2023 at 9:00 AM", ToastAction{Label: "Undo", OnClick: func() { undone++ }})
			case 's':
				toaster.Success("Saved", "", ToastAction{})
			case 'e':
				toaster.Error("Event has not been created", "", ToastAction{})
			case 'x':
				toaster.Show("red\x1b[31mtitle\x1b]0;owned\x07", "", ToastAction{})
			case 'd':
				dlg.set(true)
			}
		})
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), keys,
				twi.Text("page"),
				dlg.Content(dlg.Title(twi.Text("A dialog"))),
				toaster.Node(),
			)
		}
	})
	return d, toaster, dlg, &undone
}

func TestToastTiming(t *testing.T) {
	d, toaster, _, _ := toastDriver(t)
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: %d toasts, hovered %v:\n%s", what, len(toaster.toasts), toaster.hovered, d.Frame().Text())
		}
	}
	d.Press("t")
	expect("a toast shows its title and description at the bottom right", has("Event 1 has been created") && has("Sunday, December 03"))
	x, y, _ := at(d.Frame(), "Event 1")
	expect("in the bottom right corner, "+strconv.Itoa(x)+","+strconv.Itoa(y), x > 40 && y > 20)
	t.Logf("one toast, 100x30, twind light:\n%s", d.Frame().Text())
	d.Advance(3900 * time.Millisecond)
	expect("still visible at 3.9 s", has("Event 1 has been created"))
	d.Advance(100 * time.Millisecond)
	expect("gone at 4 s", !has("Event 1") && len(toaster.toasts) == 0)

	d.Press("t")
	d.Advance(time.Second)
	d.Press("t")
	d.Advance(3 * time.Second)
	expect("each toast counts from its own Show: the first is gone at 4 s, the second shown 1 s later stays", !has("Event 2") && has("Event 3"))
	d.Advance(time.Second)
	expect("the second goes 4 s after its own Show", !has("Event 3") && len(toaster.toasts) == 0)

	d.Press("t")
	d.Advance(time.Second)
	x, y, _ = at(d.Frame(), "Event 4")
	d.Move(x, y)
	d.Advance(10 * time.Second)
	expect("hovered, the toast stays past its 4 s", has("Event 4"))
	d.Move(0, 0)
	d.Advance(2900 * time.Millisecond)
	expect("after the pointer leaves it resumes with the 3 s it had left: there at 2.9 s", has("Event 4"))
	d.Advance(100 * time.Millisecond)
	expect("and gone at 3 s", !has("Event 4"))
}

func TestToastStack(t *testing.T) {
	d, toaster, dlg, undone := toastDriver(t)
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s: %d toasts, hovered %v, undone %d:\n%s", what, len(toaster.toasts), toaster.hovered, *undone, d.Frame().Text())
		}
	}
	d.Press("e")
	d.Press("s")
	d.Press("t")
	t.Logf("three toasts, collapsed:\n%s", d.Frame().Text())
	expect("collapsed, only the newest shows its text, in front", has("Event 1 has been created") && !has("Saved") && !has("has not been created"))
	_, front, _ := at(d.Frame(), "Event 1")
	x, y, _ := at(d.Frame(), "Event 1")
	d.Move(x, y)
	t.Logf("three toasts, hovered:\n%s", d.Frame().Text())
	_, saved, _ := at(d.Frame(), "Saved")
	_, failed, _ := at(d.Frame(), "has not been created")
	_, newest, _ := at(d.Frame(), "Event 1")
	expect("hovered, the stack expands upward: oldest on top, newest at the bottom", failed < saved && saved < newest && newest == front)
	expect("success shows its check, error its mark, default neither", strings.Contains(line(d, "Saved"), "✓") && strings.Contains(line(d, "has not been created"), "⊗") && !strings.ContainsAny(line(d, "Event 1"), "✓⊗"))
	d.Press("t")
	expect("a fourth toast keeps the oldest hidden", has("Event 2") && !has("has not been created") && len(toaster.toasts) == 4)
	gone := toaster.toasts[2]
	_, y, _ = at(d.Frame(), "Event 1")
	before, _, _ := strings.Cut(strings.Split(d.Frame().Text(), "\n")[y], "✕")
	d.Click(len([]rune(before)), y)
	expect("the close button removes only its toast and stops its timer", !has("Event 1") && has("Event 2") && has("Saved") && gone.tick == nil && len(toaster.toasts) == 3)
	x, y, _ = at(d.Frame(), "Undo")
	d.Click(x, y)
	expect("the action runs once and dismisses its toast", *undone == 1 && !has("Event 2") && len(toaster.toasts) == 2)
	d.Move(0, 0)
	d.Press("d")
	expect("a dialog opens over the page", dlg.Open)
	d.Press("escape")
	expect("escape closes the dialog first and leaves the toasts", !dlg.Open && len(toaster.toasts) == 2)
	d.Press("escape")
	expect("the next escape dismisses the newest toast", len(toaster.toasts) == 1 && toaster.toasts[0].kind == ToastError)
	d.Press("escape")
	expect("and the next the last one", len(toaster.toasts) == 0 && !has("has not been created"))
	d.Press("x")
	expect("a title holding escape sequences reaches the screen as text", has("red") && has("title") && d.Frame().Cells().At(0, 0).Grapheme != "\x1b")
	t.Logf("sanitised title:\n%s", d.Frame().Text())
}

func TestToastClearsOpenPanels(t *testing.T) {
	var (
		toaster             *Toaster
		right, left, drawer *Dialog
	)
	d := overlayDriver(t, 100, 30, func(rt *twi.Runtime) func() twi.Node {
		toaster, right, left, drawer = NewToaster(rt), NewSheet(rt, Right), NewSheet(rt, Left), NewDrawer(rt, Bottom)
		toaster.Avoid(right, left, drawer)
		panel := func(p *Dialog, title, action string) twi.Node {
			return p.Content(p.Header(p.Title(twi.Text(title))), p.Footer(Button(Default, SizeDefault, twi.Text(action))))
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"),
				twi.Text("page"), panel(right, "Your cart", "Checkout"), panel(left, "Filters", "Leftward"), panel(drawer, "Move goal", "Submit"), toaster.Node())
		}
	})
	expect := expecter(t, d)
	corners := func() (x0, x1, y1 int) {
		_, ty, _ := at(d.Frame(), "Added to cart")
		lines := strings.Split(d.Frame().Text(), "\n")
		top, bottom := []rune(lines[ty-2]), []rune(lines[ty+3])
		x0, x1 = slices.Index(top, '╭'), slices.Index(bottom, '╯')
		return x0, x1, ty + 3
	}
	column := func(title string) int {
		x, _, _ := at(d.Frame(), title)
		return x
	}
	show := func(p *Dialog) {
		p.set(true)
		d.Advance(settleTime)
		toaster.Success("Added to cart", "Hearth mug, Lake, 16 oz", ToastAction{})
		d.Advance(settleTime)
	}
	hide := func(p *Dialog) {
		p.set(false)
		toaster.dismiss(toaster.toasts[0])
		d.Advance(settleTime)
	}

	show(right)
	x0, x1, _ := corners()
	t.Logf("right sheet and a toast, 100x30:\n%s", d.Frame().Text())
	expect("a right sheet: Checkout stays visible", has(d, "Checkout"))
	expect("a right sheet: the toast "+strconv.Itoa(x0)+".."+strconv.Itoa(x1)+" sits beside the sheet, whose title starts at "+strconv.Itoa(column("Your cart")), x0 >= 0 && x1 >= 0 && x1 < column("Your cart")-3)
	hide(right)

	show(drawer)
	_, ty, _ := at(d.Frame(), "Move goal")
	_, _, y1 := corners()
	t.Logf("bottom drawer and a toast, 100x30:\n%s", d.Frame().Text())
	expect("a bottom drawer: Submit stays visible", has(d, "Submit"))
	expect("a bottom drawer: the toast's last row "+strconv.Itoa(y1)+" sits above the drawer title row "+strconv.Itoa(ty), y1 < ty-2)
	hide(drawer)

	show(left)
	x0, _, _ = corners()
	t.Logf("left sheet and a toast, 100x30:\n%s", d.Frame().Text())
	edge := slices.Index([]rune(line(d, "Filters")), '│')
	expect("a left sheet: Leftward stays visible and the toast "+strconv.Itoa(x0)+" stays right of the sheet's edge "+strconv.Itoa(edge), has(d, "Leftward") && edge > 0 && x0 > edge)
	left.set(false)
	d.Advance(settleTime)
	x, y, _ := at(d.Frame(), "Added to cart")
	expect("closed: the toast is back in the bottom right corner", x > 50 && y > 20)
}

func TestToastAvoidsOnlyPanels(t *testing.T) {
	rt := twi.New()
	for name, d := range map[string]*Dialog{"dialog": NewDialog(rt), "alert dialog": NewAlertDialog(rt)} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Avoid(%s) did not panic", name)
				}
			}()
			NewToaster(rt).Avoid(d)
		}()
	}
}

func line(d *drive.Driver, s string) string {
	_, y, _ := at(d.Frame(), s)
	return strings.Split(d.Frame().Text(), "\n")[y]
}
