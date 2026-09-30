package tailwind

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/css"
	"github.com/twind-dev/twind/twi/style"
)

type decls = []style.Declaration

const noRendering = "no terminal rendering"

func (c *compiler) declaration(prop string, raw []css.Token, v vars) (decls, problem) {
	switch prop {
	case "border-radius":
		if r, p, ok := themedRadius(raw); ok {
			return decls{{Property: style.PropRadius, Radius: r}}, p
		}
	case "box-shadow":
		return c.boxShadow(raw, v)
	case "background-image":
		return c.backgroundImage(raw, v)
	}
	var parts [][]css.Token
	for _, comp := range components(raw) {
		toks, err := c.resolve(comp, v, 0)
		switch {
		case errors.Is(err, deferred{}):
			parts = append(parts, nil)
		case err != nil:
			return nil, problem{Unsupported, err.Error()}
		default:
			parts = append(parts, components(toks)...)
		}
	}
	if slices.ContainsFunc(parts, func(p []css.Token) bool { return p == nil }) && prop != "translate" {
		if _, p := convert(prop, nil); p.reason == noRendering || p.category == Ignored {
			return nil, p
		}
		return nil, problem{}
	}
	out, p := convert(prop, parts)
	if p.reason != "" && p.category != Approximated {
		return nil, p
	}
	if token, mix, ok := themeToken(raw); ok && len(out) == 1 {
		out[0].Token, out[0].Mix = token, mix
	}
	return out, p
}

