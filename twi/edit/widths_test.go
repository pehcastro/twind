package edit

import (
	"testing"

	"github.com/twind-dev/twind/twi/text"
)

func TestWidthsCursorAfterFlag(t *testing.T) {
	for _, c := range []struct {
		widths text.Widths
		column int
	}{
		{text.Widths{text.Flag: 1}, 2},
		{text.Widths{}, 3},
	} {
		b := Buffer{Mode: MultiLine, Widths: c.widths}
		b.Insert("abcd\n🇧🇷x")
		wantCursor(t, &b, 1, c.column)
		press(t, &b, up)
		wantCursor(t, &b, 0, c.column)
		press(t, &b, down)
		wantCursor(t, &b, 1, c.column)
		press(t, &b, home, up, end, down)
		wantCursor(t, &b, 1, c.column)
		press(t, &b, home, right)
		wantCursor(t, &b, 1, c.column-1)
	}
}
