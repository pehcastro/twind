package theme_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
)

func lightness(c color.RGBA) float64 {
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
}

func TestRampFromToken(t *testing.T) {
	steps := []int{50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950}
	for _, th := range theme.Builtin() {
		for _, base := range []theme.Token{theme.Primary, theme.Accent} {
			token := th.Tokens[base]
			var ls []float64
			for _, step := range steps {
				name := base.String() + "-" + strconv.Itoa(step)
				tok, ok := theme.ParseToken(name)
				c := th.Tokens[tok]
				switch {
				case !ok || tok.String() != name:
					t.Fatalf("%s: not a token", name)
				case c.Kind != color.Literal:
					t.Errorf("%s %d: %s unset", th.Name, th.Scheme, name)
				case c.RGBA.A != token.RGBA.A:
					t.Errorf("%s %d: %s alpha %d, want the token's %d", th.Name, th.Scheme, name, c.RGBA.A, token.RGBA.A)
				case step == 500 && c != token:
					t.Errorf("%s %d: %s %v, want %s %v", th.Name, th.Scheme, name, c.RGBA, base, token.RGBA)
				}
				ls = append(ls, lightness(c.RGBA))
			}
			for i := 1; i < len(ls); i++ {
				if ls[i] > ls[i-1] {
					t.Errorf("%s %d: %s-%d L %.4f is lighter than %s-%d L %.4f", th.Name, th.Scheme, base, steps[i], ls[i], base, steps[i-1], ls[i-1])
				}
			}
			if ls[0] <= ls[len(ls)-1] || ls[4]-ls[5] > 0.25 || ls[5]-ls[6] > 0.25 {
				t.Errorf("%s %d: %s lightness %.3f, want 50 above 950 and 400 and 600 within 0.25 of 500", th.Name, th.Scheme, base, ls)
			}
		}
	}
}
