package shadcn

import (
	_ "embed"
	"fmt"
	"go/format"
	"go/token"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
)

//go:embed base.css
var base string

type File struct {
	Path string
	Src  []byte
}

type Error struct {
	File   string
	Line   int
	Reason string
}

func (e Error) Error() string { return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Reason) }

type Warning struct {
	File   string
	Line   int
	Name   string
	Reason string
}

type decl struct {
	line, scheme int
	name, value  string
}

type sheet struct {
	tokens [theme.Dark + 1]theme.Tokens
	opened [theme.Dark + 1]int
}

const (
	other = -1
	light = int(theme.Light)
	dark  = int(theme.Dark)
)

func Parse(path string, src []byte) (theme.Theme, []Warning, error) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	th := theme.Theme{Name: name}
	if !token.IsIdentifier(name) {
		return th, nil, Error{path, 1, "the file name " + strconv.Quote(name) + " is not a Go identifier"}
	}
	defaults, _, err := read("base.css", base)
	if err != nil {
		return th, nil, err
	}
	s, warnings, err := read(path, string(src))
	if err != nil {
		return th, warnings, err
	}
	if s.opened[light] == 0 {
		return th, warnings, Error{path, 1, "no :root block"}
	}
	if s.opened[dark] == 0 {
		warnings = append(warnings, Warning{path, 1, ".dark", "no .dark block, the dark scheme repeats :root"})
	}
	for tok := theme.Background; int(tok) < len(theme.Tokens{}); tok++ {
		for scheme, layers := range [][]theme.Tokens{
			light: {s.tokens[light], defaults.tokens[light]},
			dark:  {s.tokens[dark], s.tokens[light], defaults.tokens[dark]},
		} {
			switch at := slices.IndexFunc(layers, func(t theme.Tokens) bool { return t[tok].Kind == color.Literal }); {
			case at >= 0:
				th.Schemes[scheme][tok] = layers[at][tok]
			case tok < theme.Primary50:
				return th, warnings, Error{path, s.opened[light], "missing --" + tok.String()}
			}
		}
	}
	for scheme := range th.Schemes {
		ramp(&th.Schemes[scheme])
	}
	return th.WithScheme(theme.Dark), warnings, nil
}

func oklab(c color.RGBA) (l, a, b float64) {
	lin := func(v uint8) float64 {
		s := float64(v) / math.MaxUint8
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, bl := lin(c.R), lin(c.G), lin(c.B)
	lc := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*bl)
	mc := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*bl)
	sc := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*bl)
	return 0.2104542553*lc + 0.7936177850*mc - 0.0040720468*sc, 1.9779984951*lc - 2.4285922050*mc + 0.4505937099*sc, 0.0259040371*lc + 0.7827717662*mc - 0.8086757660*sc
}

func ramp(tokens *theme.Tokens) {
	for base, first := range map[theme.Token]theme.Token{theme.Primary: theme.Primary50, theme.Accent: theme.Accent50} {
		l, a, b := oklab(tokens[base].RGBA)
		chroma, hue, alpha := math.Hypot(a, b), math.Atan2(b, a)*180/math.Pi, float64(tokens[base].RGBA.A)/math.MaxUint8
		tokens[first+konst.RampSide] = tokens[base]
		for _, dir := range []int{-1, 1} {
			target, fade := min(1, max(konst.RampLightest, l+konst.RampSpan)), konst.RampLightFade
			if dir > 0 {
				target, fade = max(0, min(konst.RampDarkest, l-konst.RampSpan)), konst.RampDarkFade
			}
			for k := 1; k <= konst.RampSide; k++ {
				tok := first + theme.Token(konst.RampSide+dir*k)
				if tokens[tok].Kind == color.Literal {
					continue
				}
				f := 1 - math.Pow(1-float64(k)/konst.RampSide, konst.RampEase)
				step, _ := color.Parse(fmt.Sprintf("oklch(%f %f %f / %f)", l+(target-l)*f, chroma*(1-f*fade), hue, alpha))
				inner := tokens[first+theme.Token(konst.RampSide+dir*(k-1))]
				stepL, _, _ := oklab(step.RGBA)
				if innerL, _, _ := oklab(inner.RGBA); float64(dir)*(stepL-innerL) > 0 {
					step = inner
				}
				tokens[tok] = step
			}
		}
	}
}

