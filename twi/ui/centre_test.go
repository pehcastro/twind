package ui

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func reach(b *buffer.Buffer, x, y, dx, dy int) (n int) {
	for cx, cy := x+dx, y+dy; cx >= 0 && cy >= 0 && cx < b.Width() && cy < b.Height() && b.At(cx, cy).Bg == b.At(x, y).Bg; cx, cy = cx+dx, cy+dy {
		n++
	}
	return n
}

func centred(t *testing.T, name string, d *drive.Driver, glyph string) {
	t.Helper()
	x, y, ok := at(d.Frame(), glyph)
	if !ok {
		t.Fatalf("%s: no %q in the frame:\n%s", name, glyph, d.Frame().Text())
	}
	b := d.Frame().Cells()
	left, right, up, down := reach(b, x, y, -1, 0), reach(b, x, y, 1, 0), reach(b, x, y, 0, -1), reach(b, x, y, 0, 1)
	if left == 0 || left != right || up != down {
		t.Errorf("%s: %q has %d fill cells left, %d right, %d above, %d below:\n%s", name, glyph, left, right, up, down, d.Frame().Text())
	}
}

func staged(t *testing.T, width, height int, view func(rt *twi.Runtime) func() twi.Node) *drive.Driver {
	t.Helper()
	return overlayDriver(t, width, height, func(rt *twi.Runtime) func() twi.Node {
		inner := view(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col items-start p-1 gap-1 h-full bg-background text-foreground"), inner())
		}
	})
}

func TestGlyphsSitInTheMiddleOfTheirFill(t *testing.T) {
	for name, s := range map[string]Size{"sm": SizeSM, "default": SizeDefault, "lg": SizeLG} {
		d := staged(t, 30, 5, func(rt *twi.Runtime) func() twi.Node {
			g := NewToggleGroup(rt)
			g.Size, g.Value = s, []string{"b"}
			return func() twi.Node { return g.Node(g.Item("b", twi.Text("B")), g.Item("i", twi.Text("I"))) }
		})
		centred(t, "toggle group "+name, d, "B")
	}
	d := staged(t, 30, 5, func(rt *twi.Runtime) func() twi.Node {
		c := NewCheckbox(rt)
		c.Checked = true
		return func() twi.Node { return c.Node() }
	})
	centred(t, "checkbox", d, "✓")
	d = staged(t, 30, 5, func(rt *twi.Runtime) func() twi.Node {
		g := NewRadioGroup(rt)
		g.Value = "a"
		return func() twi.Node { return g.Node(g.Item("a"), g.Item("b")) }
	})
	centred(t, "radio group", d, "●")
	d = staged(t, 40, 6, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return Attachment(Done, Horizontal,
				AttachmentMedia(Icon, twi.Text("▤")),
				AttachmentContent(AttachmentTitle(twi.Text("report.pdf")), AttachmentDescription(twi.Text("1.2 MB"))),
				AttachmentActions(AttachmentAction(twi.Text("✕"))),
			)
		}
	})
	centred(t, "attachment media", d, "▤")
	media, title, _ := at(d.Frame(), "▤")
	close, row, _ := at(d.Frame(), "✕")
	line := []rune(strings.Split(d.Frame().Text(), "\n")[title])
	left, right := slices.Index(line, '│'), slices.Index(line[close:], '│')
	if row != title || left < 0 || right < 0 || media-left != right {
		t.Errorf("attachment: the ✕ on row %d, %d cells from the right border; the icon on row %d, %d cells from the left:\n%s", row, right, title, media-left, d.Frame().Text())
	}
}

func TestCarouselButtonsOnTheMiddleRow(t *testing.T) {
	d := staged(t, 40, 12, func(rt *twi.Runtime) func() twi.Node {
		c := NewCarousel(rt)
		return func() twi.Node {
			return twi.Element(twi.Class("flex px-5"), c.Node(twi.Class("w-20 h-9"), c.Content(c.Item(Card(twi.Class("h-full items-center justify-center"), twi.Text("1")))), c.Previous(), c.Next()))
		}
	})
	_, top, _ := at(d.Frame(), "╭")
	_, bottom, _ := at(d.Frame(), "╰")
	_, slide, _ := at(d.Frame(), "1")
	for _, glyph := range []string{"←", "→"} {
		if _, y, ok := at(d.Frame(), glyph); !ok || y-top != bottom-y || y != slide {
			t.Errorf("carousel %s on row %d, want %d, the middle of rows %d to %d:\n%s", glyph, y, (top+bottom)/2, top, bottom, d.Frame().Text())
		}
	}
}