func convert(prop string, parts [][]css.Token) (decls, problem) {
	if family, order, ok := edges(prop); ok {
		return edge(family, order, parts)
	}
	if strings.HasPrefix(prop, "border-") && strings.HasSuffix(prop, "-style") {
		prop = "border-style"
	}
	switch prop {
	case "display":
		v, p := pick(parts, map[string]style.Display{"block": style.DisplayBlock, "flex": style.DisplayFlex, "grid": style.DisplayGrid, "inline": style.DisplayInline, "none": style.DisplayNone, "inline-block": style.DisplayBlock, "inline-flex": style.DisplayFlex, "inline-grid": style.DisplayGrid}, "inline-block", "inline-flex", "inline-grid")
		return decls{{Property: style.PropDisplay, Display: v}}, p
	case "flex-direction":
		v, p := pick(parts, map[string]style.Direction{"row": style.Row, "column": style.Column, "row-reverse": style.RowReverse, "column-reverse": style.ColumnReverse})
		return decls{{Property: style.PropDirection, Direction: v}}, p
	case "flex-wrap":
		v, p := pick(parts, map[string]style.Wrapping{"nowrap": style.NoWrap, "wrap": style.Wrap, "wrap-reverse": style.WrapReverse})
		return decls{{Property: style.PropWrap, Wrap: v}}, p
	case "flex-grow", "flex-shrink", "opacity", "z-index", "--tw-gradient-from-position", "--tw-gradient-via-position", "--tw-gradient-to-position":
		if len(parts) != 1 {
			return nil, problem{Unsupported, "expects one value"}
		}
		property := map[string]style.Property{"flex-grow": style.PropGrow, "flex-shrink": style.PropShrink, "opacity": style.PropOpacity, "z-index": style.PropZIndex, "--tw-gradient-from-position": style.PropGradientFromPosition, "--tw-gradient-via-position": style.PropGradientViaPosition, "--tw-gradient-to-position": style.PropGradientToPosition}[prop]
		if prop == "z-index" && strings.EqualFold(text(parts[0]), "auto") {
			return decls{{Property: property}}, problem{}
		}
		n, p := number(parts[0])
		return decls{{Property: property, Number: n}}, p
	case "flex":
		return flex(parts)
	case "flex-basis", "width", "height", "min-width", "max-width", "min-height", "max-height", "row-gap", "column-gap":
		if len(parts) != 1 {
			return nil, problem{Unsupported, "expects one value"}
		}
		property := map[string]style.Property{"flex-basis": style.PropBasis, "width": style.PropWidth, "height": style.PropHeight, "min-width": style.PropMinWidth, "max-width": style.PropMaxWidth, "min-height": style.PropMinHeight, "max-height": style.PropMaxHeight, "row-gap": style.PropRowGap, "column-gap": style.PropColumnGap}[prop]
		l, p := length(parts[0])
		return decls{{Property: property, Length: l}}, p
	case "gap":
		return edge([4]style.Property{style.PropRowGap, style.PropColumnGap}, []int{0, 1}, parts)
	case "align-items", "align-self", "justify-items", "justify-self":
		v, p := pick(parts, aligns())
		property := map[string]style.Property{"align-items": style.PropAlignItems, "align-self": style.PropAlignSelf, "justify-items": style.PropJustifyItems, "justify-self": style.PropJustifySelf}[prop]
		return decls{{Property: property, Align: v}}, p
	case "justify-content", "align-content":
		v, p := pick(parts, justifies())
		property := style.PropJustify
		if prop == "align-content" {
			property = style.PropAlignContent
		}
		return decls{{Property: property, Justify: v}}, p
	case "place-content", "place-items", "place-self":
		return place(prop, parts)
	case "grid-template-columns", "grid-template-rows", "grid-auto-columns", "grid-auto-rows":
		t, p := tracks(parts)
		property := map[string]style.Property{"grid-template-columns": style.PropGridColumns, "grid-template-rows": style.PropGridRows, "grid-auto-columns": style.PropGridAutoColumns, "grid-auto-rows": style.PropGridAutoRows}[prop]
		return decls{{Property: property, Tracks: t}}, p
	case "grid-column", "grid-row", "grid-column-start", "grid-column-end", "grid-row-start", "grid-row-end":
		return placement(prop, parts)
	case "grid-auto-flow":
		return flow(parts)
	case "position":
		v, p := pick(parts, map[string]style.Position{"static": style.PositionStatic, "relative": style.PositionRelative, "absolute": style.PositionAbsolute, "fixed": style.PositionFixed, "sticky": style.PositionSticky})
		return decls{{Property: style.PropPosition, Position: v}}, p
	case "overflow", "overflow-x", "overflow-y":
		names := map[string]style.Overflow{"visible": style.OverflowVisible, "hidden": style.OverflowHidden, "clip": style.OverflowHidden, "scroll": style.OverflowScroll, "auto": style.OverflowAuto}
		axes := map[string][]style.Property{"overflow": {style.PropOverflowX, style.PropOverflowY}, "overflow-x": {style.PropOverflowX}, "overflow-y": {style.PropOverflowY}}[prop]
		if len(parts) > len(axes) {
			return nil, problem{Unsupported, "too many values"}
		}
		var out decls
		for i, axis := range axes {
			v, p := pick(parts[i%len(parts):][:1], names)
			if p.reason != "" {
				return nil, p
			}
			out = append(out, style.Declaration{Property: axis, Overflow: v})
		}
		return out, problem{}
	case "translate":
		return translate(parts)
	case "color", "background-color", "border-color", "--tw-gradient-from", "--tw-gradient-via", "--tw-gradient-to", "--tw-shadow-color", "--tw-inset-shadow-color", "--tw-ring-color", "--tw-ring-offset-color":
		if len(parts) != 1 {
			return nil, problem{Unsupported, "one colour per box"}
		}
		v, p := paint(parts[0])
		property := map[string]style.Property{"color": style.PropColor, "background-color": style.PropBackground, "border-color": style.PropBorderColor, "--tw-gradient-from": style.PropGradientFrom, "--tw-gradient-via": style.PropGradientVia, "--tw-gradient-to": style.PropGradientTo, "--tw-shadow-color": style.PropShadowColor, "--tw-inset-shadow-color": style.PropInsetShadowColor, "--tw-ring-color": style.PropRingColor, "--tw-ring-offset-color": style.PropRingOffsetColor}[prop]
		return decls{{Property: property, Color: v}}, p
	case "--tw-ring-offset-width":
		if len(parts) != 1 {
			return nil, problem{Unsupported, "expects one value"}
		}
		px, p := pixels(parts[0])
		return decls{{Property: style.PropRingOffsetWidth, Number: float64(px)}}, p
	case "--tw-ring-inset":
		return decls{{Property: style.PropRingInset, Flag: strings.EqualFold(text(slices.Concat(parts...)), "inset")}}, problem{}
	case "white-space":
		v, p := pick(parts, map[string]style.WhiteSpace{"normal": style.WhiteSpaceNormal, "nowrap": style.WhiteSpaceNowrap, "pre": style.WhiteSpacePre, "pre-wrap": style.WhiteSpacePreWrap, "pre-line": style.WhiteSpacePreWrap, "break-spaces": style.WhiteSpacePreWrap}, "pre-line", "break-spaces")
		return decls{{Property: style.PropWhiteSpace, WhiteSpace: v}}, p
	case "text-overflow":
		v, p := pick(parts, map[string]style.TextOverflow{"clip": style.TextOverflowClip, "ellipsis": style.TextOverflowEllipsis})
		return decls{{Property: style.PropTextOverflow, TextOverflow: v}}, p
	case "aspect-ratio":
		if strings.EqualFold(text(slices.Concat(parts...)), "auto") {
			return decls{{Property: style.PropAspectRatio}}, problem{}
		}
		n, p := number(slices.Concat(parts...))
		return decls{{Property: style.PropAspectRatio, Number: n}}, p
	case "transition-property":
		return transitionProperty(parts)
	case "transition-duration", "transition-delay":
		d, p := duration(slices.Concat(parts...))
		property := map[string]style.Property{"transition-duration": style.PropTransitionDuration, "transition-delay": style.PropTransitionDelay}[prop]
		return decls{{Property: property, Duration: d}}, p
	case "transition-timing-function":
		e, p := easing(slices.Concat(parts...))
		return decls{{Property: style.PropTransitionEasing, Easing: e}}, p
	case "animation":
		return animation(parts)
	case "border-style":
		v, p := pick(parts, borderStyles(), "groove", "ridge", "inset", "outset")
		return decls{{Property: style.PropBorderStyle, BorderStyle: v}}, p
	case "border":
		return border(parts)
	case "border-radius":
		q, err := evaluate(slices.Concat(parts...))
		switch {
		case err != nil:
			return nil, problem{Unsupported, err.Error()}
		case math.IsInf(q.value, 1):
			return decls{{Property: style.PropRadius, Radius: style.RadiusFull}}, problem{}
		case q.value == 0:
			return decls{{Property: style.PropRadius, Radius: style.RadiusNone}}, problem{}
		}
		return decls{{Property: style.PropRadius, Radius: style.RadiusSm}}, problem{Approximated, "radius length drawn as sm"}
	case "font-weight":
		return fontWeight(parts)
	case "font-style":
		v, p := pick(parts, map[string]bool{"normal": false, "italic": true, "oblique": true})
		return decls{{Property: style.PropItalic, Flag: v}}, p
	case "text-decoration-line", "text-decoration":
		return decoration(parts, prop == "text-decoration")
	case "text-align":
		v, p := pick(parts, map[string]style.TextAlign{"left": style.TextLeft, "start": style.TextLeft, "center": style.TextCenter, "right": style.TextRight, "end": style.TextRight, "justify": style.TextJustify})
		return decls{{Property: style.PropTextAlign, TextAlign: v}}, p
	case "visibility":
		v, p := pick(parts, map[string]style.Visibility{"visible": style.Visible, "hidden": style.Hidden, "collapse": style.Hidden})
		return decls{{Property: style.PropVisibility, Visibility: v}}, p
	case "cursor":
		v, p := pick(parts, map[string]style.Cursor{"auto": style.CursorAuto, "default": style.CursorDefault, "pointer": style.CursorPointer, "text": style.CursorText, "move": style.CursorMove, "not-allowed": style.CursorNotAllowed, "wait": style.CursorWait, "help": style.CursorHelp, "crosshair": style.CursorCrosshair, "grab": style.CursorGrab, "grabbing": style.CursorGrabbing, "none": style.CursorNone})
		return decls{{Property: style.PropCursor, Cursor: v}}, p
	case "user-select":
		v, p := pick(parts, map[string]style.UserSelect{"auto": style.SelectAuto, "none": style.SelectNone, "text": style.SelectText, "all": style.SelectAll})
		return decls{{Property: style.PropUserSelect, UserSelect: v}}, p
	case "box-sizing", "font-family", "font-size", "line-height", "letter-spacing":
		return nil, problem{Ignored, "terminal cells have one size and one font"}
	}
	return nil, problem{Unsupported, noRendering}
}

