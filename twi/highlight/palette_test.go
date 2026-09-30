package highlight_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/theme"
)

func wcagLuminance(c color.RGBA) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

func TestPaletteReadableOnEveryTheme(t *testing.T) {
	var table strings.Builder
	for _, th := range theme.Builtin() {
		p := highlight.DefaultPalette(th)
		bg := wcagLuminance(th.Tokens[theme.Muted].RGBA)
		scheme := [...]string{theme.Light: "light", theme.Dark: "dark"}[th.Scheme]
		fmt.Fprintf(&table, "%s-%s", th.Name, scheme)
		for k := highlight.Text; k <= highlight.Constant; k++ {
			fg := wcagLuminance(th.Tokens[p[k]].RGBA)
			ratio := (max(fg, bg) + 0.05) / (min(fg, bg) + 0.05)
			fmt.Fprintf(&table, " %s=%s:%.1f", k, p[k], ratio)
			if ratio < 4.5 {
				t.Errorf("%s %s: %s as %s is %.2f:1 on muted", th.Name, scheme, k, p[k], ratio)
			}
		}
		table.WriteString("\n")
	}
	t.Log("\n" + table.String())
}