func TestCalendarAlwaysShowsSixWeeks(t *testing.T) {
	for _, month := range []time.Month{time.February, time.September, time.August} {
		d := staged(t, 44, 22, func(rt *twi.Runtime) func() twi.Node {
			c := NewCalendar(rt)
			c.Month = time.Date(2026, month, 1, 0, 0, 0, 0, time.UTC)
			return func() twi.Node { return c.Node(twi.Class("border")) }
		})
		su, row, _ := at(d.Frame(), "Su")
		sa, _, _ := at(d.Frame(), "Sa")
		weeks := 0
		for _, l := range strings.Split(d.Frame().Text(), "\n")[row+1:] {
			if strings.ContainsAny(l, "0123456789") {
				weeks++
			}
		}
		if weeks != 6 || sa-su <= 24 {
			t.Errorf("%s 2026: %d week rows, want 6; Su to Sa %d cells, want wider than 24:\n%s", month, weeks, sa-su, d.Frame().Text())
		}
	}
}

func TestButtonGroupJoinsItsButtons(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	light := zinc(t, theme.Light)
	outline := func(label string) twi.Node { return Button(Outline, SizeDefault, twi.Text(label)) }
	md, none := style.RadiusMd, style.RadiusNone
	for _, c := range []struct {
		name  string
		o     Orientation
		group twi.Node
		want  [][4]style.Radius
	}{
		{"buttons", Horizontal, ButtonGroup(Horizontal, outline("Archive"), outline("Report"), outline("Snooze")), [][4]style.Radius{{md, none, none, md}, {none, none, none, none}, {none, md, md, none}}},
		{"text and button", Horizontal, ButtonGroup(Horizontal, ButtonGroupText(twi.Text("https://")), outline("twind.dev")), [][4]style.Radius{{md, none, none, md}, {none, md, md, none}}},
		{"column", Vertical, ButtonGroup(Vertical, outline("+"), outline("-")), [][4]style.Radius{{md, md, none, none}, {none, none, md, md}}},
	} {
		tree := rendered(twi.Element(twi.Class("flex flex-col items-start"), c.group))
		built := reflect.NewAt(tree.Type(), unsafe.Pointer(tree.UnsafeAddr())).Elem().Interface().(render.Node)
		root, err := render.Scene(built, render.Frame{Sheet: sheet.WithTheme(&light), Width: 40, Height: layout.Length{Unit: layout.Cells, Value: 6}})
		if err != nil {
			t.Fatal(err)
		}
		group := root.Children[0].Children
		for i, want := range c.want {
			r := group[i].Border.Radius
			if got := [4]style.Radius{r.At(style.CornerTopLeft), r.At(style.CornerTopRight), r.At(style.CornerBottomRight), r.At(style.CornerBottomLeft)}; got != want {
				t.Errorf("%s child %d: corners %v, want %v", c.name, i, got, want)
			}
		}
		for i := 1; i < len(group); i++ {
			a, b := group[i-1].Bounds, group[i].Bounds
			if c.o == Vertical && (b.Y != a.Y+a.H || b.W != a.W) || c.o == Horizontal && b.X != a.X+a.W {
				t.Errorf("%s child %d at %+v does not meet child %d at %+v", c.name, i, b, i-1, a)
			}
		}
	}
}

func TestSmallAvatarFillOutreachesItsLetters(t *testing.T) {
	d := staged(t, 20, 5, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node { return Avatar(SizeSM, AvatarFallback(twi.Text("CN"))) }
	})
	x, y, _ := at(d.Frame(), "CN")
	b := d.Frame().Cells()
	if left, right := reach(b, x, y, -1, 0), reach(b, x+1, y, 1, 0); left != 1 || right != 1 {
		t.Errorf("small avatar: %d fill cells left of CN, %d right, want one each:\n%s", left, right, d.Frame().Text())
	}
}