func pick[T any](parts [][]css.Token, names map[string]T, approximated ...string) (T, problem) {
	var zero T
	if len(parts) != 1 {
		return zero, problem{Unsupported, "expects one value"}
	}
	word := strings.ToLower(text(parts[0]))
	v, ok := names[word]
	switch {
	case !ok:
		return zero, problem{Unsupported, "value " + strconv.Quote(word)}
	case slices.Contains(approximated, word):
		return v, problem{Approximated, "value " + strconv.Quote(word) + " drawn as its nearest terminal form"}
	}
	return v, problem{}
}

func edges(prop string) ([4]style.Property, []int, bool) {
	var family [4]style.Property
	var suffix string
	switch {
	case strings.HasPrefix(prop, "padding"):
		family, suffix = [4]style.Property{style.PropPaddingTop, style.PropPaddingRight, style.PropPaddingBottom, style.PropPaddingLeft}, prop[len("padding"):]
	case strings.HasPrefix(prop, "margin"):
		family, suffix = [4]style.Property{style.PropMarginTop, style.PropMarginRight, style.PropMarginBottom, style.PropMarginLeft}, prop[len("margin"):]
	case strings.HasPrefix(prop, "inset"):
		family, suffix = [4]style.Property{style.PropTop, style.PropRight, style.PropBottom, style.PropLeft}, prop[len("inset"):]
	case prop == "top" || prop == "right" || prop == "bottom" || prop == "left":
		family, suffix = [4]style.Property{style.PropTop, style.PropRight, style.PropBottom, style.PropLeft}, "-"+prop
	case strings.HasPrefix(prop, "border") && strings.HasSuffix(prop, "-width"):
		family, suffix = [4]style.Property{style.PropBorderTopWidth, style.PropBorderRightWidth, style.PropBorderBottomWidth, style.PropBorderLeftWidth}, strings.TrimSuffix(prop[len("border"):], "-width")
	default:
		return family, nil, false
	}
	order, ok := map[string][]int{"": {0, 1, 2, 3}, "-top": {0}, "-right": {1}, "-bottom": {2}, "-left": {3}, "-inline": {3, 1}, "-block": {0, 2}, "-inline-start": {3}, "-inline-end": {1}, "-block-start": {0}, "-block-end": {2}}[suffix]
	return family, order, ok
}

