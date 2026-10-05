package tailwind

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/css"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
)

func tailwindVar(layer []css.Token) (string, bool) {
	toks := trim(layer)
	if len(toks) == 0 || toks[0].Kind != css.TokenFunction || !strings.EqualFold(toks[0].Text, "var(") || closing(toks, 0) != len(toks)-1 {
		return "", false
	}
	name := text(commas(toks[1 : len(toks)-1])[0])
	return name, strings.HasPrefix(name, "--tw-")
}

func trim(toks []css.Token) []css.Token {
	for len(toks) > 0 && toks[0].Kind == css.TokenWhitespace {
		toks = toks[1:]
	}
	for len(toks) > 0 && toks[len(toks)-1].Kind == css.TokenWhitespace {
		toks = toks[:len(toks)-1]
	}
	return toks
}

func (c *compiler) boxShadow(raw []css.Token, v vars) (decls, problem) {
	layers := commas(raw)
	if !slices.ContainsFunc(layers, func(l []css.Token) bool { _, ok := tailwindVar(l); return ok }) {
		shadows, p := c.shadows(raw, v)
		if p.reason != "" && p.category != Approximated {
			return nil, p
		}
		return decls{{Property: style.PropShadow, Shadows: shadows}, {Property: style.PropInsetShadow}}, p
	}
	var out decls
	var worst problem
	for _, layer := range layers {
		name, ok := tailwindVar(layer)
		if !ok {
			return nil, problem{Unsupported, "box-shadow mixes Tailwind layers with literal shadows"}
		}
		value, set := v.local[name]
		if !set {
			continue
		}
		var layerDecls decls
		var p problem
		switch name {
		case "--tw-shadow", "--tw-inset-shadow":
			var shadows []style.Shadow
			shadows, p = c.shadows(value, v)
			layerDecls = decls{{Property: map[string]style.Property{"--tw-shadow": style.PropShadow, "--tw-inset-shadow": style.PropInsetShadow}[name], Shadows: shadows}}
		case "--tw-ring-shadow":
			layerDecls, p = c.ring(value, v)
		default:
			return nil, problem{Unsupported, strings.TrimPrefix(name, "--tw-") + " has no terminal rendering"}
		}
		if p.reason != "" && p.category != Approximated {
			return nil, p
		}
		if p.reason != "" {
			worst = p
		}
		out = append(out, layerDecls...)
	}
	if len(out) == 0 {
		return nil, problem{Unsupported, "box-shadow reads no shadow layer set in its rule"}
	}
	return out, worst
}

func (c *compiler) ring(value []css.Token, v vars) (decls, problem) {
	alone := vars{theme: v.theme, local: maps.Clone(v.local)}
	alone.local["--tw-ring-inset"] = nil
	alone.local["--tw-ring-offset-width"] = []css.Token{{Kind: css.TokenNumber, Text: "0"}}
	alone.local["--tw-ring-color"] = []css.Token{{Kind: css.TokenIdent, Text: "currentcolor"}}
	shadows, p := c.shadows(value, alone)
	switch {
	case p.reason != "" && p.category != Approximated:
		return nil, p
	case len(shadows) != 1:
		return nil, problem{Unsupported, "ring of " + strconv.Itoa(len(shadows)) + " shadows"}
	}
	out := decls{{Property: style.PropRingWidth, Number: float64(shadows[0].Spread)}}
	if shadows[0].Color.Kind != color.Current {
		out = append(out, style.Declaration{Property: style.PropRingColor, Color: shadows[0].Color})
	}
	return out, p
}

