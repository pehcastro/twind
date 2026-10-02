package app

import (
	"strings"
	"testing"
	"time"
)

func TestClickPlacesTheCaretInTheForm(t *testing.T) {
	d := open(t)
	d.Click(spot(t, d, "⌕ ui"))
	d.Type("field")
	d.Press("enter")
	d.Advance(time.Second)
	form := func(step string) string {
		_, top := spot(t, d, "Name on card")
		y := top + 3
		row := d.Frame().Cells().Row(y)
		left := strings.Index(lines(d)[y], "│")
		var marked strings.Builder
		for x := left; x < len(row); x++ {
			if c := row[x]; c.Bg != row[left+1].Bg {
				marked.WriteString("[" + c.Grapheme + "]")
			} else {
				marked.WriteString(c.Grapheme)
			}
		}
		shown := strings.ReplaceAll(marked.String(), "\u00a0", " ")
		t.Logf("%s:\n%s", step, shown)
		return shown
	}
	d.Click(spot(t, d, "m@example.com"))
	d.Type("Peedro")
	d.Advance(time.Second)
	if row := form("typed Peedro"); !strings.Contains(row, "Peedro[ ]") {
		t.Errorf("the caret is not after Peedro: %q", row)
	}
	x, y := spot(t, d, "Peedro")
	d.Click(x+2, y)
	if row := form("clicked between the e's"); !strings.Contains(row, "Pe[e]dro") {
		t.Errorf("the caret is not on the second e: %q", row)
	}
	d.Press("backspace")
	if row := form("Backspace"); !strings.Contains(row, "P[e]dro") {
		t.Errorf("the field row reads %q, want P[e]dro", row)
	}
}