func edge(family [4]style.Property, order []int, parts [][]css.Token) (decls, problem) {
	if len(parts) == 0 || len(parts) > len(order) {
		return nil, problem{Unsupported, "expects one to " + strconv.Itoa(len(order)) + " values"}
	}
	var out decls
	var worst problem
	for i, side := range order {
		from := i % len(parts)
		if len(parts) == 3 && i == 3 {
			from = 1
		}
		l, p := length(parts[from])
		if p.reason != "" && p.category != Approximated {
			return nil, p
		}
		if p.reason != "" {
			worst = p
		}
		out = append(out, style.Declaration{Property: family[side], Length: l})
	}
	return out, worst
}

func flex(parts [][]css.Token) (decls, problem) {
	grow, shrink, basis := 1.0, 1.0, style.Length{Unit: style.Percent}
	switch strings.ToLower(text(slices.Concat(parts...))) {
	case "none":
		grow, shrink, basis = 0, 0, style.Length{Unit: style.Auto}
	case "auto":
		basis = style.Length{Unit: style.Auto}
	case "initial":
		grow, basis = 0, style.Length{Unit: style.Auto}
	default:
		var numbers []float64
		for _, part := range parts {
			if q, err := evaluate(part); err == nil && q.unit == "" && len(numbers) < 2 {
				numbers = append(numbers, q.value)
				continue
			}
			l, p := length(part)
			if p.reason != "" {
				return nil, p
			}
			basis = l
		}
		if len(numbers) > 0 {
			grow = numbers[0]
		}
		if len(numbers) > 1 {
			shrink = numbers[1]
		}
	}
	return decls{{Property: style.PropGrow, Number: grow}, {Property: style.PropShrink, Number: shrink}, {Property: style.PropBasis, Length: basis}}, problem{}
}

