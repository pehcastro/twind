package motion

import (
	"math"

	"github.com/twind-dev/twind/twi/color"
)

type Value struct{ ch [4]float64 }

func Float(f float64) Value { return Value{[4]float64{f}} }

func Bounds(x, y, w, h float64) Value { return Value{[4]float64{x, y, w, h}} }

func Color(c color.RGBA) Value {
	r, g, b := linearise(c.R), linearise(c.G), linearise(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return Value{[4]float64{
		0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
		float64(c.A) / 255,
	}}
}

func (v Value) Float() float64 { return v.ch[0] }

func (v Value) Bounds() (x, y, w, h float64) { return v.ch[0], v.ch[1], v.ch[2], v.ch[3] }

func (v Value) Color() color.RGBA {
	lab := v.ch
	l := lab[0] + 0.3963377774*lab[1] + 0.2158037573*lab[2]
	m := lab[0] - 0.1055613458*lab[1] - 0.0638541728*lab[2]
	s := lab[0] - 0.0894841775*lab[1] - 1.2914855480*lab[2]
	l, m, s = l*l*l, m*m*m, s*s*s
	return color.RGBA{
		R: encode(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		G: encode(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		B: encode(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
		A: uint8(math.Round(min(max(lab[3], 0), 1) * 255)),
	}
}

func linearise(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func encode(v float64) uint8 {
	v = min(max(v, 0), 1)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(v * 255))
}
