package tailwind

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi/css"
	"github.com/pehcastro/twind/twi/style"
)

func tracks(parts [][]css.Token) ([]style.Track, problem) {
	if len(parts) == 1 && strings.EqualFold(text(parts[0]), "none") {
		return nil, problem{}
	}
	var out []style.Track
	var worst problem
	for _, part := range parts {
		if called(part, "repeat(") {
			args := commas(part[1:closing(part, 0)])
			if len(args) != 2 {
				return nil, problem{Unsupported, "repeat() expects a count and a track list"}
			}
			n, err := strconv.Atoi(text(args[0]))
			if err != nil || n < 1 || n > konst.MaxGridTracks {
				return nil, problem{Unsupported, "repeat count " + strconv.Quote(text(args[0]))}
			}
			inner, p := tracks(components(trim(args[1])))
			if p.fatal() {
				return nil, p
			}
			for range n {
				out = append(out, inner...)
			}
			worst = cmp.Or(worst, p)
			continue
		}
		t, p := track(part)
		if p.fatal() {
			return nil, p
		}
		out, worst = append(out, t), cmp.Or(worst, p)
	}
	if len(out) > konst.MaxGridTracks {
		return nil, problem{Unsupported, "more than " + strconv.Itoa(konst.MaxGridTracks) + " tracks"}
	}
	return out, worst
}

func (p problem) fatal() bool { return p.reason != "" && p.category == Unsupported }

func called(toks []css.Token, name string) bool {
	return len(toks) > 0 && toks[0].Kind == css.TokenFunction && strings.EqualFold(toks[0].Text, name)
}

func track(toks []css.Token) (style.Track, problem) {
	if !called(toks, "minmax(") {
		b, p := breadth(toks)
		if b.Kind == style.SizeFr {
			return style.Track{Max: b}, p
		}
		return style.Track{Min: b, Max: b}, p
	}
	args := commas(toks[1:closing(toks, 0)])
	if len(args) != 2 {
		return style.Track{}, problem{Unsupported, "minmax() expects two values"}
	}
	lo, p := breadth(trim(args[0]))
	hi, q := breadth(trim(args[1]))
	switch {
	case p.fatal():
		return style.Track{}, p
	case q.fatal():
		return style.Track{}, q
	case lo.Kind == style.SizeFr:
		return style.Track{}, problem{Unsupported, "fr as a track minimum"}
	}
	return style.Track{Min: lo, Max: hi}, cmp.Or(p, q)
}

func breadth(toks []css.Token) (style.Breadth, problem) {
	switch strings.ToLower(text(toks)) {
	case "auto":
		return style.Breadth{}, problem{}
	case "min-content":
		return style.Breadth{Kind: style.SizeMinContent}, problem{}
	case "max-content":
		return style.Breadth{Kind: style.SizeMaxContent}, problem{}
	}
	if q, err := evaluate(toks); err == nil && q.unit == "fr" && q.value >= 0 {
		return style.Breadth{Kind: style.SizeFr, Value: q.value}, problem{}
	}
	l, p := length(toks)
	switch {
	case p.fatal():
		return style.Breadth{}, p
	case l.Unit == style.Cells:
		return style.Breadth{Kind: style.SizeCells, Value: l.Value}, p
	case l.Unit == style.Percent:
		return style.Breadth{Kind: style.SizePercent, Value: l.Value}, p
	}
	return style.Breadth{}, problem{Unsupported, "track size " + strconv.Quote(text(toks))}
}

