package style

import (
	"fmt"
	"math"
	"slices"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
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
	PropShadow
	PropInsetShadow
	PropGradient
	PropGradientFrom
	PropGradientVia
	PropGradientTo
	PropGradientFromPosition
	PropGradientViaPosition
	PropGradientToPosition
	PropShadowColor
	PropInsetShadowColor
	PropRingWidth
	PropRingColor
	PropRingInset
	PropRingOffsetWidth
	PropRingOffsetColor
	PropWhiteSpace
	PropTextOverflow
	PropAspectRatio
	PropTransitionProperty
	PropTransitionDuration
	PropTransitionDelay
	PropTransitionEasing
	PropAnimationKeyframes
	PropAnimationDuration
	PropAnimationEasing
	PropAnimationIterations
)

type Declaration struct {
	Property     Property
	Length       Length
	Number       float64
	Flag         bool
	Color        color.Color
	Token        theme.Token
	Mix          float64
	Display      Display
	Direction    Direction
	Align        Align
	Justify      Justify
	Position     Position
	Overflow     Overflow
	BorderStyle  BorderStyle
	Radius       Radius
	TextAlign    TextAlign
	Visibility   Visibility
	Cursor       Cursor
	UserSelect   UserSelect
	Shadows      []Shadow
	Line         GradientLine
	WhiteSpace   WhiteSpace
	TextOverflow TextOverflow
	Transition   TransitionProperty
	Keyframes    Keyframes
	Duration     time.Duration
	Easing       Easing
}

type State uint8

const (
	StateHover State = 1 << iota
	StateFocus
	StateFocusVisible
	StateActive
	StateDisabled
	StateFocusWithin
	StateChecked
)

type NodeState struct {
	States State
	Attrs  []Attr
}

type Attr struct {
	Name     string
	Value    string
	AnyValue bool
}

type Scheme uint8

const (
	SchemeAny Scheme = iota
	SchemeLight
	SchemeDark
)

type Condition struct {
	States    State
	Attrs     []Attr
	MinCols   int
	BelowCols int
	Scheme    Scheme
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
	theme     *theme.Theme
}

func (s Sheet) WithTheme(t *theme.Theme) Sheet {
	s.theme = t
	return s
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
		switch {
		case r.When.MinCols != 0 || r.When.BelowCols != 0:
		case r.Class == "":
			s.universal = append(s.universal, i)
		default:
			s.byClass[r.Class] = append(s.byClass[r.Class], i)
		}
	}
	return s, nil
}

func (n NodeState) matches(when *Condition) bool {
	if when.States&^n.States != 0 {
		return false
	}
	for _, want := range when.Attrs {
		if !slices.ContainsFunc(n.Attrs, func(a Attr) bool { return a.Name == want.Name && (want.AnyValue || a.Value == want.Value) }) {
			return false
		}
	}
	return true
}

func (s Sheet) themed(c color.Color, token theme.Token, mix float64) color.Color {
	if token == 0 || s.theme == nil || s.theme.Tokens[token].Kind == color.Unset {
		return c
	}
	c = s.theme.Tokens[token]
	c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * mix / konst.OpaquePercent))
	return c
}

func (s Sheet) Compute(parent ComputedStyle, classes []string) ComputedStyle {
	return s.ComputeState(parent, classes, NodeState{})
}

