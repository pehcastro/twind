package motion

import (
	"math"
	"math/bits"
	"slices"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/motion"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

type Key uint32

type Pose struct{ Opacity, Scale, Turn, TranslateY float64 }

type property uint8

const (
	propColor property = iota
	propBackground
	propBorder
	propGradientFrom
	propGradientVia
	propGradientTo
	propOpacity
	propTranslateX
	propTranslateY
	propCount
)

type clock struct {
	start, duration time.Duration
	easing          Easing
	factor          float64
}

type track struct {
	clock
	from, to, goal, reverse [4]float64
}

type shadowTrack struct {
	clock
	from, to, reverse, out []style.Shadow
}

type entry struct {
	key       Key
	live      uint16
	animation style.Animation
	began     time.Duration
	tracks    [propCount]track
	shadows   [2]shadowTrack
}

type Styles struct {
	Reduced bool
	index   []int32
	entries []entry
	now     time.Duration
	srgb    srgb
}

type Animated struct {
	Pose    Pose
	moved   uint16
	values  [propCount][4]float64
	shadows [2][]style.Shadow
}

func (s *Styles) Overlay(a *Animated, st *style.ComputedStyle) {
	for m := a.moved; m != 0; m &= m - 1 {
		if b := bits.TrailingZeros16(m); b < int(propCount) {
			write(property(b), st, a.values[b], &s.srgb)
		} else {
			*shadows(st, b-int(propCount)) = a.shadows[b-int(propCount)]
		}
	}
}

func (s *Styles) Frame(key Key, prev, next *style.ComputedStyle, now time.Duration, a *Animated) bool {
	s.now = now
	a.moved, a.Pose = 0, Pose{Opacity: next.Opacity, Scale: 1}
	if int(key) >= len(s.index) {
		s.index = slices.Grow(s.index, int(key)+1-len(s.index))[:key+1]
	}
	if s.index[key] == 0 {
		if next.Animation.Keyframes == style.KeyframesNone && (prev == nil || same(prev, next)) {
			return false
		}
		s.entries = append(s.entries, entry{key: key})
		s.index[key] = int32(len(s.entries))
	}
	e := &s.entries[s.index[key]-1]
	t := next.Transition
	if on := !s.Reduced && t.Duration+t.Delay > 0; (prev != next || s.Reduced) && (!on || prev == nil || !same(prev, next)) {
		live := e.live
		e.live = 0
		for p := range propCount {
			if e.tracks[p].update(live&(1<<p) != 0, p, prev, next, now, on && t.Properties&p.flag() != 0, &s.srgb) {
				e.live |= 1 << p
			}
		}
		for k := range e.shadows {
			if e.shadows[k].update(live&(1<<(int(propCount)+k)) != 0, k, prev, next, now, on && t.Properties&style.TransitionShadow != 0, &s.srgb) {
				e.live |= 1 << (int(propCount) + k)
			}
		}
	}
	if e.animation.Keyframes != next.Animation.Keyframes {
		e.began = now
	}
	e.animation = next.Animation
	if !s.Reduced && e.animation.Keyframes != style.KeyframesNone {
		a.Pose = Keyframe(next.Animation, next.Opacity, now-e.began)
	}
	if a.Pose.Opacity != next.Opacity {
		a.values[propOpacity], a.moved = [4]float64{a.Pose.Opacity}, a.moved|1<<propOpacity
	}
	if ty := next.TranslateY; a.Pose.TranslateY != 0 && (ty.Unit == style.Percent || ty.Value == 0) {
		a.values[propTranslateY], a.moved = [4]float64{ty.Value + a.Pose.TranslateY, float64(style.Percent)}, a.moved|1<<propTranslateY
	}
	for m := e.live; m != 0; m &= m - 1 {
		b := bits.TrailingZeros16(m)
		c := e.clock(b)
		if now >= c.start+c.duration {
			e.live &^= 1 << b
			continue
		}
		a.moved |= 1 << b
		if b < int(propCount) {
			tr := &e.tracks[b]
			a.values[b] = lerp(tr.from, tr.to, c.eased(now))
		} else {
			a.shadows[b-int(propCount)] = e.shadows[b-int(propCount)].mix(c.eased(now), next.Color, &s.srgb)
		}
	}
	if e.live == 0 && e.animation.Keyframes == style.KeyframesNone {
		s.Drop(key)
	}
	return a.moved != 0
}

func (e *entry) clock(b int) *clock {
	if b < int(propCount) {
		return &e.tracks[b].clock
	}
	return &e.shadows[b-int(propCount)].clock
}

func (s *Styles) Drop(key Key) {
	if int(key) >= len(s.index) || s.index[key] == 0 {
		return
	}
	at, last := s.index[key], len(s.entries)
	s.entries[at-1] = s.entries[last-1]
	s.index[s.entries[at-1].key] = at
	s.entries = s.entries[:last-1]
	s.index[key] = 0
}

func (s *Styles) Holds(key Key) bool { return int(key) < len(s.index) && s.index[key] != 0 }

func (s *Styles) Wake() (time.Duration, bool) {
	at, moving := time.Duration(math.MaxInt64), false
	for i := range s.entries {
		e := &s.entries[i]
		a := e.animation
		if !s.Reduced && a.Keyframes != style.KeyframesNone && a.Duration > 0 && (a.Infinite || s.now-e.began < time.Duration(float64(a.Duration)*a.Iterations)) {
			return s.now, true
		}
		for m := e.live; m != 0; m &= m - 1 {
			at, moving = min(at, max(e.clock(bits.TrailingZeros16(m)).start, s.now)), true
		}
	}
	return at, moving
}

func (c *clock) begin(now time.Duration, t style.Transition, factor float64) {
	delay := t.Delay
	if delay < 0 {
		delay = time.Duration(float64(delay) * factor)
	}
	c.factor, c.start = factor, now+delay
	c.duration = time.Duration(float64(t.Duration) * factor)
	c.easing = CubicBezier(t.Easing.X1, t.Easing.Y1, t.Easing.X2, t.Easing.Y2)
}

func (c *clock) eased(now time.Duration) float64 {
	if now >= c.start+c.duration {
		return 1
	}
	if now <= c.start {
		return c.easing.at(0)
	}
	return c.easing.at(float64(now-c.start) / float64(c.duration))
}

func (c *clock) reversal(now time.Duration) float64 {
	return min(math.Abs(c.eased(now)*c.factor+1-c.factor), 1)
}

func (tr *track) update(live bool, p property, prev, next *style.ComputedStyle, now time.Duration, on bool, table *srgb) bool {
	goal, ok := read(p, next)
	if live && on && goal == tr.goal {
		return true
	}
	if !live {
		if prev == nil {
			return false
		}
		before, okBefore := read(p, prev)
		if !on || !ok || !okBefore || before == goal || !p.pairs(before, goal) {
			return false
		}
		tr.from, tr.to, tr.goal, tr.reverse = space(p, before), space(p, goal), goal, before
		tr.begin(now, next.Transition, 1)
		return true
	}
	if !on || !ok || !p.pairs(tr.goal, goal) {
		return false
	}
	current := lerp(tr.from, tr.to, tr.eased(now))
	factor, reverse := 1.0, unspace(p, current, table)
	if goal == tr.reverse {
		factor, reverse = tr.reversal(now), tr.goal
	}
	tr.from, tr.to, tr.goal, tr.reverse = current, space(p, goal), goal, reverse
	tr.begin(now, next.Transition, factor)
	return true
}

func (sh *shadowTrack) update(live bool, k int, prev, next *style.ComputedStyle, now time.Duration, on bool, table *srgb) bool {
	goal := *shadows(next, k)
	if live && on && slices.Equal(goal, sh.to) {
		return true
	}
	if !live {
		if prev == nil {
			return false
		}
		before := *shadows(prev, k)
		if !on || slices.Equal(before, goal) || !pairs(before, goal) {
			return false
		}
		sh.from = append(sh.from[:0], before...)
		sh.reverse = append(sh.reverse[:0], before...)
		sh.to = append(sh.to[:0], goal...)
		sh.begin(now, next.Transition, 1)
		return true
	}
	if !on || !pairs(sh.to, goal) {
		return false
	}
	factor := 1.0
	current := sh.mix(sh.eased(now), next.Color, table)
	if slices.Equal(goal, sh.reverse) {
		factor = sh.reversal(now)
		sh.reverse = append(sh.reverse[:0], sh.to...)
	} else {
		sh.reverse = append(sh.reverse[:0], current...)
	}
	sh.from = append(sh.from[:0], current...)
	sh.to = append(sh.to[:0], goal...)
	sh.begin(now, next.Transition, factor)
	return true
}

func (sh *shadowTrack) mix(eased float64, text color.Color, table *srgb) []style.Shadow {
	sh.out = sh.out[:0]
	for i := range max(len(sh.from), len(sh.to)) {
		a, b := padded(sh.from, sh.to, i), padded(sh.to, sh.from, i)
		px := func(a, b style.Pixels) style.Pixels {
			return style.Pixels(math.Round(float64(a) + float64(b-a)*eased))
		}
		b.X, b.Y, b.Blur, b.Spread = px(a.X, b.X), px(a.Y, b.Y), px(a.Blur, b.Blur), px(a.Spread, b.Spread)
		from, _ := channels(resolve(a.Color, text))
		to, _ := channels(resolve(b.Color, text))
		b.Color = color.Color{Kind: color.Literal, RGBA: table.rgba(lerp(premix(from), premix(to), eased))}
		sh.out = append(sh.out, b)
	}
	return sh.out
}

func padded(list, other []style.Shadow, i int) style.Shadow {
	if i < len(list) {
		return list[i]
	}
	return style.Shadow{Inset: other[i].Inset, Color: color.Color{Kind: color.Literal}}
}

func pairs(a, b []style.Shadow) bool {
	for i := range min(len(a), len(b)) {
		if a[i].Inset != b[i].Inset {
			return false
		}
	}
	return true
}

func shadows(st *style.ComputedStyle, k int) *[]style.Shadow {
	if k == 0 {
		return &st.Shadows
	}
	return &st.InsetShadows
}

func same(prev, next *style.ComputedStyle) bool {
	a, b := &prev.Gradient, &next.Gradient
	s, t := &prev.Transition, &next.Transition
	return s.Properties == t.Properties && s.Duration == t.Duration && s.Delay == t.Delay && s.Easing == t.Easing && prev.Color == next.Color && prev.Background == next.Background &&
		prev.BorderColor == next.BorderColor && a.From.Color == b.From.Color && a.Via.Color == b.Via.Color && a.To.Color == b.To.Color &&
		prev.Opacity == next.Opacity && prev.TranslateX == next.TranslateX && prev.TranslateY == next.TranslateY &&
		slices.Equal(prev.Shadows, next.Shadows) && slices.Equal(prev.InsetShadows, next.InsetShadows)
}

func (p property) flag() style.TransitionProperty {
	switch p {
	case propColor:
		return style.TransitionColor
	case propBackground:
		return style.TransitionBackground
	case propBorder:
		return style.TransitionBorderColor
	case propGradientFrom, propGradientVia, propGradientTo:
		return style.TransitionGradient
	case propOpacity:
		return style.TransitionOpacity
	case propTranslateX, propTranslateY:
		return style.TransitionTranslate
	}
	panic("motion: unknown property")
}

func (p property) colour(st *style.ComputedStyle) *color.Color {
	switch p {
	case propColor:
		return &st.Color
	case propBackground:
		return &st.Background
	case propBorder:
		return &st.BorderColor
	case propGradientFrom:
		return &st.Gradient.From.Color
	case propGradientVia:
		return &st.Gradient.Via.Color
	case propGradientTo:
		return &st.Gradient.To.Color
	case propOpacity, propTranslateX, propTranslateY:
		return nil
	}
	panic("motion: unknown property")
}

func (p property) paints() bool {
	return p.flag()&(style.TransitionOpacity|style.TransitionTranslate) == 0
}

func (p property) pairs(a, b [4]float64) bool {
	return p != propTranslateX && p != propTranslateY || a[1] == b[1]
}

func read(p property, st *style.ComputedStyle) ([4]float64, bool) {
	switch p {
	case propOpacity:
		return [4]float64{st.Opacity}, true
	case propTranslateX:
		return length(st.TranslateX)
	case propTranslateY:
		return length(st.TranslateY)
	case propColor:
		return channels(st.Color)
	}
	return channels(resolve(*p.colour(st), st.Color))
}

func write(p property, st *style.ComputedStyle, v [4]float64, table *srgb) {
	switch p {
	case propOpacity:
		st.Opacity = v[0]
	case propTranslateX:
		st.TranslateX = style.Length{Unit: style.Unit(v[1]), Value: v[0]}
	case propTranslateY:
		st.TranslateY = style.Length{Unit: style.Unit(v[1]), Value: v[0]}
	default:
		*p.colour(st) = color.Color{Kind: color.Literal, RGBA: table.rgba(v)}
	}
}

func space(p property, raw [4]float64) [4]float64 {
	if !p.paints() {
		return raw
	}
	return premix(raw)
}

func unspace(p property, v [4]float64, table *srgb) [4]float64 {
	if !p.paints() {
		return v
	}
	c, _ := channels(color.Color{Kind: color.Literal, RGBA: table.rgba(v)})
	return c
}

func length(l style.Length) ([4]float64, bool) {
	return [4]float64{l.Value, float64(l.Unit)}, l.Unit == style.Cells || l.Unit == style.Percent
}

func resolve(c, text color.Color) color.Color {
	switch c.Kind {
	case color.Current:
		return text
	case color.Unset:
		return color.Color{Kind: color.Literal}
	case color.Literal:
		return c
	}
	panic("motion: unknown colour kind")
}

func channels(c color.Color) ([4]float64, bool) {
	r := c.RGBA
	return [4]float64{float64(r.R), float64(r.G), float64(r.B), float64(r.A)}, c.Kind == color.Literal
}

func premix(raw [4]float64) [4]float64 {
	v := Color(color.RGBA{R: uint8(raw[0]), G: uint8(raw[1]), B: uint8(raw[2]), A: uint8(raw[3])})
	l, m, s := v.cone()
	a := v.ch[3]
	return [4]float64{l * a, m * a, s * a, a}
}

func lerp(a, b [4]float64, t float64) [4]float64 {
	for i := range a {
		a[i] += (b[i] - a[i]) * t
	}
	return a
}

func Keyframe(a style.Animation, opacity float64, elapsed time.Duration) Pose {
	pose := Pose{Opacity: opacity, Scale: 1}
	if a.Keyframes == style.KeyframesNone || a.Duration <= 0 || elapsed < 0 {
		return pose
	}
	cycles := float64(elapsed) / float64(a.Duration)
	if !a.Infinite && cycles >= a.Iterations {
		return pose
	}
	t := cycles - math.Floor(cycles)
	ease := CubicBezier(a.Easing.X1, a.Easing.Y1, a.Easing.X2, a.Easing.Y2)
	mid := func(v float64) float64 { return (t - v) / (1 - v) }
	switch a.Keyframes {
	case style.KeyframesSpin:
		pose.Turn = ease.at(t)
	case style.KeyframesPing:
		k := ease.at(min(t/konst.PingPeak, 1))
		pose.Scale, pose.Opacity = 1+(konst.PingScale-1)*k, opacity*(1-k)
	case style.KeyframesPulse:
		if t < konst.PulseMiddle {
			pose.Opacity = opacity + (konst.PulseOpacity-opacity)*ease.at(t/konst.PulseMiddle)
		} else {
			pose.Opacity = konst.PulseOpacity + (opacity-konst.PulseOpacity)*ease.at(mid(konst.PulseMiddle))
		}
	case style.KeyframesBounce:
		if t < konst.BounceMiddle {
			fall := CubicBezier(konst.BounceFallX1, konst.BounceFallY1, konst.BounceFallX2, konst.BounceFallY2)
			pose.TranslateY = konst.BounceLift * (1 - fall.at(t/konst.BounceMiddle))
		} else {
			rise := CubicBezier(konst.BounceRiseX1, konst.BounceRiseY1, konst.BounceRiseX2, konst.BounceRiseY2)
			pose.TranslateY = konst.BounceLift * rise.at(mid(konst.BounceMiddle))
		}
	case style.KeyframesNone:
	default:
		panic("motion: unknown keyframes")
	}
	return pose
}
