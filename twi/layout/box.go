package layout

import (
	"fmt"

	konst "github.com/twind-dev/twind/internal/konst/layout"
)

type Display uint8

const (
	DisplayFlex Display = iota
	DisplayNone
)

type Direction uint8

const (
	Row Direction = iota
	Column
)

type Justify uint8

const (
	JustifyStart Justify = iota
	JustifyEnd
	JustifyCenter
	JustifyBetween
	JustifyAround
	JustifyEvenly
)

type Align uint8

const (
	AlignAuto Align = iota
	AlignStart
	AlignEnd
	AlignCenter
	AlignStretch
)

type Position uint8

const (
	PositionStatic Position = iota
	PositionRelative
	PositionAbsolute
	PositionFixed
)

type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowHidden
	OverflowScroll
)

type Unit uint8

const (
	Auto Unit = iota
	Cells
	Percent
)

type Length struct {
	Unit  Unit
	Value int
}

type Edges struct {
	Top, Right, Bottom, Left int
}

type Insets struct {
	Top, Right, Bottom, Left Length
}

type Style struct {
	Display    Display
	Direction  Direction
	Justify    Justify
	AlignItems Align
	AlignSelf  Align
	Grow       int
	Shrink     int
	Basis      Length
	Width      Length
	Height     Length
	MinWidth   Length
	MinHeight  Length
	MaxWidth   Length
	MaxHeight  Length
	RowGap     int
	ColumnGap  int
	Padding    Edges
	Margin     Edges
	Border     Edges
	Position   Position
	Inset      Insets
	Overflow   Overflow
	ZIndex     int
}

type Measure func(availableWidth int) (width, height int)

type Rect struct {
	X, Y, W, H int
}

type Box struct {
	Style    Style
	Measure  Measure
	Children []*Box

	BorderBox  Rect
	PaddingBox Rect
	ContentBox Rect
	Clip       Rect

	ScrollX, ScrollY          int
	ScrollWidth, ScrollHeight int
}

func visible(b *Box) bool {
	switch b.Style.Display {
	case DisplayFlex:
		return true
	case DisplayNone:
		return false
	}
	panic(fmt.Sprintf("layout: unknown display %d", b.Style.Display))
}

func flowing(p Position) bool {
	switch p {
	case PositionStatic, PositionRelative:
		return true
	case PositionAbsolute, PositionFixed:
		return false
	}
	panic(fmt.Sprintf("layout: unknown position %d", p))
}

func clips(o Overflow) bool {
	switch o {
	case OverflowVisible:
		return false
	case OverflowHidden, OverflowScroll:
		return true
	}
	panic(fmt.Sprintf("layout: unknown overflow %d", o))
}

func intersect(a, b Rect) Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return Rect{x, y, max(min(a.X+a.W, b.X+b.W)-x, 0), max(min(a.Y+a.H, b.Y+b.H)-y, 0)}
}

func isRow(d Direction) bool {
	switch d {
	case Row:
		return true
	case Column:
		return false
	}
	panic(fmt.Sprintf("layout: unknown direction %d", d))
}

func alignOf(parent, child Style) Align {
	a := child.AlignSelf
	if a == AlignAuto {
		a = parent.AlignItems
	}
	if a == AlignAuto {
		return AlignStretch
	}
	return a
}

func offset(a Align, free int) int {
	switch a {
	case AlignStart, AlignStretch:
		return 0
	case AlignEnd:
		return free
	case AlignCenter:
		return free / 2
	}
	panic(fmt.Sprintf("layout: unknown align %d", a))
}

func resolve(l Length, base int, baseDefinite bool) (int, bool) {
	switch l.Unit {
	case Auto:
		return 0, false
	case Cells:
		return l.Value, true
	case Percent:
		return l.Value * base / konst.PercentWhole, baseDefinite
	}
	panic(fmt.Sprintf("layout: unknown unit %d", l.Unit))
}

func frame(s Style) (w, h int) {
	return s.Padding.Left + s.Padding.Right + s.Border.Left + s.Border.Right,
		s.Padding.Top + s.Padding.Bottom + s.Border.Top + s.Border.Bottom
}

func inset(r Rect, e Edges) Rect {
	return Rect{r.X + e.Left, r.Y + e.Top, max(r.W-e.Left-e.Right, 0), max(r.H-e.Top-e.Bottom, 0)}
}
