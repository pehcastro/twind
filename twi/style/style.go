package style

import "github.com/twind-dev/twind/twi/color"

type Unit uint8

const (
	Cells Unit = iota
	Percent
	Auto
	None
)

type Length struct {
	Unit  Unit
	Value float64
}

type Edges struct{ Top, Right, Bottom, Left Length }

type Display uint8

const (
	DisplayBlock Display = iota
	DisplayFlex
	DisplayGrid
	DisplayInline
	DisplayNone
)

type Direction uint8

const (
	Row Direction = iota
	Column
	RowReverse
	ColumnReverse
)

type Align uint8

const (
	AlignAuto Align = iota
	AlignStretch
	AlignStart
	AlignEnd
	AlignCenter
	AlignBaseline
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
	OverflowAuto
)

type Visibility uint8

const (
	Visible Visibility = iota
	Hidden
)

type TextAlign uint8

const (
	TextLeft TextAlign = iota
	TextCenter
	TextRight
	TextJustify
)

type BorderStyle uint8

const (
	BorderNone BorderStyle = iota
	BorderSingle
	BorderDashed
	BorderDotted
	BorderDouble
)

type Radius uint8

const (
	RadiusNone Radius = iota
	RadiusSm
	RadiusMd
	RadiusLg
	RadiusFull
)

type Cursor uint8

const (
	CursorAuto Cursor = iota
	CursorDefault
	CursorPointer
	CursorText
	CursorMove
	CursorNotAllowed
	CursorWait
	CursorHelp
	CursorCrosshair
	CursorGrab
	CursorGrabbing
	CursorNone
)

type UserSelect uint8

const (
	SelectAuto UserSelect = iota
	SelectNone
	SelectText
	SelectAll
)

type ComputedStyle struct {
	Display    Display
	Direction  Direction
	Grow       float64
	Shrink     float64
	Basis      Length
	AlignItems Align
	AlignSelf  Align
	Justify    Justify
	RowGap     Length
	ColumnGap  Length

	Width     Length
	Height    Length
	MinWidth  Length
	MaxWidth  Length
	MinHeight Length
	MaxHeight Length

	Padding     Edges
	Margin      Edges
	Position    Position
	Inset       Edges
	OverflowX   Overflow
	OverflowY   Overflow
	ZIndex      int
	TranslateX  Length
	TranslateY  Length
	BorderWidth Edges
	BorderStyle BorderStyle
	BorderColor color.Color
	Radius      Radius
	Background  color.Color
	Opacity     float64

	Color         color.Color
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	TextAlign     TextAlign
	Visibility    Visibility
	Cursor        Cursor
	UserSelect    UserSelect
}
