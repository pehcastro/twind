package docsapp

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

func framed(t *testing.T, page string) (*drive.Driver, screenRows, [4]int) {
	t.Helper()
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		view, err := New(rt, Start{Page: page, Theme: "twind-dark"})
		if err != nil {
			t.Fatal(err)
		}
		return view
	}, drive.Size(120, 36), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := d.Err(); err != nil {
			t.Error(err)
		}
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	for notches := 0; ; notches++ {
		var rows screenRows
		for line := range strings.SplitSeq(d.Frame().Text(), "\n") {
			rows = append(rows, []rune(line))
		}
		top, left, right, bottom, whole := rows.box(0, 0, 120)
		if whole {
			return d, rows, [4]int{top, left, right, bottom}
		}
		if notches == 4 || left < 0 {
			t.Fatalf("%s: no whole preview frame after %d wheel notches:\n%s", page, notches, d.Frame().Text())
		}
		d.Wheel(60, top, 1)
	}
}

func TestButtonGroupDemoInsideItsFrame(t *testing.T) {
	d, rows, f := framed(t, "button-group")
	for y := f[0] + 1; y < f[3]; y++ {
		if rows.at(f[1]+1, y) != ' ' || rows.at(f[2]-1, y) != ' ' {
			t.Errorf("row %d of the preview touches its border:\n%s", y, d.Frame().Text())
		}
	}
	for _, label := range []string{"Archive", "Snooze", "twind.dev"} {
		if x, _ := spot(t, d, label); x <= f[1] || x >= f[2] {
			t.Errorf("%s at column %d, outside the frame %d to %d", label, x, f[1], f[2])
		}
	}
}

func TestDialogShowsNoPageThroughItWhileItMoves(t *testing.T) {
	d, _, _ := framed(t, "dialog")
	rest := func() {
		d.Move(119, 35)
		d.Advance(time.Second)
	}
	rest()
	before := d.Frame()
	clean := func(when string) {
		for y, line := range strings.Split(d.Frame().Text(), "\n") {
			row := []rune(line)
			left := slices.IndexFunc(row, func(r rune) bool { return r == '▐' })
			right := slices.Index(row, '▌')
			if left < 0 || right < left+3 || !strings.ContainsRune("│╭╰", row[left+1]) {
				continue
			}
			inside := string(row[left+2 : right-1])
			for _, word := range []string{"overlaid", "underneath", "inert", "Preview", "Code", "Usage", "NewDialog"} {
				if strings.Contains(inside, word) {
					t.Errorf("%s: the page's %q shows inside the dialog on row %d:\n%s", when, word, y, d.Frame().Text())
					return
				}
			}
		}
	}
	x, y := spot(t, d, "Edit profile")
	d.Click(x+1, y)
	for step := range 15 {
		clean(fmt.Sprintf("open +%d ms", step*10))
		d.Advance(10 * time.Millisecond)
	}
	d.Advance(time.Second)
	t.Logf("open:\n%s", d.Frame().Text())
	x, y = spot(t, d, "Cancel")
	d.Click(x+1, y)
	for step := range 30 {
		if step == 3 {
			t.Logf("mid close, +30 ms:\n%s", d.Frame().Text())
		}
		clean(fmt.Sprintf("close +%d ms", step*10))
		d.Advance(10 * time.Millisecond)
	}
	d.Advance(time.Second)
	rest()
	t.Logf("closed:\n%s", d.Frame().Text())
	after := d.Frame().Cells()
	for y := range after.Height() {
		for x := range after.Width() {
			if was, is := before.Cells().At(x, y), after.At(x, y); was != is {
				t.Fatalf("the closed page differs at %d,%d from the page before the dialog opened: %+v, was %+v:\n%s", x, y, is, was, d.Frame().Text())
			}
		}
	}
}

func TestDrawerDemoFitsAndDrags(t *testing.T) {
	d, _, _ := framed(t, "drawer")
	x, y := spot(t, d, "Open drawer")
	d.Click(x+1, y)
	d.Advance(time.Second)
	cells := d.Frame().Cells()
	sx, sy := spot(t, d, "Submit")
	fill := cells.At(sx, sy).Bg
	left, right := sx, sx
	for left > 0 && cells.At(left-1, sy).Bg == fill {
		left--
	}
	for right < cells.Width()-1 && cells.At(right+1, sy).Bg == fill {
		right++
	}
	if right-left+1 > 48 || cells.At(sx, sy-1).Bg != fill || cells.At(sx, sy+1).Bg != fill {
		t.Errorf("Submit fills columns %d to %d; want 48 at most and a row of fill above and below:\n%s", left, right, d.Frame().Text())
	}
	cx, cy := spot(t, d, "Cancel")
	if cells.At(cx, cy-1).Grapheme != "─" || cells.At(cx, cy+1).Grapheme != "─" {
		t.Errorf("Cancel: want its border on the rows above and below:\n%s", d.Frame().Text())
	}
	_, top := spot(t, d, "Move goal")
	hx, hy := cells.Width()/2, top
	for hy > 0 && cells.At(hx, hy).Bg == cells.At(1, hy).Bg {
		hy--
	}
	t.Logf("drawer open:\n%s", d.Frame().Text())
	d.Down(hx, hy)
	d.Move(hx, hy+6)
	d.Advance(20 * time.Millisecond)
	t.Logf("drawer dragged six rows down by its handle:\n%s", d.Frame().Text())
	if _, moved := spot(t, d, "Move goal"); moved != top+6 {
		t.Errorf("the title is on row %d mid drag, want %d", moved, top+6)
	}
	d.Up(hx, hy+6)
	d.Advance(time.Second)
	if strings.Contains(d.Frame().Text(), "Move goal") {
		t.Errorf("a drag of six rows down did not close the drawer:\n%s", d.Frame().Text())
	}
}

func TestCardDemoButtonsHaveRowsAboveAndBelow(t *testing.T) {
	d, _, _ := framed(t, "card")
	cells := d.Frame().Cells()
	for _, label := range []string{" Login  ", "Login with Google"} {
		x, y := spot(t, d, label)
		button := func(dy int) bool {
			c := cells.At(x, y+dy)
			return c.Bg == cells.At(x, y).Bg || c.Grapheme == "─"
		}
		if !button(-1) || !button(1) || button(-2) || button(2) {
			t.Errorf("%q: want its button's fill or edge on the row above and below and no further:\n%s", label, d.Frame().Text())
		}
	}
}
