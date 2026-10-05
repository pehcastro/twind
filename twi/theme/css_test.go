package theme_test

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/theme/shadcn"
)

var owners = []string{"twind", "dream", "mono", "minimal", "dew", "cloud", "sukuna"}

func builtin(t *testing.T, name string, scheme theme.Scheme) theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if th.Name == name && th.Scheme == scheme {
			return th
		}
	}
	t.Fatalf("%s %d: not built in", name, scheme)
	return theme.Theme{}
}

func oklchToSRGB(t *testing.T, body string) [3]float64 {
	t.Helper()
	f := strings.Fields(body)
	if len(f) != 3 {
		t.Fatalf("oklch(%s): want three channels", body)
	}
	var v [3]float64
	for i, s := range f {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(s, "%") {
			n /= 100
		}
		v[i] = n
	}
	l, a, b := v[0], v[1]*math.Cos(v[2]*math.Pi/180), v[1]*math.Sin(v[2]*math.Pi/180)
	cube := func(x float64) float64 { return x * x * x }
	lc := cube(l + 0.3963377774*a + 0.2158037573*b)
	mc := cube(l - 0.1055613458*a - 0.0638541728*b)
	sc := cube(l - 0.0894841775*a - 1.2914855480*b)
	lin := [3]float64{
		4.0767416621*lc - 3.3077115913*mc + 0.2309699292*sc,
		-1.2684380046*lc + 2.6097574011*mc - 0.3413193965*sc,
		-0.0041960863*lc - 0.7034186147*mc + 1.7076147010*sc,
	}
	for i, x := range lin {
		x = min(max(x, 0), 1)
		if x > 0.0031308 {
			x = 1.055*math.Pow(x, 1/2.4) - 0.055
		} else {
			x *= 12.92
		}
		lin[i] = x * 255
	}
	return lin
}

func TestCSSTokensMatchTheOwnersThemes(t *testing.T) {
	decl := regexp.MustCompile(`^\s*--([a-z0-9-]+):\s*oklch\(([^)]*)\);\s*$`)
	for _, name := range owners {
		src, err := os.ReadFile(filepath.Join("css", name+".css"))
		if err != nil {
			t.Fatal(err)
		}
		themes := [...]theme.Theme{theme.Light: builtin(t, name, theme.Light), theme.Dark: builtin(t, name, theme.Dark)}
		seen := [theme.Dark + 1]map[theme.Token]bool{{}, {}}
		scheme := -1
		for i, line := range strings.Split(string(src), "\n") {
			switch strings.TrimSpace(line) {
			case ":root {":
				scheme = int(theme.Light)
			case ".dark {":
				scheme = int(theme.Dark)
			case "}":
				scheme = -1
			}
			m := decl.FindStringSubmatch(line)
			if scheme < 0 || m == nil {
				continue
			}
			tok, ok := theme.ParseToken(m[1])
			if !ok {
				continue
			}
			if seen[scheme][tok] {
				t.Errorf("%s.css:%d: %s twice", name, i+1, m[1])
			}
			seen[scheme][tok] = true
			want, got := oklchToSRGB(t, m[2]), themes[scheme].Tokens[tok]
			rgb := [3]uint8{got.RGBA.R, got.RGBA.G, got.RGBA.B}
			for c := range rgb {
				if got.Kind != color.Literal || math.Abs(float64(rgb[c])-want[c]) > 1 || got.RGBA.A != 255 {
					t.Errorf("%s.css:%d: %s is %v, want %.1f", name, i+1, m[1], got, want)
					break
				}
			}
		}
		for s, set := range seen {
			shadcn := 0
			for tok := range set {
				if tok < theme.Selection || tok == theme.DestructiveForeground {
					shadcn++
				}
			}
			if shadcn != 32 {
				t.Errorf("%s scheme %d: %d shadcn tokens compared, want 32", name, s, shadcn)
			}
		}
	}
}

