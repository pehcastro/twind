package style

import (
	"fmt"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
)

type Property uint8

const (
	PropDisplay Property = iota + 1
	PropDirection
	PropGrow
	PropShrink
	PropBasis
	PropAlignItems
	PropAlignSelf
	PropJustify
	PropRowGap
	PropColumnGap
	PropWidth
	PropHeight
	PropMinWidth
	PropMaxWidth
	PropMinHeight
	PropMaxHeight
	PropPaddingTop
	PropPaddingRight
	PropPaddingBottom
	PropPaddingLeft
	PropMarginTop
	PropMarginRight
	PropMarginBottom
	PropMarginLeft
	PropPosition
	PropTop
	PropRight
	PropBottom
	PropLeft
	PropOverflowX
	PropOverflowY
	PropZIndex
	PropTranslateX
	PropTranslateY
	PropBorderTopWidth
	PropBorderRightWidth
	PropBorderBottomWidth
	PropBorderLeftWidth
	PropBorderStyle
	PropBorderColor
	PropRadius
	PropBackground
	PropOpacity
	PropColor
	PropBold
	PropItalic
	PropUnderline
	PropStrikethrough
	PropTextAlign
	PropVisibility
	PropCursor
	PropUserSelect
)

type Declaration struct {
	Property    Property
	Length      Length
	Number      float64
	Flag        bool
	Color       color.Color
	Display     Display
	Direction   Direction
	Align       Align
	Justify     Justify
	Position    Position
	Overflow    Overflow
	BorderStyle BorderStyle
	Radius      Radius
	TextAlign   TextAlign
	Visibility  Visibility
	Cursor      Cursor
	UserSelect  UserSelect
}

type State uint8

const (
	StateHover State = 1 << iota
	StateFocus
	StateFocusVisible
	StateActive
	StateDisabled
)

type Attr struct {
	Name     string
	Value    string
	AnyValue bool
}

type Condition struct {
	States    State
	Attrs     []Attr
	MinCols   int
	BelowCols int
}

type Rule struct {
	Class string
	When  Condition
	Decls []Declaration
}

type Sheet struct {
	rules     []Rule
	universal []int
	byClass   map[string][]int
}

type VersionError struct{ Got int }

func (e VersionError) Error() string {
	return fmt.Sprintf("style: IR version %d, runtime reads %d", e.Got, konst.IRVersion)
}

func NewSheet(version int, rules []Rule) (Sheet, error) {
	if version != konst.IRVersion {
		return Sheet{}, VersionError{Got: version}
	}
	s := Sheet{rules: rules, byClass: map[string][]int{}}
	for i, r := range rules {
		unconditional := r.When.States == 0 && len(r.When.Attrs) == 0 && r.When.MinCols == 0 && r.When.BelowCols == 0
		switch {
		case !unconditional:
		case r.Class == "":
			s.universal = append(s.universal, i)
		default:
			s.byClass[r.Class] = append(s.byClass[r.Class], i)
		}
	}
	return s, nil
}

func (s Sheet) Compute(parent ComputedStyle, classes []string) ComputedStyle {
	matched := slices.Clone(s.universal)
	for _, c := range classes {
		matched = append(matched, s.byClass[c]...)
	}
	slices.Sort(matched)
	out := ComputedStyle{
		Shrink:        1,
		AlignItems:    AlignStretch,
		Basis:         Length{Unit: Auto},
		Width:         Length{Unit: Auto},
		Height:        Length{Unit: Auto},
		MinWidth:      Length{Unit: Auto},
		MinHeight:     Length{Unit: Auto},
		MaxWidth:      Length{Unit: None},
		MaxHeight:     Length{Unit: None},
		Inset:         Edges{Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}},
		BorderColor:   color.Color{Kind: color.Current},
		Opacity:       1,
		Color:         parent.Color,
		Bold:          parent.Bold,
		Italic:        parent.Italic,
		Underline:     parent.Underline,
		Strikethrough: parent.Strikethrough,
		TextAlign:     parent.TextAlign,
		Visibility:    parent.Visibility,
		Cursor:        parent.Cursor,
		UserSelect:    parent.UserSelect,
	}
	for _, i := range slices.Compact(matched) {
		for _, d := range s.rules[i].Decls {
			out.apply(d, parent)
		}
	}
	return out
}