func placement(prop string, parts [][]css.Token) (decls, problem) {
	props := map[string][]style.Property{
		"grid-column": {style.PropGridColumnStart, style.PropGridColumnEnd}, "grid-row": {style.PropGridRowStart, style.PropGridRowEnd},
		"grid-column-start": {style.PropGridColumnStart}, "grid-column-end": {style.PropGridColumnEnd},
		"grid-row-start": {style.PropGridRowStart}, "grid-row-end": {style.PropGridRowEnd},
	}[prop]
	var sides [][]css.Token
	start := 0
	flat := slices.Concat(parts...)
	for i, t := range flat {
		if t.Kind == css.TokenDelim && t.Text == "/" {
			sides, start = append(sides, flat[start:i]), i+1
		}
	}
	sides = append(sides, flat[start:])
	if len(sides) > len(props) {
		return nil, problem{Unsupported, "too many grid lines"}
	}
	var out decls
	for i, p := range props {
		var line style.GridLine
		if i < len(sides) {
			var ok bool
			if line, ok = gridLine(sides[i]); !ok {
				return nil, problem{Unsupported, "grid line " + strconv.Quote(text(sides[i]))}
			}
		}
		out = append(out, style.Declaration{Property: p, GridLine: line})
	}
	return out, problem{}
}

func gridLine(toks []css.Token) (style.GridLine, bool) {
	integer := func(t css.Token) int {
		n, err := strconv.Atoi(t.Text)
		if t.Kind != css.TokenNumber || err != nil {
			return 0
		}
		return n
	}
	switch {
	case len(toks) == 1 && toks[0].Kind == css.TokenIdent && strings.EqualFold(toks[0].Text, "auto"):
		return style.GridLine{}, true
	case len(toks) == 2 && toks[0].Kind == css.TokenIdent && strings.EqualFold(toks[0].Text, "span"):
		n := integer(toks[1])
		return style.GridLine{Span: n}, n > 0
	case len(toks) == 1:
		n := integer(toks[0])
		return style.GridLine{Line: n}, n != 0
	}
	return style.GridLine{}, false
}

func flow(parts [][]css.Token) (decls, problem) {
	words := make([]string, len(parts))
	for i, part := range parts {
		words[i] = strings.ToLower(text(part))
	}
	value := strings.Join(words, " ")
	v, ok := map[string]style.GridFlow{
		"row": style.FlowRow, "column": style.FlowColumn,
		"dense": style.FlowRowDense, "row dense": style.FlowRowDense, "dense row": style.FlowRowDense,
		"column dense": style.FlowColumnDense, "dense column": style.FlowColumnDense,
	}[value]
	if !ok {
		return nil, problem{Unsupported, "grid-auto-flow " + strconv.Quote(value)}
	}
	return decls{{Property: style.PropGridFlow, Flow: v}}, problem{}
}

func aligns() map[string]style.Align {
	return map[string]style.Align{"auto": style.AlignAuto, "normal": style.AlignStretch, "stretch": style.AlignStretch, "start": style.AlignStart, "flex-start": style.AlignStart, "self-start": style.AlignStart, "left": style.AlignStart, "end": style.AlignEnd, "flex-end": style.AlignEnd, "self-end": style.AlignEnd, "right": style.AlignEnd, "center": style.AlignCenter, "baseline": style.AlignBaseline}
}

func justifies() map[string]style.Justify {
	return map[string]style.Justify{"normal": style.JustifyStretch, "stretch": style.JustifyStretch, "start": style.JustifyStart, "flex-start": style.JustifyStart, "left": style.JustifyStart, "end": style.JustifyEnd, "flex-end": style.JustifyEnd, "right": style.JustifyEnd, "center": style.JustifyCenter, "space-between": style.JustifyBetween, "space-around": style.JustifyAround, "space-evenly": style.JustifyEvenly}
}

func place(prop string, parts [][]css.Token) (decls, problem) {
	if len(parts) == 0 || len(parts) > 2 {
		return nil, problem{Unsupported, "expects one or two values"}
	}
	first, second := parts[:1], parts[len(parts)-1:]
	if prop == "place-content" {
		a, p := pick(first, justifies())
		j, q := pick(second, justifies())
		return decls{{Property: style.PropAlignContent, Justify: a}, {Property: style.PropJustify, Justify: j}}, cmp.Or(p, q)
	}
	props := map[string][2]style.Property{"place-items": {style.PropAlignItems, style.PropJustifyItems}, "place-self": {style.PropAlignSelf, style.PropJustifySelf}}[prop]
	a, p := pick(first, aligns())
	j, q := pick(second, aligns())
	return decls{{Property: props[0], Align: a}, {Property: props[1], Align: j}}, cmp.Or(p, q)
}
