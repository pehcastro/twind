package style

import (
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

type Unit uint8

const (
	Cells Unit = iota
	Percent
	Auto
	None
	FitContent
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

type Wrapping uint8

const (
	NoWrap Wrapping = iota
	Wrap
	WrapReverse
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
	JustifyStretch
)

type TrackSize uint8

const (
	SizeAuto TrackSize = iota
	SizeCells
	SizePercent
	SizeFr
	SizeMinContent
	SizeMaxContent
)

type Breadth struct {
	Kind  TrackSize
	Value float64
}

type Track struct{ Min, Max Breadth }

type GridLine struct{ Line, Span int }

type GridPlacement struct{ Start, End GridLine }

type GridFlow uint8

const (
	FlowRow GridFlow = iota
	FlowColumn
	FlowRowDense
	FlowColumnDense
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

type Pixels int

type Shadow struct {
	X, Y, Blur, Spread Pixels
	Color              color.Color
	Inset              bool
	Tintable           bool
	Token              theme.Token
	Mix                float64
}

type Ring struct {
	Width, OffsetWidth Pixels
	Color, OffsetColor color.Color
	Inset              bool
}

type WhiteSpace uint8

const (
	WhiteSpaceNormal WhiteSpace = iota
	WhiteSpaceNowrap
	WhiteSpacePre
	WhiteSpacePreWrap
)

type TextOverflow uint8

const (
	TextOverflowClip TextOverflow = iota
	TextOverflowEllipsis
)

type OverflowWrap uint8

const (
	OverflowWrapNormal OverflowWrap = iota
	OverflowWrapBreakWord
	OverflowWrapAnywhere
)

type WordBreak uint8

const (
	WordBreakNormal WordBreak = iota
	WordBreakAll
	WordBreakKeepAll
)

type PointerEvents uint8

const (
	PointerAuto PointerEvents = iota
	PointerNone
)

type Element uint8

const (
	ElementAny Element = iota
	ElementA
	ElementButton
	ElementImg
	ElementInput
	ElementKbd
	ElementLabel
	ElementP
	ElementSelect
	ElementSpan
	ElementSVG
	ElementTextarea
)

type Relation uint8

const (
	RelationSelf Relation = iota
	RelationChild
	RelationDescendant
	RelationAncestor
	RelationPrevious
)

type Markers uint64

type Match struct {
	Relation Relation
	Class    string
	Element  Element
	States   State
	Attrs    []Attr
	mark     Markers
}

type Easing struct{ X1, Y1, X2, Y2 float64 }

type TransitionProperty uint8

const (
	TransitionColor TransitionProperty = 1 << iota
	TransitionBackground
	TransitionBorderColor
	TransitionGradient
	TransitionOpacity
	TransitionShadow
	TransitionTranslate
	TransitionAll = 1<<iota - 1
)

type Transition struct {
	Properties      TransitionProperty
	Duration, Delay time.Duration
	Easing          Easing
}

type Keyframes uint8

const (
	KeyframesNone Keyframes = iota
	KeyframesSpin
	KeyframesPing
	KeyframesPulse
	KeyframesBounce
	KeyframesEnter
	KeyframesExit
)

type Fill uint8

const (
	FillNone Fill = iota
	FillForwards
	FillBackwards
	FillBoth
)

type Pose struct {
	Opacity, Scale         float64
	TranslateX, TranslateY Length
	Degrees                float64
}

type Animation struct {
	Keyframes  Keyframes
	Duration   time.Duration
	Delay      time.Duration
	Easing     Easing
	Iterations float64
	Infinite   bool
	Fill       Fill
	Enter      Pose
	Exit       Pose
}

type GradientKind uint8

const (
	GradientNone GradientKind = iota
	GradientLinear
)

type GradientDirection uint8

const (
	ToBottom GradientDirection = iota
	ToTop
	ToRight
	ToLeft
	ToTopRight
	ToTopLeft
	ToBottomRight
	ToBottomLeft
	GradientAngle
)

type ColorSpace uint8

const (
	OKLab ColorSpace = iota
	SRGB
)

type GradientLine struct {
	Kind      GradientKind
	Direction GradientDirection
	Angle     float64
	Space     ColorSpace
}

type GradientStop struct {
	Color    color.Color
	Position float64
}

type Gradient struct {
	GradientLine
	From, Via, To GradientStop
	HasVia        bool
}

type ComputedStyle struct {
	Display    Display
	Direction  Direction
	Wrap       Wrapping
	Grow       float64
	Shrink     float64
	Basis      Length
	AlignItems Align
	AlignSelf  Align
	Justify    Justify
	RowGap     Length
	ColumnGap  Length

	GridColumns     []Track
	GridRows        []Track
	GridAutoColumns []Track
	GridAutoRows    []Track
	GridColumn      GridPlacement
	GridRow         GridPlacement
	GridFlow        GridFlow
	JustifyItems    Align
	JustifySelf     Align
	AlignContent    Justify

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
	ScaleX      float64
	ScaleY      float64
	BorderWidth Edges
	BorderStyle BorderStyle
	BorderColor color.Color
	Radius      Radius
	Background  color.Color
	Gradient    Gradient
	Opacity     float64

	Shadows          []Shadow
	InsetShadows     []Shadow
	ShadowColor      color.Color
	InsetShadowColor color.Color
	Ring             Ring

	Color         color.Color
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	TextAlign     TextAlign
	Visibility    Visibility
	Cursor        Cursor
	UserSelect    UserSelect
	WhiteSpace    WhiteSpace
	TextOverflow  TextOverflow
	OverflowWrap  OverflowWrap
	WordBreak     WordBreak
	PointerEvents PointerEvents
	AspectRatio   float64

	Transition Transition
	Animation  Animation
}