func translate(parts [][]css.Token) (decls, problem) {
	if len(parts) == 1 && strings.EqualFold(text(parts[0]), "none") {
		return decls{{Property: style.PropTranslateX}, {Property: style.PropTranslateY}}, problem{}
	}
	if len(parts) > 2 {
		return nil, problem{Unsupported, "translate on the z axis"}
	}
	var out decls
	for i, part := range parts {
		if part == nil {
			continue
		}
		l, p := length(part)
		if p.reason != "" {
			return nil, p
		}
		out = append(out, style.Declaration{Property: []style.Property{style.PropTranslateX, style.PropTranslateY}[i], Length: l})
	}
	return out, problem{}
}

func borderStyles() map[string]style.BorderStyle {
	return map[string]style.BorderStyle{"none": style.BorderNone, "hidden": style.BorderNone, "solid": style.BorderSingle, "dashed": style.BorderDashed, "dotted": style.BorderDotted, "double": style.BorderDouble, "groove": style.BorderSingle, "ridge": style.BorderSingle, "inset": style.BorderSingle, "outset": style.BorderSingle}
}

func border(parts [][]css.Token) (decls, problem) {
	width, borderStyle, paintColor := style.Length{Unit: style.Cells, Value: 1}, style.BorderNone, color.Color{Kind: color.Current}
	for _, part := range parts {
		if s, ok := borderStyles()[strings.ToLower(text(part))]; ok {
			borderStyle = s
			continue
		}
		if l, p := length(part); p.reason == "" {
			width = l
			continue
		}
		c, p := paint(part)
		if p.reason != "" {
			return nil, p
		}
		paintColor = c
	}
	out := decls{{Property: style.PropBorderStyle, BorderStyle: borderStyle}, {Property: style.PropBorderColor, Color: paintColor}}
	for _, side := range []style.Property{style.PropBorderTopWidth, style.PropBorderRightWidth, style.PropBorderBottomWidth, style.PropBorderLeftWidth} {
		out = append(out, style.Declaration{Property: side, Length: width})
	}
	return out, problem{}
}

func themedRadius(raw []css.Token) (style.Radius, problem, bool) {
	name, ok := strings.CutPrefix(text(raw), "var(--radius-")
	name, closed := strings.CutSuffix(name, ")")
	if !ok || !closed {
		return 0, problem{}, false
	}
	switch name {
	case "none":
		return style.RadiusNone, problem{}, true
	case "sm":
		return style.RadiusSm, problem{}, true
	case "md":
		return style.RadiusMd, problem{}, true
	case "lg":
		return style.RadiusLg, problem{}, true
	case "full":
		return style.RadiusFull, problem{}, true
	case "xs":
		return style.RadiusSm, problem{Approximated, "radius xs drawn as sm"}, true
	}
	return style.RadiusLg, problem{Approximated, "radius " + name + " drawn as lg"}, true
}

func fontWeight(parts [][]css.Token) (decls, problem) {
	weight := map[string]float64{"normal": konst.NormalWeight, "bold": konst.BoldWeight, "bolder": konst.BoldWeight}[strings.ToLower(text(slices.Concat(parts...)))]
	if weight == 0 {
		n, p := number(slices.Concat(parts...))
		if p.reason != "" {
			return nil, p
		}
		weight = n
	}
	bold := decls{{Property: style.PropBold, Flag: weight >= konst.BoldMinWeight}}
	if weight != konst.NormalWeight && weight != konst.BoldWeight {
		return bold, problem{Approximated, "weight " + strconv.FormatFloat(weight, 'g', -1, 64) + " drawn as bold or normal"}
	}
	return bold, problem{}
}

func decoration(parts [][]css.Token, shorthand bool) (decls, problem) {
	underline, strike := false, false
	var skipped problem
	for _, part := range parts {
		switch strings.ToLower(text(part)) {
		case "none":
		case "underline":
			underline = true
		case "line-through":
			strike = true
		default:
			if !shorthand {
				return nil, problem{Unsupported, "decoration " + strconv.Quote(text(part))}
			}
			skipped = problem{Approximated, "decoration style, colour and thickness are not drawn"}
		}
	}
	return decls{{Property: style.PropUnderline, Flag: underline}, {Property: style.PropStrikethrough, Flag: strike}}, skipped
}

