package highlight

import (
	"math"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

const readableContrast = 4.5

type Palette [kindEnd]theme.Token

func DefaultPalette(t theme.Theme) Palette {
	keyword := []theme.Token{theme.Chart4, theme.Chart1, theme.Primary}
	function := []theme.Token{theme.Chart1, theme.Chart4, theme.Primary}
	str := []theme.Token{theme.Chart2, theme.Chart3, theme.Chart1}
	accent := []theme.Token{theme.Chart3, theme.Chart2, theme.Chart1}
	number := []theme.Token{theme.Chart5, theme.Destructive, theme.Chart1}
	quiet := []theme.Token{theme.MutedForeground, theme.Chart3, theme.Chart2}
	choices := [kindEnd][]theme.Token{
		Keyword: keyword, String: str, Escape: accent, Number: number, Comment: quiet, Function: function,
		Punctuation: quiet, Property: function, Boolean: number, Variable: accent, Builtin: function, Regex: accent,
		Datetime: number, TableHeader: {theme.Primary, theme.Chart4, theme.Chart1}, Namespace: accent,
		Parameter: accent, Constant: number,
	}
	bg := luminance(t.Tokens[theme.Muted].RGBA)
	var p Palette
	for k, tokens := range choices {
		p[k] = theme.Foreground
		for _, token := range tokens {
			fg := luminance(t.Tokens[token].RGBA)
			if (max(fg, bg)+0.05)/(min(fg, bg)+0.05) >= readableContrast {
				p[k] = token
				break
			}
		}
	}
	return p
}

func luminance(c color.RGBA) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}
