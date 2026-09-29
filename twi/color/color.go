package color

import (
	"math"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/color"
)

type Kind uint8

const (
	Unset Kind = iota
	Literal
	Current
)

type RGBA struct{ R, G, B, A uint8 }

type Color struct {
	Kind Kind
	RGBA RGBA
}

type Profile uint8

const (
	None Profile = iota
	ANSI16
	ANSI256
	TrueColor
)

type SyntaxError struct{ Input string }

func (e SyntaxError) Error() string {
	return "color: cannot parse " + strconv.Quote(e.Input)
}

func Parse(input string) (Color, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "transparent" {
		return Color{Kind: Literal}, nil
	}
	if s == "currentcolor" {
		return Color{Kind: Current}, nil
	}
	var c RGBA
	var ok bool
	if hex, isHex := strings.CutPrefix(s, "#"); isHex {
		c, ok = parseHex(hex)
	} else if body, isOKLCH := strings.CutPrefix(s, "oklch("); isOKLCH {
		c, ok = parseOKLCH(body)
	}
	if !ok {
		return Color{}, SyntaxError{Input: input}
	}
	return Color{Kind: Literal, RGBA: c}, nil
}

func parseHex(hex string) (RGBA, bool) {
	if len(hex) == 3 || len(hex) == 4 {
		long := make([]byte, 0, 2*len(hex))
		for i := range len(hex) {
			long = append(long, hex[i], hex[i])
		}
		hex = string(long)
	}
	if len(hex) == 6 {
		hex += "ff"
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || len(hex) != 8 {
		return RGBA{}, false
	}
	return RGBA{uint8(n >> 24), uint8(n >> 16), uint8(n >> 8), uint8(n)}, true
}

func parseOKLCH(body string) (RGBA, bool) {
	body, closed := strings.CutSuffix(body, ")")
	channels, alphaText, hasAlpha := strings.Cut(body, "/")
	fields := strings.Fields(channels)
	if !closed || len(fields) != 3 {
		return RGBA{}, false
	}
	l, okL := fraction(fields[0])
	chroma, okC := number(fields[1])
	hue, okH := number(strings.TrimSuffix(fields[2], "deg"))
	alpha, okA := 1.0, true
	if hasAlpha {
		alpha, okA = fraction(strings.TrimSpace(alphaText))
	}
	if !okL || !okC || !okH || !okA {
		return RGBA{}, false
	}
	l = min(max(l, 0), 1)
	chroma = max(chroma, 0)
	a := chroma * math.Cos(hue*math.Pi/180)
	b := chroma * math.Sin(hue*math.Pi/180)
	lr := l + 0.3963377774*a + 0.2158037573*b
	mr := l - 0.1055613458*a - 0.0638541728*b
	sr := l - 0.0894841775*a - 1.2914855480*b
	lc, mc, sc := lr*lr*lr, mr*mr*mr, sr*sr*sr
	return RGBA{
		R: gamma(4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc),
		G: gamma(-1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc),
		B: gamma(-0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc),
		A: channel(alpha),
	}, true
}

func number(s string) (float64, bool) {
	if s == "none" {
		return 0, true
	}
	n, err := strconv.ParseFloat(s, 64)
	return n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
}

func fraction(s string) (float64, bool) {
	if pct, isPct := strings.CutSuffix(s, "%"); isPct {
		n, ok := number(pct)
		return n / 100, ok
	}
	return number(s)
}

func gamma(linear float64) uint8 {
	if linear <= 0.0031308 {
		return channel(12.92 * linear)
	}
	return channel(1.055*math.Pow(linear, 1/2.4) - 0.055)
}

func channel(v float64) uint8 {
	return uint8(math.Round(min(max(v, 0), 1) * math.MaxUint8))
}

func (c RGBA) distance(r, g, b int) int {
	dr, dg, db := int(c.R)-r, int(c.G)-g, int(c.B)-b
	return dr*dr + dg*dg + db*db
}

func (c RGBA) ANSI256() uint8 {
	level := func(i int) int {
		if i == 0 {
			return 0
		}
		return konst.CubeFirstLevel + (i-1)*konst.CubeLevelStep
	}
	nearest := func(v uint8) int {
		best := 0
		for i := 1; i < konst.CubeSide; i++ {
			if abs(int(v)-level(i)) < abs(int(v)-level(best)) {
				best = i
			}
		}
		return best
	}
	r, g, b := nearest(c.R), nearest(c.G), nearest(c.B)
	avg := (int(c.R) + int(c.G) + int(c.B)) / 3
	grey := min(max((avg-konst.GreyFirstLevel+konst.GreyLevelStep/2)/konst.GreyLevelStep, 0), konst.GreySteps-1)
	greyLevel := konst.GreyFirstLevel + grey*konst.GreyLevelStep
	if c.distance(greyLevel, greyLevel, greyLevel) < c.distance(level(r), level(g), level(b)) {
		return uint8(konst.GreyBase + grey)
	}
	return uint8(konst.CubeBase + (r*konst.CubeSide+g)*konst.CubeSide + b)
}

func (c RGBA) ANSI16() uint8 {
	xterm := [...]uint32{
		0x000000, 0xcd0000, 0x00cd00, 0xcdcd00, 0x0000ee, 0xcd00cd, 0x00cdcd, 0xe5e5e5,
		0x7f7f7f, 0xff0000, 0x00ff00, 0xffff00, 0x5c5cff, 0xff00ff, 0x00ffff, 0xffffff,
	}
	best, bestDistance := 0, math.MaxInt
	for i, p := range xterm {
		if d := c.distance(int(p>>16), int(p>>8&0xff), int(p&0xff)); d < bestDistance {
			best, bestDistance = i, d
		}
	}
	return uint8(best)
}

func abs(v int) int {
	return max(v, -v)
}