func (s Sheet) ComputeState(parent ComputedStyle, classes []string, node NodeState) ComputedStyle {
	var stack [konst.MatchedRules]int
	matched := append(stack[:0], s.universal...)
	for _, c := range classes {
		matched = append(matched, s.byClass[c]...)
	}
	slices.Sort(matched)
	ease := Easing{X1: konst.EaseX1, Y1: konst.EaseY1, X2: konst.EaseX2, Y2: konst.EaseY2}
	out := ComputedStyle{
		Shrink:      1,
		AlignItems:  AlignStretch,
		Basis:       Length{Unit: Auto},
		Width:       Length{Unit: Auto},
		Height:      Length{Unit: Auto},
		MinWidth:    Length{Unit: Auto},
		MinHeight:   Length{Unit: Auto},
		MaxWidth:    Length{Unit: None},
		MaxHeight:   Length{Unit: None},
		Inset:       Edges{Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}},
		BorderColor: color.Color{Kind: color.Current},
		Opacity:     1,
		Gradient: Gradient{
			From: GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.FromPosition},
			Via:  GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.ViaPosition},
			To:   GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.ToPosition},
		},
		Color:         parent.Color,
		Bold:          parent.Bold,
		Italic:        parent.Italic,
		Underline:     parent.Underline,
		Strikethrough: parent.Strikethrough,
		TextAlign:     parent.TextAlign,
		Visibility:    parent.Visibility,
		Cursor:        parent.Cursor,
		UserSelect:    parent.UserSelect,
		WhiteSpace:    parent.WhiteSpace,
		Ring: Ring{
			Color:       color.Color{Kind: color.Current},
			OffsetColor: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: konst.RingOffsetWhite, G: konst.RingOffsetWhite, B: konst.RingOffsetWhite, A: konst.RingOffsetWhite}},
		},
		Transition: Transition{Properties: TransitionAll, Easing: ease},
		Animation:  Animation{Iterations: 1, Easing: ease},
	}
	scheme := SchemeAny
	switch {
	case s.theme == nil:
	case s.theme.Scheme == theme.Dark:
		scheme = SchemeDark
	default:
		scheme = SchemeLight
	}
	for _, i := range slices.Compact(matched) {
		r := &s.rules[i]
		if r.When.Scheme != SchemeAny && r.When.Scheme != scheme || !node.matches(&r.When) {
			continue
		}
		for j := range r.Decls {
			d := &r.Decls[j]
			if d.Token != 0 && s.theme != nil {
				themed := *d
				themed.Color = s.themed(d.Color, d.Token, d.Mix)
				d = &themed
			}
			out.apply(d, parent.Color)
		}
	}
	out.Shadows = s.shade(out.Shadows, out.ShadowColor)
	out.InsetShadows = s.shade(out.InsetShadows, out.InsetShadowColor)
	if r := out.Ring; r.Width > 0 {
		var ring []Shadow
		if r.OffsetWidth > 0 {
			ring = append(ring, Shadow{Spread: r.OffsetWidth, Color: r.OffsetColor, Inset: r.Inset})
		}
		ring = append(ring, Shadow{Spread: r.Width + r.OffsetWidth, Color: r.Color, Inset: r.Inset})
		if r.Inset {
			out.InsetShadows = slices.Concat(out.InsetShadows, ring)
		} else {
			out.Shadows = slices.Concat(ring, out.Shadows)
		}
	}
	return out
}

func (s Sheet) shade(shadows []Shadow, tint color.Color) []Shadow {
	if tint.Kind == color.Unset && (s.theme == nil || !slices.ContainsFunc(shadows, func(sh Shadow) bool { return sh.Token != 0 })) {
		return shadows
	}
	var out []Shadow
	for _, sh := range shadows {
		sh.Color = s.themed(sh.Color, sh.Token, sh.Mix)
		if sh.Tintable && tint.Kind != color.Unset {
			sh.Color = tint
		}
		if sh.Color.Kind != color.Literal || sh.Color.RGBA.A > 0 {
			out = append(out, sh)
		}
	}
	return out
}

func (s *ComputedStyle) apply(d *Declaration, inherited color.Color) {
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
			s.Color = inherited
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
	case PropShadow:
		s.Shadows = d.Shadows
	case PropInsetShadow:
		s.InsetShadows = d.Shadows
	case PropShadowColor:
		s.ShadowColor = d.Color
	case PropInsetShadowColor:
		s.InsetShadowColor = d.Color
	case PropGradient:
		s.Gradient.GradientLine = d.Line
	case PropGradientFrom:
		s.Gradient.From.Color = d.Color
	case PropGradientVia:
		s.Gradient.Via.Color = d.Color
		s.Gradient.HasVia = true
	case PropGradientTo:
		s.Gradient.To.Color = d.Color
	case PropGradientFromPosition:
		s.Gradient.From.Position = d.Number
	case PropGradientViaPosition:
		s.Gradient.Via.Position = d.Number
	case PropGradientToPosition:
		s.Gradient.To.Position = d.Number
	case PropRingWidth:
		s.Ring.Width = Pixels(d.Number)
	case PropRingColor:
		s.Ring.Color = d.Color
	case PropRingInset:
		s.Ring.Inset = d.Flag
	case PropRingOffsetWidth:
		s.Ring.OffsetWidth = Pixels(d.Number)
	case PropRingOffsetColor:
		s.Ring.OffsetColor = d.Color
	case PropWhiteSpace:
		s.WhiteSpace = d.WhiteSpace
	case PropTextOverflow:
		s.TextOverflow = d.TextOverflow
	case PropAspectRatio:
		s.AspectRatio = d.Number
	case PropTransitionProperty:
		s.Transition.Properties = d.Transition
	case PropTransitionDuration:
		s.Transition.Duration = d.Duration
	case PropTransitionDelay:
		s.Transition.Delay = d.Duration
	case PropTransitionEasing:
		s.Transition.Easing = d.Easing
	case PropAnimationKeyframes:
		s.Animation.Keyframes = d.Keyframes
	case PropAnimationDuration:
		s.Animation.Duration = d.Duration
	case PropAnimationEasing:
		s.Animation.Easing = d.Easing
	case PropAnimationIterations:
		s.Animation.Iterations, s.Animation.Infinite = d.Number, d.Flag
	default:
		panic(fmt.Sprintf("style: unknown property %d", d.Property))
	}
}
