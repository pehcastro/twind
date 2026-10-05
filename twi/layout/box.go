package layout

import (
	"fmt"
	"math"
	"slices"

	konst "github.com/pehcastro/twind/internal/konst/layout"
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
	PositionSticky
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
	Display      Display
	Direction    Direction
	Wrap         Wrapping
	Justify      Justify
	AlignItems   Align
	AlignSelf    Align
	Position     Position
	Overflow     Overflow
	Flow         Flow
	JustifyItems Align
	JustifySelf  Align
	AlignContent Justify
	Height       Length
	Padding      Edges
	Border       Edges
	Margin       Edges
	Grow         int
	Shrink       int
	Basis        Length
	Width        Length
	MinWidth     Length
	MinHeight    Length
	MaxWidth     Length
	MaxHeight    Length
	RowGap       int
	ColumnGap    int
	Aspect       Ratio
	Inset        Insets
	ZIndex       int
	RowUnits     int

	Columns     []Track
	Rows        []Track
	AutoColumns []Track
	AutoRows    []Track
	Column      Placement
	Row         Placement
}

func (s *Style) Equal(o *Style) bool {
	return s.Display == o.Display && s.Direction == o.Direction && s.Wrap == o.Wrap && s.Justify == o.Justify &&
		s.AlignItems == o.AlignItems && s.AlignSelf == o.AlignSelf && s.Position == o.Position && s.Overflow == o.Overflow &&
		s.Flow == o.Flow && s.JustifyItems == o.JustifyItems && s.JustifySelf == o.JustifySelf && s.AlignContent == o.AlignContent &&
		s.Height == o.Height && s.Padding == o.Padding && s.Border == o.Border && s.Margin == o.Margin &&
		s.Grow == o.Grow && s.Shrink == o.Shrink && s.Basis == o.Basis && s.Width == o.Width &&
		s.MinWidth == o.MinWidth && s.MinHeight == o.MinHeight && s.MaxWidth == o.MaxWidth && s.MaxHeight == o.MaxHeight &&
		s.RowGap == o.RowGap && s.ColumnGap == o.ColumnGap && s.Aspect == o.Aspect && s.Inset == o.Inset && s.ZIndex == o.ZIndex && s.RowUnits == o.RowUnits &&
		s.Column == o.Column && s.Row == o.Row &&
		slices.Equal(s.Columns, o.Columns) && slices.Equal(s.Rows, o.Rows) && slices.Equal(s.AutoColumns, o.AutoColumns) && slices.Equal(s.AutoRows, o.AutoRows)
}

type Measure func(availableWidth int) (width, height int)

type Rect struct {
	X, Y, W, H int
}

type Box struct {
	Measure                  Measure
	Children                 []*Box
	parent                   *Box
	current, prepared, stale bool
	adopted, plain           bool
	Moved                    bool
	frameW, frameH           int
	frame                    Rect

	BorderBox  Rect
	PaddingBox Rect
	ContentBox Rect
	Clip       Rect

	spot  spot
	memo  memo
	Style Style

	ScrollX, ScrollY          int
	ScrollWidth, ScrollHeight int

	arena *arena
}

type memo struct {
	intrinsic                            [2]int
	widthAvail, width                    int
	heightWidth, height                  int
	framesW, framesH                     int
	natural                              int
	intrinsicKnown                       [2]bool
	widthKnown, heightKnown, framesKnown bool
	framesMode                           heightMode
}

type placing uint8

const (
	placed placing = iota + 1
	hidden
)

type spot struct {
	mode     heightMode
	state    placing
	absolute container
}

func (b *Box) Invalidate() {
	b.current = false
	for p := b; p != nil && p.prepared; p = p.parent {
		p.prepared = false
	}
}

func ready(b *Box) {
	switch {
	case !b.current:
		refresh(b)
		b.current, b.prepared, b.stale, b.adopted = true, true, true, false
	case !b.prepared:
		prepare(b)
	}
}

