package docsapp

import (
	"strings"
	"testing"

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