func valid(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", "neutral.css"))
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

func TestCSSMalformedFailsWithItsLine(t *testing.T) {
	neutral := valid(t)
	for _, c := range []struct {
		name, src string
		line      int
	}{
		{"colour", ":root {\n  --background: oklch(1 0);\n}\n", 2},
		{"unknown variable", ":root {\n\n  --canvas: #fff;\n}\n", 3},
		{"no colon", ":root {\n  --background #fff;\n}\n", 2},
		{"unclosed", "@layer base {\n  :root {\n    --background: #fff;\n", 2},
		{"stray brace", ":root {\n}\n}\n", 3},
		{"missing token", strings.Replace(neutral, "  --ring: oklch(0.708 0 0);\n", "", 1), 1},
		{"radius", strings.Replace(neutral, ":root {\n", ":root {\n  --radius: big;\n", 1), 2},
		{"shadow", strings.Replace(neutral, ":root {\n", ":root {\n  --shadow: 0 1px nope 0 #000;\n", 1), 2},
		{"currentcolor", strings.Replace(neutral, ".dark {\n", ".dark {\n  --ring: currentColor;\n", 1), 36},
	} {
		_, _, err := shadcn.Parse("bad.css", []byte(c.src))
		var perr shadcn.Error
		if !errors.As(err, &perr) || perr.File != "bad.css" || perr.Line != c.line {
			t.Errorf("%s: %v, want an error at bad.css:%d", c.name, err, c.line)
		}
	}
}

func TestCSSColourFormsAndInheritance(t *testing.T) {
	neutral := valid(t)
	src := strings.NewReplacer(
		"  --background: oklch(1 0 0);\n", "  /* } ; { */\n  --font-sans: \"A;}\", serif;\n  --background: 0 0% 100%;\n",
		"  --card: oklch(1 0 0);\n", "  --card: hsl(120deg 100% 25%);\n",
		"  --popover: oklch(1 0 0);\n", "  --popover: hsl(210, 40%, 98%);\n",
		"  --primary: oklch(0.205 0 0);\n", "  --primary: #ff000080;\n  --syntax-comment: #123456;\n  --shadow-sm: 0px 4px 10px 0px hsl(0, 0, 0 / 0.10), 0 1px 2px -1px hsl(0 0% 0% / 2.00);\n  --radius: 0.625rem;\n",
	).Replace(neutral)
	src = strings.Replace(src, "  --ring: oklch(0.556 0 0);\n", "", 1)
	th, warnings, err := shadcn.Parse("forms.css", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	light, dark := th.WithScheme(theme.Light), th.WithScheme(theme.Dark)
	for _, c := range []struct {
		tok  theme.Token
		want color.RGBA
	}{
		{theme.Background, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{theme.Card, color.RGBA{G: 128, A: 255}},
		{theme.Popover, color.RGBA{R: 248, G: 250, B: 252, A: 255}},
		{theme.Primary, color.RGBA{R: 255, A: 128}},
	} {
		if got := light.Tokens[c.tok].RGBA; got != c.want {
			t.Errorf("%s = %v, want %v", c.tok, got, c.want)
		}
	}
	for _, tok := range []theme.Token{theme.Ring, theme.SyntaxComment} {
		if dark.Tokens[tok] != light.Tokens[tok] {
			t.Errorf("dark %s %v: .dark leaves it out, so it inherits :root %v", tok, dark.Tokens[tok], light.Tokens[tok])
		}
	}
	if th.Name != "forms" || th.Scheme != theme.Dark {
		t.Errorf("parsed as %q scheme %d, want forms dark", th.Name, th.Scheme)
	}
	if len(warnings) == 0 || warnings[0].Line != 3 || warnings[0].Name != "--font-sans" {
		t.Errorf("warnings %v: want --font-sans ignored at line 3 first", warnings)
	}
}

func TestGeneratedThemesAreCurrent(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("css", "*.css"))
	if err != nil || len(paths) != len(owners) {
		t.Fatalf("%d css files, %v: want %d", len(paths), err, len(owners))
	}
	var files []shadcn.File
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, shadcn.File{Path: p, Src: src})
	}
	want, _, err := shadcn.Generate("theme", files)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("builtin_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	g, w := strings.Split(strings.ReplaceAll(string(got), "\r\n", "\n"), "\n"), strings.Split(string(want), "\n")
	for i := range max(len(g), len(w)) {
		if i >= len(g) || i >= len(w) || g[i] != w[i] {
			t.Fatalf("builtin_gen.go is stale from line %d, run go generate ./twi/theme", i+1)
		}
	}
}

func TestSchemeSwapsEveryToken(t *testing.T) {
	all := theme.Builtin()
	if len(all) != 2*len(owners) {
		t.Fatalf("%d built-in themes, want %d in two schemes", len(all), len(owners))
	}
	for _, th := range all {
		for _, s := range []theme.Scheme{theme.Light, theme.Dark} {
			got, want := th.WithScheme(s), builtin(t, th.Name, s)
			if got.Scheme != s || got.Name != th.Name {
				t.Errorf("%s %d to %d: got %s %d", th.Name, th.Scheme, s, got.Name, got.Scheme)
			}
			for tok := theme.Background; tok <= theme.DestructiveForeground; tok++ {
				if got.Tokens[tok] != want.Tokens[tok] || got.Tokens[tok].Kind != color.Literal {
					t.Errorf("%s %d to %d: %s is %v, want %v", th.Name, th.Scheme, s, tok, got.Tokens[tok], want.Tokens[tok])
				}
			}
		}
	}
}
