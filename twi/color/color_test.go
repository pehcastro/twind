package color

import (
	"os"
	"strings"
	"testing"
)

func TestOKLCHTheme(t *testing.T) {
	theme, err := os.ReadFile("testdata/theme-4.3.3.css")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]RGBA{}
	for line := range strings.Lines(string(theme)) {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		c, err := Parse(strings.TrimSuffix(strings.TrimSpace(value), ";"))
		if err != nil || c.Kind != Literal {
			t.Fatalf("%s: %v %+v", name, err, c)
		}
		got[name] = c.RGBA
	}
	if len(got) != 288 {
		t.Fatalf("parsed %d colours, want 288", len(got))
	}
	for name, want := range map[string]RGBA{
		"--color-zinc-950": {0x09, 0x09, 0x0b, 0xff},
		"--color-zinc-100": {0xf4, 0xf4, 0xf5, 0xff},
		"--color-black":    {0, 0, 0, 0xff},
		"--color-white":    {0xff, 0xff, 0xff, 0xff},
	} {
		c := got[name]
		t.Logf("%s = %d %d %d %d", name, c.R, c.G, c.B, c.A)
		if apart(c.R, want.R) || apart(c.G, want.G) || apart(c.B, want.B) || c.A != want.A {
			t.Errorf("%s = %+v, want %+v within 1", name, c, want)
		}
	}
}

func apart(a, b uint8) bool {
	return int(a)-int(b) > 1 || int(b)-int(a) > 1
}

func TestOKLCHForms(t *testing.T) {
	for in, want := range map[string]RGBA{
		"oklch(0.141 0.005 285.823)":       {0x09, 0x09, 0x0b, 0xff},
		"OKLCH(14.1% 0.005 285.823deg)":    {0x09, 0x09, 0x0b, 0xff},
		"oklch(100% 0 0 / 50%)":            {0xff, 0xff, 0xff, 0x80},
		"oklch(100% 0 0 / 0.5)":            {0xff, 0xff, 0xff, 0x80},
		"oklch(0% 0 0 / 2)":                {0, 0, 0, 0xff},
		"oklch(150% -1 0)":                 {0xff, 0xff, 0xff, 0xff},
		"oklch(62.8% 0.2577 29.23)":        {0xff, 0, 0, 0xff},
		"oklch(70% 0.4 30)":                {0xff, 0, 0, 0xff},
		"oklch(45.2% 0.313214 264.052)":    {0, 0, 0xff, 0xff},
		"  oklch(  100%   0   0  )  ":      {0xff, 0xff, 0xff, 0xff},
		"oklch(51.975% 0.1769 142.495 /1)": {0, 0x80, 0, 0xff},
		"oklch(100% none none)":            {0xff, 0xff, 0xff, 0xff},
		"oklch(100% 0 0 / none)":           {0xff, 0xff, 0xff, 0},
	} {
		c, err := Parse(in)
		if err != nil || c.Kind != Literal || c.RGBA != want {
			t.Errorf("%q = %+v %v, want %+v", in, c.RGBA, err, want)
		}
	}
}

func TestParseKeywordsAndHex(t *testing.T) {
	for in, want := range map[string]Color{
		"transparent":  {Kind: Literal},
		"currentcolor": {Kind: Current},
		"CurrentColor": {Kind: Current},
		"#000":         {Kind: Literal, RGBA: RGBA{0, 0, 0, 0xff}},
		"#FfF":         {Kind: Literal, RGBA: RGBA{0xff, 0xff, 0xff, 0xff}},
		"#1234":        {Kind: Literal, RGBA: RGBA{0x11, 0x22, 0x33, 0x44}},
		"#09090b":      {Kind: Literal, RGBA: RGBA{0x09, 0x09, 0x0b, 0xff}},
		"#09090b80":    {Kind: Literal, RGBA: RGBA{0x09, 0x09, 0x0b, 0x80}},
	} {
		c, err := Parse(in)
		if err != nil || c != want {
			t.Errorf("%q = %+v %v, want %+v", in, c, err, want)
		}
	}
	if (Color{}) == (Color{Kind: Literal, RGBA: RGBA{A: 0xff}}) {
		t.Error("unset equals black")
	}
}

