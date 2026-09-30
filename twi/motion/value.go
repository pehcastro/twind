package motion

import (
	"math"
	"slices"

	konst "github.com/twind-dev/twind/internal/konst/motion"

	"github.com/twind-dev/twind/twi/color"
)

type Value struct{ ch [4]float64 }

func Float(f float64) Value { return Value{[4]float64{f}} }

func Bounds(x, y, w, h float64) Value { return Value{[4]float64{x, y, w, h}} }

func Color(c color.RGBA) Value {
	r, g, b := linearise(float64(c.R)/255), linearise(float64(c.G)/255), linearise(float64(c.B)/255)
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
	r, g, b := uncone(v.cone())
	return color.RGBA{R: encode(r), G: encode(g), B: encode(b), A: alpha(v.ch[3])}
}

func (v Value) cone() (l, m, s float64) {
	lab := v.ch
	return lab[0] + 0.3963377774*lab[1] + 0.2158037573*lab[2],
		lab[0] - 0.1055613458*lab[1] - 0.0638541728*lab[2],
		lab[0] - 0.0894841775*lab[1] - 1.2914855480*lab[2]
}

func uncone(l, m, s float64) (r, g, b float64) {
	l, m, s = l*l*l, m*m*m, s*s*s
	return 4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.7076147010*s
}

func alpha(a float64) uint8 {
	if a <= 0 {
		return 0
	}
	if a >= 1 {
		return 255
	}
	return uint8(a*255 + 0.5)
}

func linearise(v float64) float64 {
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

type srgb struct {
	rounds [255]float64
	starts [konst.EncodeBuckets]uint8
}

func (t *srgb) build() {
	for k := range t.rounds {
		t.rounds[k] = linearise((float64(k) + 0.5) / 255)
	}
	for j := range t.starts {
		code, exact := slices.BinarySearch(t.rounds[:], float64(j)/konst.EncodeBuckets)
		if exact {
			code++
		}
		t.starts[j] = uint8(code)
	}
}

func (t *srgb) rgba(premixed [4]float64) color.RGBA {
	a := premixed[3]
	if a <= 0 {
		return color.RGBA{}
	}
	if t.rounds[len(t.rounds)-1] == 0 {
		t.build()
	}
	unit := 1 / a
	r, g, b := uncone(premixed[0]*unit, premixed[1]*unit, premixed[2]*unit)
	return color.RGBA{R: t.encode(r), G: t.encode(g), B: t.encode(b), A: alpha(a)}
}

func (t *srgb) encode(linear float64) uint8 {
	if linear <= 0 {
		return 0
	}
	if linear >= 1 {
		return 255
	}
	code := t.starts[int(linear*konst.EncodeBuckets)]
	if code < 255 && linear >= t.rounds[code] {
		code++
	}
	return code
}