func transitionProperty(parts [][]css.Token) (decls, problem) {
	fields := map[string]style.TransitionProperty{
		"all": style.TransitionAll, "none": 0, "color": style.TransitionColor, "background-color": style.TransitionBackground, "border-color": style.TransitionBorderColor,
		"--tw-gradient-from": style.TransitionGradient, "--tw-gradient-via": style.TransitionGradient, "--tw-gradient-to": style.TransitionGradient,
		"opacity": style.TransitionOpacity, "box-shadow": style.TransitionShadow, "transform": style.TransitionTranslate, "translate": style.TransitionTranslate,
		"outline-color": 0, "text-decoration-color": 0, "fill": 0, "stroke": 0, "scale": 0, "rotate": 0, "filter": 0, "-webkit-backdrop-filter": 0, "backdrop-filter": 0,
		"display": 0, "content-visibility": 0, "overlay": 0, "pointer-events": 0,
	}
	var set style.TransitionProperty
	var p problem
	for _, name := range commas(slices.Concat(parts...)) {
		word := strings.ToLower(text(name))
		field, known := fields[word]
		if !known {
			p = problem{Approximated, word + " is not animated"}
		}
		set |= field
	}
	return decls{{Property: style.PropTransitionProperty, Transition: set}}, p
}

func duration(toks []css.Token) (time.Duration, problem) {
	if strings.EqualFold(text(toks), "initial") {
		return 0, problem{}
	}
	q, err := evaluate(toks)
	scale, ok := map[string]time.Duration{"ms": time.Millisecond, "s": time.Second}[q.unit]
	if err != nil || !ok {
		return 0, problem{Unsupported, "duration " + strconv.Quote(text(toks))}
	}
	return time.Duration(math.Round(q.value * float64(scale))), problem{}
}

func easing(toks []css.Token) (style.Easing, problem) {
	toks = trim(toks)
	switch strings.ToLower(text(toks)) {
	case "linear":
		return style.Easing{X2: 1, Y2: 1}, problem{}
	case "ease", "initial":
		return cssEase(), problem{}
	}
	unsupported := problem{Unsupported, "timing function " + strconv.Quote(text(toks))}
	if len(toks) == 0 || toks[0].Kind != css.TokenFunction || !strings.EqualFold(toks[0].Text, "cubic-bezier(") {
		return style.Easing{}, unsupported
	}
	args := commas(toks[1:closing(toks, 0)])
	if len(args) != 4 {
		return style.Easing{}, unsupported
	}
	var v [4]float64
	for i, arg := range args {
		q, err := evaluate(arg)
		if err != nil || q.unit != "" {
			return style.Easing{}, unsupported
		}
		v[i] = q.value
	}
	return style.Easing{X1: v[0], Y1: v[1], X2: v[2], Y2: v[3]}, problem{}
}

func cssEase() style.Easing {
	return style.Easing{X1: konst.EaseX1, Y1: konst.EaseY1, X2: konst.EaseX2, Y2: konst.EaseY2}
}

func animation(parts [][]css.Token) (decls, problem) {
	keyframes, curve := style.KeyframesNone, cssEase()
	var length time.Duration
	iterations, infinite, timed := 1.0, false, false
	for _, part := range parts {
		word := strings.ToLower(text(part))
		named, isName := map[string]style.Keyframes{"none": style.KeyframesNone, "spin": style.KeyframesSpin, "ping": style.KeyframesPing, "pulse": style.KeyframesPulse, "bounce": style.KeyframesBounce}[word]
		d, dp := duration(part)
		e, ep := easing(part)
		n, np := number(part)
		switch {
		case isName:
			keyframes = named
		case word == "infinite":
			infinite = true
		case dp.reason == "" && !timed:
			length, timed = d, true
		case ep.reason == "":
			curve = e
		case np.reason == "":
			iterations = n
		default:
			return nil, problem{Unsupported, "animation value " + strconv.Quote(word)}
		}
	}
	return decls{
		{Property: style.PropAnimationKeyframes, Keyframes: keyframes},
		{Property: style.PropAnimationDuration, Duration: length},
		{Property: style.PropAnimationEasing, Easing: curve},
		{Property: style.PropAnimationIterations, Number: iterations, Flag: infinite},
	}, problem{}
}