func children(b *Box) []*Box {
	if b.adopted {
		return b.Children
	}
	for _, c := range b.Children {
		if c.parent != b {
			c.parent = b
		}
		ready(c)
	}
	b.adopted = true
	return b.Children
}

func refresh(b *Box) {
	s := &b.Style
	switch {
	case s.Display > DisplayNone:
		panic(fmt.Sprintf("layout: unknown display %d", s.Display))
	case s.Direction > Column:
		panic(fmt.Sprintf("layout: unknown direction %d", s.Direction))
	case s.Position > PositionSticky:
		panic(fmt.Sprintf("layout: unknown position %d", s.Position))
	case s.Overflow > OverflowScroll:
		panic(fmt.Sprintf("layout: unknown overflow %d", s.Overflow))
	}
	if a := max(s.AlignItems, s.AlignSelf, s.JustifyItems, s.JustifySelf); a > AlignStretch {
		panic(fmt.Sprintf("layout: unknown align %d", a))
	}
	sizes := s.Basis.Unit | s.Width.Unit | s.Height.Unit | s.MinWidth.Unit | s.MinHeight.Unit | s.MaxWidth.Unit | s.MaxHeight.Unit
	if sizes|s.Inset.Top.Unit|s.Inset.Right.Unit|s.Inset.Bottom.Unit|s.Inset.Left.Unit > Percent {
		checkUnit(max(s.Basis.Unit, s.Width.Unit, s.Height.Unit, s.MinWidth.Unit, s.MinHeight.Unit, s.MaxWidth.Unit, s.MaxHeight.Unit,
			s.Inset.Top.Unit, s.Inset.Right.Unit, s.Inset.Bottom.Unit, s.Inset.Left.Unit))
	}
	b.memo, b.plain = memo{}, sizes == Auto && s.Aspect == Ratio{}
	b.frameW = s.Padding.Left + s.Padding.Right + s.Border.Left + s.Border.Right
	b.frameH = s.Padding.Top + s.Padding.Bottom + s.Border.Top + s.Border.Bottom
}

func prepare(b *Box) bool {
	if b.prepared {
		return false
	}
	changed := !b.current
	if changed {
		refresh(b)
	}
	inner := changed
	for _, c := range b.Children {
		if c.parent != b {
			c.parent = b
		}
		if prepare(c) && !inner {
			inner, b.memo = true, memo{}
		}
	}
	b.current, b.prepared, b.stale, b.adopted = true, true, true, true
	s := &b.Style
	contained := s.Width.Unit == Cells && s.Height.Unit == Cells && clips(s.Overflow)
	return changed || inner && !contained
}

func (b *Box) height(base int, definite bool) (int, bool) {
	if b.plain {
		return 0, false
	}
	return resolve(b.Style.Height, base, definite)
}

func (b *Box) widthLimit(base int, definite bool) bounds {
	if b.plain {
		return bounds{max: math.MaxInt}
	}
	return limit(b.Style.MinWidth, b.Style.MaxWidth, base, definite)
}

func (b *Box) heightLimit(base int, definite bool) bounds {
	if b.plain {
		return bounds{max: math.MaxInt}
	}
	return limit(b.Style.MinHeight, b.Style.MaxHeight, base, definite)
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
	return p == PositionStatic || p == PositionRelative || p == PositionSticky
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

func (b *Box) nudge(y int) int {
	s := &b.Style
	if s.RowUnits <= 1 || len(b.Children) == 0 && b.Measure == nil {
		return 0
	}
	top := y + s.Border.Top + s.Padding.Top
	return whole(top+s.RowUnits-1, s.RowUnits) - top
}

func whole(v, step int) int {
	if step <= 1 {
		return v
	}
	q := v / step
	if v%step < 0 {
		q--
	}
	return q * step
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

func inset(r Rect, e Edges) Rect {
	return Rect{r.X + e.Left, r.Y + e.Top, max(r.W-e.Left-e.Right, 0), max(r.H-e.Top-e.Bottom, 0)}
}