func scan(path, src string) ([]decl, [theme.Dark + 1]int, error) {
	var opened [theme.Dark + 1]int
	var decls []decl
	type open struct{ scheme, line int }
	var stack []open
	var seg strings.Builder
	line, segLine := 1, 0
	flush := func() error {
		text := strings.TrimSpace(seg.String())
		seg.Reset()
		if text == "" || len(stack) == 0 || stack[len(stack)-1].scheme == other {
			return nil
		}
		name, value, ok := strings.Cut(text, ":")
		if !ok {
			return Error{path, segLine, "want name: value, got " + strconv.Quote(text)}
		}
		decls = append(decls, decl{segLine, stack[len(stack)-1].scheme, strings.TrimSpace(name), strings.TrimSpace(value)})
		return nil
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '/' && strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return nil, opened, Error{path, line, "unclosed comment"}
			}
			line += strings.Count(src[i:i+2+end], "\n")
			i += end + 3
			continue
		case c == '"' || c == '\'':
			end := i + 1
			for end < len(src) && src[end] != c {
				if src[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(src) {
				return nil, opened, Error{path, line, "unclosed string"}
			}
			if seg.Len() == 0 {
				segLine = line
			}
			seg.WriteString(src[i : end+1])
			line += strings.Count(src[i:end], "\n")
			i = end
			continue
		case c == '{':
			scheme := other
			for _, sel := range strings.Split(seg.String(), ",") {
				switch strings.TrimSpace(sel) {
				case ":root":
					scheme = light
				case ".dark":
					scheme = dark
				}
			}
			seg.Reset()
			if scheme != other && opened[scheme] == 0 {
				opened[scheme] = line
			}
			stack = append(stack, open{scheme, line})
			continue
		case c == ';' || c == '}':
			if err := flush(); err != nil {
				return nil, opened, err
			}
			if c == ';' {
				continue
			}
			if len(stack) == 0 {
				return nil, opened, Error{path, line, "a } closes nothing"}
			}
			stack = stack[:len(stack)-1]
			continue
		case c == '\n':
			line++
		}
		if seg.Len() == 0 {
			if c <= ' ' {
				continue
			}
			segLine = line
		}
		seg.WriteByte(c)
	}
	if len(stack) > 0 {
		return nil, opened, Error{path, stack[len(stack)-1].line, "unclosed block"}
	}
	return decls, opened, nil
}

func read(path, src string) (sheet, []Warning, error) {
	var s sheet
	var warnings []Warning
	decls, opened, err := scan(path, src)
	if err != nil {
		return s, nil, err
	}
	s.opened = opened
	ignored := map[string]int{}
	for _, d := range decls {
		reason, failed := resolve(&s.tokens[d.scheme], d.name, d.value)
		if failed != "" {
			return s, warnings, Error{path, d.line, d.name + ": " + failed}
		}
		if reason == "" {
			continue
		}
		at, seen := ignored[reason]
		switch {
		case !seen:
			ignored[reason] = len(warnings)
			warnings = append(warnings, Warning{path, d.line, d.name, reason})
		case !slices.Contains(strings.Split(warnings[at].Name, ", "), d.name):
			warnings[at].Name += ", " + d.name
		}
	}
	return s, warnings, nil
}

