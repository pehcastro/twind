package layout

import (
	"fmt"

	konst "github.com/twind-dev/twind/internal/konst/layout"
)

type Display uint8

const (
	DisplayFlex Display = iota
	DisplayGrid
	DisplayNone
)

type Direction uint8

const (
	Row Direction = iota
	Column
)

type Justify uint8

const (
	JustifyStretch Justify = iota
	JustifyStart
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

type Wrapping uint8

const (
	NoWrap Wrapping = iota
	Wrap
	WrapReverse
)

type Ratio struct{ W, H int }

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
	Wrap       Wrapping
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
	Aspect     Ratio
	RowGap     int
	ColumnGap  int
	Padding    Edges
	Margin     Edges
	Border     Edges
	Position   Position
	Inset      Insets
	Overflow   Overflow
	ZIndex     int

	Columns      []Track
	Rows         []Track
	AutoColumns  []Track
	AutoRows     []Track
	Column       Placement
	Row          Placement
	Flow         Flow
	JustifyItems Align
	JustifySelf  Align
	AlignContent Justify
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

	memo  memo
	arena *arena
}

type memo struct {
	intrinsic           [2]int
	intrinsicKnown      [2]bool
	widthAvail, width   int
	widthKnown          bool
	heightWidth, height int
	heightKnown         bool
	frames              []Rect
	framesW, framesH    int
	framesKnown         bool
}

func (m *memo) keep(frames []Rect, w, h int) {
	m.frames, m.framesW, m.framesH, m.framesKnown = frames, w, h, true
}

func prepare(b *Box) {
	s := &b.Style
	switch {
	case s.Display > DisplayNone:
		panic(fmt.Sprintf("layout: unknown display %d", s.Display))
	case s.Direction > Column:
		panic(fmt.Sprintf("layout: unknown direction %d", s.Direction))
	case s.Position > PositionFixed:
		panic(fmt.Sprintf("layout: unknown position %d", s.Position))
	case s.Overflow > OverflowScroll:
		panic(fmt.Sprintf("layout: unknown overflow %d", s.Overflow))
	}
	if a := max(s.AlignItems, s.AlignSelf, s.JustifyItems, s.JustifySelf); a > AlignStretch {
		panic(fmt.Sprintf("layout: unknown align %d", a))
	}
	checkUnit(max(s.Basis.Unit, s.Width.Unit, s.Height.Unit, s.MinWidth.Unit, s.MinHeight.Unit, s.MaxWidth.Unit, s.MaxHeight.Unit,
		s.Inset.Top.Unit, s.Inset.Right.Unit, s.Inset.Bottom.Unit, s.Inset.Left.Unit))
	b.memo = memo{}
	for _, c := range b.Children {
		prepare(c)
	}
}

func checkUnit(u Unit) {
	if u > Percent {
		panic(fmt.Sprintf("layout: unknown unit %d", u))
	}
}

func visible(b *Box) bool {
	return b.Style.Display != DisplayNone
}

func flowing(p Position) bool {
	return p == PositionStatic || p == PositionRelative
}

func clips(o Overflow) bool {
	return o != OverflowVisible
}

func intersect(a, b Rect) Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return Rect{x, y, max(min(a.X+a.W, b.X+b.W)-x, 0), max(min(a.Y+a.H, b.Y+b.H)-y, 0)}
}

func isRow(d Direction) bool {
	return d == Row
}

func alignOf(parent, child *Style) Align {
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
	case AlignEnd:
		return free
	case AlignCenter:
		return free / 2
	}
	return 0
}

func resolve(l Length, base int, baseDefinite bool) (int, bool) {
	switch l.Unit {
	case Cells:
		return l.Value, true
	case Percent:
		return l.Value * base / konst.PercentWhole, baseDefinite
	}
	return 0, false
}

func frame(s *Style) (w, h int) {
	return s.Padding.Left + s.Padding.Right + s.Border.Left + s.Border.Right,
		s.Padding.Top + s.Padding.Bottom + s.Border.Top + s.Border.Bottom
}

func inset(r Rect, e Edges) Rect {
	return Rect{r.X + e.Left, r.Y + e.Top, max(r.W-e.Left-e.Right, 0), max(r.H-e.Top-e.Bottom, 0)}
}
