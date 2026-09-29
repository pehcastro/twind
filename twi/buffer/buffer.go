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

func New(width, height int) *Buffer {
	b := &Buffer{}
	b.Resize(width, height)
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
	if c.Width == Continuation {
		panic("buffer: a continuation cell is written by its wide head, never set directly")
	}
	if c.Width == Wide && x == b.width-1 {
		c = space(c)
	}
	row := b.Row(y)
	detach(row, x)
	if c.Width == Wide {
		detach(row, x+1)
		row[x+1] = Cell{Fg: c.Fg, Bg: c.Bg, Attr: c.Attr, Width: Continuation}
	}
	row[x] = c
}

func detach(row []Cell, x int) {
	switch row[x].Width {
	case Narrow:
	case Wide:
		row[x+1] = space(row[x+1])
	case Continuation:
		row[x-1] = space(row[x-1])
	default:
		panic("buffer: unknown cell width")
	}
}

func (b *Buffer) Fill(r Rect, c Cell) {
	step := 1
	if c.Width == Wide {
		step = 2
	}
	for y := max(r.Y, 0); y < min(r.Y+r.H, b.height); y++ {
		for x := max(r.X, 0); x < min(r.X+r.W, b.width); x += step {
			b.Set(x, y, c)
		}
	}
}

func (b *Buffer) Resize(width, height int) {
	cells := make([]Cell, width*height)
	for i := range cells {
		cells[i] = Cell{Grapheme: " "}
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

func Diff(dst []Run, prev, cur *Buffer) []Run {
	if prev.width != cur.width || prev.height != cur.height {
		panic("buffer: diff of buffers with different sizes")
	}
	for y := range cur.height {
		p, c := prev.Row(y), cur.Row(y)
		for x := 0; x < len(c); {
			if p[x] == c[x] {
				x++
				continue
			}
			start := x
			for x < len(c) && (p[x] != c[x] || c[x].Width == Continuation) {
				x++
			}
			dst = append(dst, Run{X: start, Y: y, Len: x - start})
		}
	}
	return dst
}
