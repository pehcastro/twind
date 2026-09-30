package tailwind

import (
	"errors"
	"math"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/css"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

type deferred struct{}

func (deferred) Error() string { return "registered property left to the cascade" }

func (c *compiler) resolve(toks []css.Token, v vars, depth int) ([]css.Token, error) {
	if depth > konst.MaxVarDepth {
		return nil, errors.New("var() nested too deep")
	}
	var out []css.Token
	for i := 0; i < len(toks); i++ {
		if toks[i].Kind != css.TokenFunction || !strings.EqualFold(toks[i].Text, "var(") {
			out = append(out, toks[i])
			continue
		}
		end := closing(toks, i)
		args := toks[i+1 : end]
		i = end
		name, fallback, hasFallback := text(args), []css.Token(nil), false
		for j, t := range args {
			if t.Kind == css.TokenComma {
				name, fallback, hasFallback = text(args[:j]), args[j+1:], true
				break
			}
		}
		value, local := v.local[name]
		themed, inTheme := v.theme[name]
		hasInitial, registered := c.initial[name]
		switch {
		case local:
		case inTheme:
			value = themed
		case hasInitial:
			return nil, deferred{}
		case hasFallback:
			value = fallback
		case registered:
			return nil, deferred{}
		default:
			return nil, errors.New("undefined variable " + name)
		}
		resolved, err := c.resolve(value, v, depth+1)
		if err != nil {
			return nil, err
		}
		out = append(out, resolved...)
	}
	return out, nil
}

func closing(toks []css.Token, open int) int {
	depth := 0
	for i := open; i < len(toks); i++ {
		switch toks[i].Kind {
		case css.TokenFunction, css.TokenOpenParen:
			depth++
		case css.TokenCloseParen:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(toks)
}

func components(toks []css.Token) [][]css.Token {
	var out [][]css.Token
	start, depth := -1, 0
	for i, t := range toks {
		switch t.Kind {
		case css.TokenFunction, css.TokenOpenParen:
			depth++
		case css.TokenCloseParen:
			depth--
		}
		if t.Kind == css.TokenWhitespace && depth == 0 {
			if start >= 0 {
				out = append(out, toks[start:i])
			}
			start = -1
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, toks[start:])
	}
	return out
}

func commas(toks []css.Token) [][]css.Token {
	var out [][]css.Token
	start, depth := 0, 0
	for i, t := range toks {
		switch {
		case t.Kind == css.TokenFunction || t.Kind == css.TokenOpenParen:
			depth++
		case t.Kind == css.TokenCloseParen:
			depth--
		case t.Kind == css.TokenComma && depth == 0:
			out = append(out, toks[start:i])
			start = i + 1
		}
	}
	return append(out, toks[start:])
}

type quantity struct {
	value float64
	unit  string
}

type evaluator struct {
	toks []css.Token
	i    int
}

func evaluate(toks []css.Token) (quantity, error) {
	e := evaluator{toks: toks}
	q, err := e.sum()
	e.space()
	if err == nil && e.i != len(e.toks) {
		err = errors.New("unexpected " + strconv.Quote(e.toks[e.i].Text))
	}
	return q, err
}

func (e *evaluator) space() {
	for e.i < len(e.toks) && e.toks[e.i].Kind == css.TokenWhitespace {
		e.i++
	}
}

func (e *evaluator) operator(ops string) byte {
	e.space()
	if e.i < len(e.toks) && e.toks[e.i].Kind == css.TokenDelim && strings.Contains(ops, e.toks[e.i].Text) {
		e.i++
		return e.toks[e.i-1].Text[0]
	}
	return 0
}

func (e *evaluator) sum() (quantity, error) {
	left, err := e.product()
	for op := e.operator("+-"); err == nil && op != 0; op = e.operator("+-") {
		var right quantity
		if right, err = e.product(); err != nil {
			break
		}
		switch {
		case left.unit == right.unit:
		case left.value == 0 && left.unit == "":
			left.unit = right.unit
		case right.value != 0 || right.unit != "":
			return quantity{}, errors.New("calc() mixes " + left.unit + " and " + right.unit)
		}
		if op == '-' {
			right.value = -right.value
		}
		left.value += right.value
	}
	return left, err
}

func (e *evaluator) product() (quantity, error) {
	left, err := e.atom()
	for op := e.operator("*/"); err == nil && op != 0; op = e.operator("*/") {
		var right quantity
		if right, err = e.atom(); err != nil {
			break
		}
		switch {
		case op == '/' && right.unit == "":
			left.value /= right.value
		case op == '*' && right.unit == "":
			left.value *= right.value
		case op == '*' && left.unit == "":
			left = quantity{value: left.value * right.value, unit: right.unit}
		default:
			return quantity{}, errors.New("calc() cannot " + string(op) + " two lengths")
		}
	}
	return left, err
}

func (e *evaluator) atom() (quantity, error) {
	e.space()
	if e.i == len(e.toks) {
		return quantity{}, errors.New("missing value")
	}
	t := e.toks[e.i]
	e.i++
	switch t.Kind {
	case css.TokenNumber, css.TokenPercentage, css.TokenDimension:
		return dimension(t.Text)
	case css.TokenIdent:
		if strings.EqualFold(t.Text, "infinity") {
			return quantity{value: math.Inf(1)}, nil
		}
	case css.TokenFunction, css.TokenOpenParen:
		if t.Kind == css.TokenFunction && !strings.EqualFold(t.Text, "calc(") {
			break
		}
		q, err := e.sum()
		e.space()
		if err == nil && (e.i == len(e.toks) || e.toks[e.i].Kind != css.TokenCloseParen) {
			err = errors.New("calc() not closed")
		}
		e.i++
		return q, err
	}
	return quantity{}, errors.New("cannot compute " + strconv.Quote(t.Text))
}

func dimension(s string) (quantity, error) {
	end := strings.LastIndexAny(s, "0123456789.") + 1
	n, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return quantity{}, errors.New("bad number " + strconv.Quote(s))
	}
	return quantity{value: n, unit: strings.ToLower(s[end:])}, nil
}

func length(toks []css.Token) (style.Length, problem) {
	switch strings.ToLower(text(toks)) {
	case "auto":
		return style.Length{Unit: style.Auto}, problem{}
	case "none":
		return style.Length{Unit: style.None}, problem{}
	case "fit-content":
		return style.Length{Unit: style.FitContent}, problem{}
	}
	q, err := evaluate(toks)
	if err != nil {
		return style.Length{}, problem{Unsupported, err.Error()}
	}
	switch {
	case math.IsInf(q.value, 0):
		return style.Length{}, problem{Unsupported, "infinite length"}
	case q.unit == "%":
		return style.Length{Unit: style.Percent, Value: q.value}, problem{}
	case q.unit != "px" && (q.unit != "" || q.value != 0):
		return style.Length{}, problem{Unsupported, "unit " + strconv.Quote(q.unit) + " has no cell size in the preset"}
	case q.value != math.Round(q.value):
		return style.Length{Unit: style.Cells, Value: math.Round(q.value)}, problem{Approximated, "fractional cells rounded"}
	}
	return style.Length{Unit: style.Cells, Value: q.value}, problem{}
}

func pixels(toks []css.Token) (style.Pixels, problem) {
	q, err := evaluate(toks)
	switch {
	case err != nil:
		return 0, problem{Unsupported, err.Error()}
	case q.unit == "rem":
		q.value *= konst.RemPixels
	case q.unit != "px" && (q.unit != "" || q.value != 0):
		return 0, problem{Unsupported, "unit " + strconv.Quote(q.unit) + " is not pixels"}
	}
	if q.value != math.Round(q.value) {
		return style.Pixels(math.Round(q.value)), problem{Approximated, "fractional pixels rounded"}
	}
	return style.Pixels(q.value), problem{}
}

func number(toks []css.Token) (float64, problem) {
	q, err := evaluate(toks)
	switch {
	case err != nil:
		return 0, problem{Unsupported, err.Error()}
	case q.unit == "%":
		return q.value / konst.OpaquePercent, problem{}
	case q.unit != "":
		return 0, problem{Unsupported, "a number cannot have unit " + strconv.Quote(q.unit)}
	}
	return q.value, problem{}
}

func themeToken(raw []css.Token) (theme.Token, float64, bool) {
	toks, mix := trim(raw), float64(konst.OpaquePercent)
	if len(toks) > 0 && toks[0].Kind == css.TokenFunction && strings.EqualFold(toks[0].Text, "color-mix(") {
		args := commas(toks[1:closing(toks, 0)])
		if len(args) != 3 || !strings.EqualFold(text(args[2]), "transparent") {
			return 0, 0, false
		}
		mixed := components(args[1])
		if len(mixed) != 2 {
			return 0, 0, false
		}
		q, err := dimension(text(mixed[1]))
		if err != nil || q.unit != "%" {
			return 0, 0, false
		}
		toks, mix = mixed[0], q.value
	}
	name, _ := tailwindVar(toks)
	token, ok := theme.ParseToken(strings.TrimPrefix(name, "--color-"))
	return token, mix, ok
}

func paint(toks []css.Token) (color.Color, problem) {
	s := strings.ToLower(text(toks))
	if s == "inherit" {
		return color.Color{Kind: color.Current}, problem{}
	}
	if len(toks) == 0 || toks[0].Kind != css.TokenFunction || !strings.EqualFold(toks[0].Text, "color-mix(") {
		c, err := color.Parse(s)
		if err != nil {
			return c, problem{Unsupported, err.Error()}
		}
		return c, problem{}
	}
	args := commas(toks[1:closing(toks, 0)])
	if len(args) != 3 || !strings.EqualFold(text(args[2]), "transparent") {
		return color.Color{}, problem{Unsupported, "color-mix other than with transparent"}
	}
	mixed := strings.ToLower(text(args[1]))
	split := strings.LastIndex(mixed, " ")
	c, err := color.Parse(mixed[:max(split, 0)])
	amount, perr := strconv.ParseFloat(strings.TrimSuffix(mixed[split+1:], "%"), 64)
	switch {
	case err != nil || perr != nil:
		return color.Color{}, problem{Unsupported, "cannot read " + strconv.Quote(s)}
	case c.Kind == color.Current:
		return c, problem{Approximated, "currentcolor keeps its alpha"}
	}
	c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * amount / konst.OpaquePercent))
	return c, problem{}
}
