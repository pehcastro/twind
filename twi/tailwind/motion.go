package tailwind

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/css"
	"github.com/twind-dev/twind/twi/style"
)

func motion(prop string, parts [][]css.Token) (decls, problem, bool) {
	whole := slices.Concat(parts...)
	switch prop {
	case "transition-property":
		out, p := transitionProperty(parts)
		return out, p, true
	case "transition-duration", "transition-delay", "animation-duration", "animation-delay", "--tw-duration", "--tw-animation-duration", "--tw-animation-delay":
		d, p := duration(whole)
		property := map[string]style.Property{"transition-duration": style.PropTransitionDuration, "transition-delay": style.PropTransitionDelay, "animation-duration": style.PropAnimationDuration, "animation-delay": style.PropAnimationDelay, "--tw-duration": style.PropTailwindDuration, "--tw-animation-duration": style.PropTailwindAnimationDuration, "--tw-animation-delay": style.PropTailwindAnimationDelay}[prop]
		return decls{{Property: property, Duration: d}}, p, true
	case "transition-timing-function", "--tw-ease":
		e, p := easing(whole)
		property := map[string]style.Property{"transition-timing-function": style.PropTransitionEasing, "--tw-ease": style.PropTailwindEasing}[prop]
		return decls{{Property: property, Easing: e}}, p, true
	case "animation-iteration-count", "--tw-animation-iteration-count":
		d, p := iterations(whole)
		d.Property = map[string]style.Property{"animation-iteration-count": style.PropAnimationIterations, "--tw-animation-iteration-count": style.PropTailwindAnimationIterations}[prop]
		return decls{d}, p, true
	case "animation-fill-mode", "--tw-animation-fill-mode":
		f, p := pick(parts, fills())
		property := map[string]style.Property{"animation-fill-mode": style.PropAnimationFill, "--tw-animation-fill-mode": style.PropTailwindAnimationFill}[prop]
		return decls{{Property: property, Fill: f}}, p, true
	case "animation-direction":
		_, p := pick(parts, map[string]bool{"normal": true})
		return nil, p, true
	case "--tw-enter-opacity", "--tw-enter-scale", "--tw-exit-opacity", "--tw-exit-scale":
		n, p := number(whole)
		property := map[string]style.Property{"--tw-enter-opacity": style.PropEnterOpacity, "--tw-enter-scale": style.PropEnterScale, "--tw-exit-opacity": style.PropExitOpacity, "--tw-exit-scale": style.PropExitScale}[prop]
		return decls{{Property: property, Number: n}}, p, true
	case "--tw-enter-translate-x", "--tw-enter-translate-y", "--tw-exit-translate-x", "--tw-exit-translate-y":
		l, p := length(whole)
		property := map[string]style.Property{"--tw-enter-translate-x": style.PropEnterTranslateX, "--tw-enter-translate-y": style.PropEnterTranslateY, "--tw-exit-translate-x": style.PropExitTranslateX, "--tw-exit-translate-y": style.PropExitTranslateY}[prop]
		return decls{{Property: property, Length: l}}, p, true
	case "--tw-enter-rotate", "--tw-exit-rotate":
		q, err := evaluate(whole)
		if err != nil || q.unit != "deg" && (q.unit != "" || q.value != 0) {
			return nil, problem{Unsupported, "angle " + strconv.Quote(text(whole)) + " is not in degrees"}, true
		}
		property := map[string]style.Property{"--tw-enter-rotate": style.PropEnterDegrees, "--tw-exit-rotate": style.PropExitDegrees}[prop]
		return decls{{Property: property, Number: q.value}}, problem{}, true
	case "scale":
		out, p := scale(parts)
		return out, p, true
	}
	return nil, problem{}, false
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

func iterations(toks []css.Token) (style.Declaration, problem) {
	if strings.EqualFold(text(toks), "infinite") {
		return style.Declaration{Number: 1, Flag: true}, problem{}
	}
	n, p := number(toks)
	return style.Declaration{Number: n}, p
}

func fills() map[string]style.Fill {
	return map[string]style.Fill{"none": style.FillNone, "forwards": style.FillForwards, "backwards": style.FillBackwards, "both": style.FillBoth}
}

func (c *compiler) animation(raw []css.Token, v vars) (decls, problem) {
	out := decls{
		{Property: style.PropAnimationKeyframes},
		{Property: style.PropAnimationDuration},
		{Property: style.PropAnimationDelay},
		{Property: style.PropAnimationEasing, Easing: cssEase()},
		{Property: style.PropAnimationIterations, Number: 1},
		{Property: style.PropAnimationFill},
	}
	name, times, eased, count, fill := &out[0], out[1:3], &out[3], &out[4], &out[5]
	timed, filled := 0, false
	for _, item := range calls(raw) {
		fallback := false
		for variable, ok := tailwindVar(item); ok; variable, ok = tailwindVar(item) {
			item = trim(item)
			args := item[1 : len(item)-1]
			first := commas(args)[0]
			if _, set := v.local[variable]; set || len(first) == len(args) {
				break
			}
			item, fallback = args[len(first)+1:], true
		}
		toks, err := c.resolve(item, v, 0)
		if err != nil {
			return nil, problem{Unsupported, err.Error()}
		}
		for _, part := range components(toks) {
			word := strings.ToLower(text(part))
			d, dp := duration(part)
			e, ep := easing(part)
			n, np := iterations(part)
			f, isFill := fills()[word]
			keyframes, isName := map[string]style.Keyframes{"none": style.KeyframesNone, "spin": style.KeyframesSpin, "ping": style.KeyframesPing, "pulse": style.KeyframesPulse, "bounce": style.KeyframesBounce, "enter": style.KeyframesEnter, "exit": style.KeyframesExit}[word]
			switch {
			case dp.reason == "" && timed < len(times):
				times[timed].Duration, times[timed].Fallback = d, fallback
				timed++
			case ep.reason == "":
				eased.Easing, eased.Fallback = e, fallback
			case np.reason == "":
				count.Number, count.Flag, count.Fallback = n.Number, n.Flag, fallback
			case isFill && !filled:
				fill.Fill, fill.Fallback, filled = f, fallback, true
			case word == "normal" || word == "running":
			case isName:
				name.Keyframes = keyframes
			default:
				return nil, problem{Unsupported, "animation value " + strconv.Quote(word)}
			}
		}
	}
	return out, problem{}
}

func calls(toks []css.Token) [][]css.Token {
	var out [][]css.Token
	for _, part := range components(toks) {
		for len(part) > 0 {
			end := len(part)
			if part[0].Kind == css.TokenFunction {
				end = min(closing(part, 0)+1, end)
			}
			out, part = append(out, part[:end]), part[end:]
		}
	}
	return out
}

func scale(parts [][]css.Token) (decls, problem) {
	switch {
	case len(parts) == 1 && strings.EqualFold(text(parts[0]), "none"):
		return decls{{Property: style.PropScaleX, Number: 1}, {Property: style.PropScaleY, Number: 1}}, problem{}
	case len(parts) == 1:
		parts = append(parts, parts[0])
	case len(parts) > 2:
		return nil, problem{Unsupported, "scale on the z axis"}
	}
	var out decls
	for i, part := range parts {
		if part == nil {
			continue
		}
		n, p := number(part)
		if p.reason != "" {
			return nil, p
		}
		out = append(out, style.Declaration{Property: []style.Property{style.PropScaleX, style.PropScaleY}[i], Number: n})
	}
	return out, problem{}
}
