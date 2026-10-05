package edit

import "github.com/pehcastro/twind/twi/text"

type Unit uint8

const (
	Grapheme Unit = iota
	Word
	Line
)

func (b *Buffer) At(row, column int) int {
	rows := b.Rows()
	r := rows[max(min(row, len(rows)-1), 0)]
	at, x := r.Start, 0
	for g := range text.Graphemes(b.value[r.Start:r.End]) {
		w := b.Widths.Width(g)
		if column < x+w {
			if column-x < (w+1)/2 {
				return at
			}
			return at + len(g)
		}
		x, at = x+w, at+len(g)
	}
	return at
}

func (b *Buffer) Press(at int, u Unit, extend bool) {
	if extend {
		b.unit, b.grabbed = Grapheme, [2]int{b.anchor, b.anchor}
		b.move(at, true)
		return
	}
	lo, hi := b.span(at, u)
	b.unit, b.grabbed = u, [2]int{lo, hi}
	b.anchor, b.cursor, b.last = lo, hi, stepOther
}

func (b *Buffer) Drag(at int) {
	lo, hi := b.span(at, b.unit)
	if lo < b.grabbed[0] {
		b.anchor, b.cursor = b.grabbed[1], lo
	} else {
		b.anchor, b.cursor = b.grabbed[0], max(hi, b.grabbed[1])
	}
	b.last = stepOther
}

func (b *Buffer) span(at int, u Unit) (lo, hi int) {
	line, end := b.lineStart(at), b.lineEnd(at)
	switch u {
	case Grapheme:
		return at, at
	case Line:
		return line, end
	case Word:
		for w := range text.Words(b.value[line:end]) {
			if next := line + len(w); at < next || next == end {
				return line, next
			}
			line += len(w)
		}
		return at, at
	}
	panic("edit: unknown unit")
}