func (s *ComputedStyle) apply(d Declaration, parent ComputedStyle) {
	switch d.Property {
	case PropDisplay:
		s.Display = d.Display
	case PropDirection:
		s.Direction = d.Direction
	case PropGrow:
		s.Grow = d.Number
	case PropShrink:
		s.Shrink = d.Number
	case PropBasis:
		s.Basis = d.Length
	case PropAlignItems:
		s.AlignItems = d.Align
	case PropAlignSelf:
		s.AlignSelf = d.Align
	case PropJustify:
		s.Justify = d.Justify
	case PropRowGap:
		s.RowGap = d.Length
	case PropColumnGap:
		s.ColumnGap = d.Length
	case PropWidth:
		s.Width = d.Length
	case PropHeight:
		s.Height = d.Length
	case PropMinWidth:
		s.MinWidth = d.Length
	case PropMaxWidth:
		s.MaxWidth = d.Length
	case PropMinHeight:
		s.MinHeight = d.Length
	case PropMaxHeight:
		s.MaxHeight = d.Length
	case PropPaddingTop:
		s.Padding.Top = d.Length
	case PropPaddingRight:
		s.Padding.Right = d.Length
	case PropPaddingBottom:
		s.Padding.Bottom = d.Length
	case PropPaddingLeft:
		s.Padding.Left = d.Length
	case PropMarginTop:
		s.Margin.Top = d.Length
	case PropMarginRight:
		s.Margin.Right = d.Length
	case PropMarginBottom:
		s.Margin.Bottom = d.Length
	case PropMarginLeft:
		s.Margin.Left = d.Length
	case PropPosition:
		s.Position = d.Position
	case PropTop:
		s.Inset.Top = d.Length
	case PropRight:
		s.Inset.Right = d.Length
	case PropBottom:
		s.Inset.Bottom = d.Length
	case PropLeft:
		s.Inset.Left = d.Length
	case PropOverflowX:
		s.OverflowX = d.Overflow
	case PropOverflowY:
		s.OverflowY = d.Overflow
	case PropZIndex:
		s.ZIndex = int(d.Number)
	case PropTranslateX:
		s.TranslateX = d.Length
	case PropTranslateY:
		s.TranslateY = d.Length
	case PropBorderTopWidth:
		s.BorderWidth.Top = d.Length
	case PropBorderRightWidth:
		s.BorderWidth.Right = d.Length
	case PropBorderBottomWidth:
		s.BorderWidth.Bottom = d.Length
	case PropBorderLeftWidth:
		s.BorderWidth.Left = d.Length
	case PropBorderStyle:
		s.BorderStyle = d.BorderStyle
	case PropBorderColor:
		s.BorderColor = d.Color
	case PropRadius:
		s.Radius = d.Radius
	case PropBackground:
		s.Background = d.Color
	case PropOpacity:
		s.Opacity = d.Number
	case PropColor:
		s.Color = d.Color
		if d.Color.Kind == color.Current {
			s.Color = parent.Color
		}
	case PropBold:
		s.Bold = d.Flag
	case PropItalic:
		s.Italic = d.Flag
	case PropUnderline:
		s.Underline = d.Flag
	case PropStrikethrough:
		s.Strikethrough = d.Flag
	case PropTextAlign:
		s.TextAlign = d.TextAlign
	case PropVisibility:
		s.Visibility = d.Visibility
	case PropCursor:
		s.Cursor = d.Cursor
	case PropUserSelect:
		s.UserSelect = d.UserSelect
	default:
		panic(fmt.Sprintf("style: unknown property %d", d.Property))
	}
}
