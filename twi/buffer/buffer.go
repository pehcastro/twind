package buffer

import "github.com/twind-dev/twind/twi/color"

type Attr uint8

const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	Strikethrough
	Inverse
)

type Width uint8

const (
	Narrow Width = iota
	Wide
	Continuation
)

type Cell struct {
	Grapheme string
	Fg, Bg   color.Color
	Attr     Attr
	Width    Width
}

type Rect struct{ X, Y, W, H int }

type Run struct{ X, Y, Len int }

type Buffer struct {
	width, height int
	cells         []Cell
}

func space(c Cell) Cell {
	return Cell{Grapheme: " ", Fg: c.Fg, Bg: c.Bg, Attr: c.Attr}
}

func New(width, height int) *Buffer { return Filled(width, height, Cell{Grapheme: " "}) }

func Filled(width, height int, c Cell) *Buffer {
	b := &Buffer{width: width, height: height, cells: make([]Cell, width*height)}
	if c != (Cell{}) {
		b.Fill(Rect{W: width, H: height}, c)
	}
	return b
}

func (b *Buffer) Width() int  { return b.width }
func (b *Buffer) Height() int { return b.height }

func (b *Buffer) Row(y int) []Cell { return b.cells[y*b.width : (y+1)*b.width] }

func (b *Buffer) At(x, y int) Cell { return b.cells[y*b.width+x] }

func (b *Buffer) Set(x, y int, c Cell) {
	if x < 0 || y < 0 || x >= b.width || y >= b.height {
		return
	}
	row := b.Row(y)
	if c.Width == Narrow && row[x].Width == Narrow {
		put(&row[x], c)
		return
	}
	if c.Width == Continuation {
		panic("buffer: a continuation cell is written by its wide head, never set directly")
	}
	if c.Width == Wide && x == b.width-1 {
		c = space(c)
	}
	detach(row, x)
	if c.Width == Wide {
		detach(row, x+1)
		put(&row[x+1], Cell{Fg: c.Fg, Bg: c.Bg, Attr: c.Attr, Width: Continuation})
	}
	put(&row[x], c)
}

func put(dst *Cell, c Cell) {
	dst.Grapheme, dst.Fg, dst.Bg, dst.Attr, dst.Width = c.Grapheme, c.Fg, c.Bg, c.Attr, c.Width
}

func detach(row []Cell, x int) {
	switch row[x].Width {
	case Narrow:
	case Wide:
		put(&row[x+1], space(row[x+1]))
	case Continuation:
		put(&row[x-1], space(row[x-1]))
	default:
		panic("buffer: unknown cell width")
	}
}

func (b *Buffer) Fill(r Rect, c Cell) {
	lo, hi := max(r.X, 0), min(r.X+r.W, b.width)
	for y := max(r.Y, 0); y < min(r.Y+r.H, b.height) && lo < hi; y++ {
		if c.Width != Narrow {
			for x := lo; x < hi; x += 2 {
				b.Set(x, y, c)
			}
			continue
		}
		row := b.Row(y)
		detach(row, lo)
		detach(row, hi-1)
		for x := range row[lo:hi] {
			put(&row[lo+x], c)
		}
	}
}

func (b *Buffer) Resize(width, height int) {
	cells := make([]Cell, width*height)
	for i := range cells {
		cells[i].Grapheme = " "
	}
	for y := range min(height, b.height) {
		row := cells[y*width : (y+1)*width]
		copy(row, b.Row(y))
		if width > 0 && row[width-1].Width == Wide {
			row[width-1] = space(row[width-1])
		}
	}
	b.width, b.height, b.cells = width, height, cells
}

func same(a, b *Cell) bool {
	return a.Bg == b.Bg && a.Fg == b.Fg && a.Attr == b.Attr && a.Width == b.Width && a.Grapheme == b.Grapheme
}

func Diff(dst []Run, prev, cur *Buffer) []Run {
	if prev.width != cur.width || prev.height != cur.height {
		panic("buffer: diff of buffers with different sizes")
	}
	for y := range cur.height {
		dst = DiffRow(dst, prev, cur, y, 0, cur.width)
	}
	return dst
}

func DiffRow(dst []Run, prev, cur *Buffer, y, from, to int) []Run {
	p, c := prev.Row(y), cur.Row(y)
	for x := from; x < to; {
		if same(&p[x], &c[x]) {
			x++
			continue
		}
		start := x
		for x < len(c) && (!same(&p[x], &c[x]) || c[x].Width == Continuation) {
			x++
		}
		dst = append(dst, Run{X: start, Y: y, Len: x - start})
	}
	return dst
}