func (c *compiler) shadows(raw []css.Token, v vars) ([]style.Shadow, problem) {
	var out []style.Shadow
	var worst problem
	for _, written := range commas(raw) {
		tintable, token, mix := false, theme.Token(0), 0.0
		for _, part := range components(written) {
			if name, ok := tailwindVar(part); ok && strings.HasSuffix(name, "shadow-color") {
				tintable = true
				if args := commas(part[1:closing(part, 0)]); len(args) == 2 {
					part = args[1]
				}
			}
			if t, m, ok := themeToken(part); ok {
				token, mix = t, m
			}
		}
		toks, err := c.resolve(written, v, 0)
		switch {
		case err != nil:
			return nil, problem{Unsupported, err.Error()}
		case strings.EqualFold(text(toks), "none"):
			continue
		}
		for _, layer := range commas(toks) {
			s := style.Shadow{Color: color.Color{Kind: color.Current}, Tintable: tintable, Token: token, Mix: mix}
			var lengths []style.Pixels
			colored := false
			for _, part := range components(layer) {
				if strings.EqualFold(text(part), "inset") {
					s.Inset = true
					continue
				}
				if px, p := pixels(part); len(lengths) < 4 && (p.reason == "" || p.category == Approximated) {
					if p.reason != "" {
						worst = p
					}
					lengths = append(lengths, px)
					continue
				}
				paintColor, p := paint(part)
				if p.reason != "" || colored {
					return nil, problem{Unsupported, "shadow value " + strconv.Quote(text(part))}
				}
				s.Color, colored = paintColor, true
			}
			if len(lengths) < 2 {
				return nil, problem{Unsupported, "shadow needs an x and a y offset"}
			}
			lengths = append(lengths, 0, 0)
			s.X, s.Y, s.Blur, s.Spread = lengths[0], lengths[1], lengths[2], lengths[3]
			if s.Color.Kind == color.Literal && s.Color.RGBA.A == 0 {
				continue
			}
			out = append(out, s)
		}
	}
	return out, worst
}

func (c *compiler) backgroundImage(raw []css.Token, v vars) (decls, problem) {
	toks := trim(raw)
	if strings.EqualFold(text(toks), "none") {
		return decls{{Property: style.PropGradient}}, problem{}
	}
	if len(toks) == 0 || toks[0].Kind != css.TokenFunction {
		return nil, problem{Unsupported, "background image " + strconv.Quote(text(toks))}
	}
	if fn := strings.ToLower(toks[0].Text); fn != "linear-gradient(" {
		return nil, problem{Unsupported, strings.TrimSuffix(fn, "(") + " has no terminal rendering"}
	}
	if text(toks[1:len(toks)-1]) != "var(--tw-gradient-stops)" {
		return nil, problem{Unsupported, "gradient stops other than from-, via- and to- utilities"}
	}
	position, err := c.resolve(v.local["--tw-gradient-position"], v, 0)
	if err != nil {
		return nil, problem{Unsupported, err.Error()}
	}
	parts := components(position)
	words := make([]string, len(parts))
	for i, p := range parts {
		words[i] = strings.ToLower(text(p))
	}
	space := ""
	if in := slices.Index(words, "in"); in >= 0 {
		space, words, parts = strings.Join(words[in+1:], " "), words[:in], parts[:in]
	}
	direction, named := map[string]style.GradientDirection{
		"": style.ToBottom, "to bottom": style.ToBottom, "to top": style.ToTop, "to right": style.ToRight, "to left": style.ToLeft,
		"to top right": style.ToTopRight, "to right top": style.ToTopRight, "to top left": style.ToTopLeft, "to left top": style.ToTopLeft,
		"to bottom right": style.ToBottomRight, "to right bottom": style.ToBottomRight, "to bottom left": style.ToBottomLeft, "to left bottom": style.ToBottomLeft,
	}[strings.Join(words, " ")]
	line := style.GradientLine{Kind: style.GradientLinear, Direction: direction}
	if !named {
		q, err := evaluate(slices.Concat(parts...))
		if err != nil || q.unit != "deg" {
			return nil, problem{Unsupported, "gradient direction " + strconv.Quote(strings.Join(words, " "))}
		}
		line.Direction, line.Angle = style.GradientAngle, q.value
	}
	var p problem
	switch space {
	case "", "oklab":
	case "srgb":
		line.Space = style.SRGB
	default:
		p = problem{Approximated, "interpolation in " + space + " drawn in oklab"}
	}
	return decls{{Property: style.PropGradient, Line: line}}, p
}
