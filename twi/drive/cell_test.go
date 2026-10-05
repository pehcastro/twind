package drive

import (
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
)

func TestCellKeepsTheBufferMeaning(t *testing.T) {
	for _, c := range []struct {
		from buffer.Attr
		want Attr
	}{{buffer.Bold, Bold}, {buffer.Dim, Dim}, {buffer.Italic, Italic}, {buffer.Underline, Underline}, {buffer.Strikethrough, Strikethrough}, {buffer.Inverse, Inverse}} {
		if got := cell(buffer.Cell{Attr: c.from}).Attr; got != c.want {
			t.Errorf("buffer attribute %d reads as %d, want %d", c.from, got, c.want)
		}
	}
	for _, c := range []struct {
		from buffer.Width
		want Width
	}{{buffer.Narrow, Narrow}, {buffer.Wide, Wide}, {buffer.Continuation, Continuation}} {
		if got := cell(buffer.Cell{Width: c.from}).Width; got != c.want {
			t.Errorf("buffer width %d reads as %d, want %d", c.from, got, c.want)
		}
	}
}