func isLength(s string) bool {
	for _, unit := range []string{"px", "rem", "em"} {
		if n, ok := strings.CutSuffix(s, unit); ok {
			s = n
			break
		}
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func resolve(tokens *theme.Tokens, name, value string) (ignored, failed string) {
	if !strings.HasPrefix(name, "--") {
		return "not a custom property", ""
	}
	if tok, ok := theme.ParseToken(name[2:]); ok {
		c, ok := parseColor(value)
		if !ok {
			return "", "cannot read the colour " + strconv.Quote(value)
		}
		tokens[tok] = c
		return "", ""
	}
	switch {
	case strings.HasPrefix(name, "--font-"):
		return "a terminal has one font", ""
	case strings.HasPrefix(name, "--tracking-") || name == "--spacing":
		return "a terminal has one cell size", ""
	case slices.Contains([]string{"--shadow-x", "--shadow-y", "--shadow-blur", "--shadow-spread", "--shadow-opacity", "--shadow-color"}, name):
		return "the parts of a shadow, the --shadow values carry them", ""
	case name == "--radius":
		if !isLength(value) {
			return "", "want a length, got " + strconv.Quote(value)
		}
		return "radius and shadows come from Tailwind classes", ""
	case name == "--shadow" || slices.Contains([]string{"2xs", "xs", "sm", "md", "lg", "xl", "2xl"}, strings.TrimPrefix(name, "--shadow-")):
		if !shadow(value) {
			return "", "want a box-shadow, got " + strconv.Quote(value)
		}
		return "radius and shadows come from Tailwind classes", ""
	}
	return "", "not a shadcn or Twind theme variable"
}

func fields(s string, sep byte) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i <= len(s); i++ {
		switch {
		case i == len(s) || s[i] == sep && depth == 0:
			if f := strings.TrimSpace(s[start:i]); f != "" {
				out = append(out, f)
			}
			start = i + 1
		case s[i] == '(':
			depth++
		case s[i] == ')':
			depth--
		}
	}
	return out
}

func shadow(value string) bool {
	if value == "none" {
		return true
	}
	for _, layer := range fields(value, ',') {
		lengths, rest := 0, []string{}
		for _, f := range fields(layer, ' ') {
			switch {
			case f == "inset":
			case isLength(f) && len(rest) == 0:
				lengths++
			default:
				rest = append(rest, f)
			}
		}
		if _, ok := parseColor(strings.Join(rest, " ")); lengths < 2 || lengths > 4 || len(rest) > 1 || len(rest) == 1 && !ok {
			return false
		}
	}
	return true
}

func parseColor(value string) (color.Color, bool) {
	s := strings.ToLower(value)
	fn, body, isFunc := strings.Cut(s, "(")
	if !isFunc && !strings.HasPrefix(s, "#") && len(strings.Fields(s)) == 3 {
		fn, body, isFunc = "hsl", s+")", true
	}
	if isFunc && (fn == "hsl" || fn == "hsla") {
		return hsl(body)
	}
	c, err := color.Parse(value)
	return c, err == nil && c.Kind == color.Literal
}

func hsl(body string) (color.Color, bool) {
	body, closed := strings.CutSuffix(strings.TrimSpace(body), ")")
	channels, alphaText, hasAlpha := strings.Cut(body, "/")
	f := strings.Fields(strings.ReplaceAll(channels, ",", " "))
	if len(f) == 4 && !hasAlpha {
		alphaText, hasAlpha, f = f[3], true, f[:3]
	}
	if !closed || len(f) != 3 {
		return color.Color{}, false
	}
	unit := func(s string, bare float64) (float64, bool) {
		pct, isPct := strings.CutSuffix(strings.TrimSpace(s), "%")
		n, err := strconv.ParseFloat(pct, 64)
		if isPct {
			bare = 100
		}
		return min(max(n/bare, 0), 1), err == nil
	}
	h, errH := strconv.ParseFloat(strings.TrimSuffix(f[0], "deg"), 64)
	sat, okS := unit(f[1], 100)
	l, okL := unit(f[2], 100)
	a, okA := 1.0, true
	if hasAlpha {
		a, okA = unit(alphaText, 1)
	}
	if errH != nil || !okS || !okL || !okA {
		return color.Color{}, false
	}
	h = math.Mod(math.Mod(h, 360)+360, 360)
	ch := func(n float64) uint8 {
		k := math.Mod(n+h/30, 12)
		return uint8(math.Round((l - sat*min(l, 1-l)*max(-1, min(k-3, 9-k, 1))) * 255))
	}
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: ch(0), G: ch(8), B: ch(4), A: uint8(math.Round(a * 255))}}, true
}

func goName(tok theme.Token) string {
	var b strings.Builder
	for _, part := range strings.Split(tok.String(), "-") {
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

func Generate(pkg string, files []File) ([]byte, []Warning, error) {
	q := "theme."
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\nimport (\n\"github.com/pehcastro/twind/twi/color\"\n", pkg)
	if pkg == "theme" {
		q = ""
	} else {
		b.WriteString("\"github.com/pehcastro/twind/twi/theme\"\n")
	}
	b.WriteString(")\n")
	var warnings []Warning
	for _, f := range files {
		th, w, err := Parse(f.Path, f.Src)
		warnings = append(warnings, w...)
		if err != nil {
			return nil, warnings, err
		}
		fmt.Fprintf(&b, "\nfunc %s() %sTheme {\nreturn %sTheme{Name: %q, Schemes: [%sDark + 1]%sTokens{\n", th.Name, q, q, th.Name, q, q)
		for scheme, tokens := range th.Schemes {
			fmt.Fprintf(&b, "%s%s: {\n", q, [...]string{light: "Light", dark: "Dark"}[scheme])
			for tok := theme.Background; int(tok) < len(tokens); tok++ {
				c := tokens[tok].RGBA
				fmt.Fprintf(&b, "%s%s: {Kind: color.Literal, RGBA: color.RGBA{R: %d, G: %d, B: %d, A: %d}},\n", q, goName(tok), c.R, c.G, c.B, c.A)
			}
			b.WriteString("},\n")
		}
		fmt.Fprintf(&b, "}}.WithScheme(%sDark)\n}\n", q)
	}
	out, err := format.Source([]byte(b.String()))
	return out, warnings, err
}