func TestParseMalformed(t *testing.T) {
	for _, in := range []string{
		"", "#", "#12", "#12345", "#1234567", "#gg0000", "#+12", "# 12",
		"oklch(", "oklch()", "oklch(50%)", "oklch(50% 0.1)", "oklch(50% 0.1 10 20)",
		"oklch(50% x 10)", "oklch(50%% 0.1 10)", "oklch(50% 0.1 10deg2)", "oklch(50% 0.1% 10)",
		"oklch(50% 0.1 10%)", "oklch(50% 0.1 10 /)", "oklch(50% 0.1 10 / x)", "oklch(nan 0.1 10)",
		"oklch(50% inf 10)", "oklch(50% 0.1 10", "oklch 50% 0.1 10)", "oklch(none% 0 0)", "red",
	} {
		c, err := Parse(in)
		if err == nil {
			t.Errorf("%q = %+v, want an error", in, c)
		}
	}
}

func TestDowngrade(t *testing.T) {
	for _, tc := range []struct {
		c    RGBA
		i256 uint8
		i16  uint8
		name string
	}{
		{RGBA{0xff, 0, 0, 0xff}, 196, 9, "exact cube hit"},
		{RGBA{0x09, 0x09, 0x0b, 0xff}, 232, 0, "zinc-950 is on the grey ramp"},
		{RGBA{0, 0, 0, 0xff}, 16, 0, "black"},
		{RGBA{0xff, 0xff, 0xff, 0xff}, 231, 15, "white"},
		{RGBA{0x80, 0x80, 0x80, 0xff}, 244, 8, "mid grey nearer the ramp than the cube"},
		{RGBA{0xf4, 0xf4, 0xf5, 0xff}, 255, 15, "zinc-100"},
		{RGBA{0x5f, 0x87, 0xaf, 0xff}, 67, 8, "exact cube hit off the corners"},
	} {
		if got := tc.c.ANSI256(); got != tc.i256 {
			t.Errorf("%s: ANSI256 = %d, want %d", tc.name, got, tc.i256)
		}
		if got := tc.c.ANSI16(); got != tc.i16 {
			t.Errorf("%s: ANSI16 = %d, want %d", tc.name, got, tc.i16)
		}
	}
}

func TestRGBForms(t *testing.T) {
	for in, want := range map[string]RGBA{
		"rgb(0 0 0 / 0.1)":             {0, 0, 0, 26},
		"rgb(0 0 0 / 0.07)":            {0, 0, 0, 18},
		"rgb(255 128 0)":               {255, 128, 0, 255},
		"RGB(255 128 0 / 50%)":         {255, 128, 0, 128},
		"rgba(255 128 0)":              {255, 128, 0, 255},
		"rgb(100% 50% 0%)":             {255, 128, 0, 255},
		"rgb(100% 128 0)":              {255, 128, 0, 255},
		"rgb(300 -5 0 / 2)":            {255, 0, 0, 255},
		"rgb(none 1.4 1.6)":            {0, 1, 2, 255},
		"rgb(1 2 3 / none)":            {1, 2, 3, 0},
		"rgba(255,0,0,0.5)":            {255, 0, 0, 128},
		"rgba( 255 , 0 , 0 , 50% )":    {255, 0, 0, 128},
		"rgb(255, 0, 0)":               {255, 0, 0, 255},
		"rgba(100%, 0%, 50%, 1)":       {255, 0, 128, 255},
		"  rgb(  10   20   30 / .5 ) ": {10, 20, 30, 128},
	} {
		c, err := Parse(in)
		if err != nil || c.Kind != Literal || c.RGBA != want {
			t.Errorf("%q = %+v %v, want %+v", in, c.RGBA, err, want)
		}
	}
}

func TestRGBMalformed(t *testing.T) {
	for _, in := range []string{
		"rgb(", "rgb()", "rgb(0 0)", "rgb(0 0 0 0)", "rgb(0 0 0 /)", "rgb(0 0 0 / 1 1)", "rgb(0 0 0 / x)",
		"rgb(0px 0 0)", "rgb(0 0 0", "rgb 0 0 0)", "rgbx(0 0 0)", "rgb(nan 0 0)", "rgb(inf 0 0)", "rgb(none% 0 0)",
		"rgb(0, 0 0)", "rgb(0 0 0, 1)", "rgb(0, 0, 0 / 1)", "rgb(0, 0)", "rgb(0, 0, 0, 1, 1)", "rgb(0, 0, 0,)",
		"rgb(,0, 0, 0)", "rgb(10%, 0, 0)", "rgb(none, 0, 0)", "rgba(0, 0, 0, none)", "rgb(0 0 0) x",
	} {
		c, err := Parse(in)
		if err == nil {
			t.Errorf("%q = %+v, want an error", in, c)
		}
	}
}
