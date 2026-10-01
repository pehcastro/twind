package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func TestCellLookOneRowControlsDriven(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []theme.Scheme{theme.Dark, theme.Light} {
		th := zinc(t, scheme)
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			rt.SetTheme(th)
			in, invalid := NewInput(rt), NewInput(rt)
			in.Placeholder, invalid.Placeholder, invalid.Invalid = "m@example.com", "not an email", true
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col items-start gap-1 p-2 w-full h-full bg-background text-foreground"),
					Button(Outline, SizeDefault, twi.Text("Login with Google")),
					twi.Element(twi.Class("w-20"), in.Node()),
					twi.Element(twi.Class("w-20"), invalid.Node()),
					Badge(Outline, twi.Text("Outline")),
				)
			}
		}, drive.Size(80, 24), drive.Styles(sheet))
		f := d.Frame()
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		t.Logf("scheme %d, 80x24:\n%s", scheme, f.Text())
		lines, page := strings.Split(f.Text(), "\n"), th.Tokens[theme.Background].RGBA
		luma := func(c color.RGBA) int { return 299*int(c.R) + 587*int(c.G) + 114*int(c.B) }
		for _, label := range []string{"Login with Google", "m@example.com", "not an email", "Outline"} {
			y := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, label) })
			if y < 0 {
				t.Fatalf("scheme %d: no %q in the frame", scheme, label)
			}
			if strings.ContainsAny(lines[y], "│┃") {
				t.Errorf("scheme %d: %q, a side bar on a one-row control", scheme, lines[y])
			}
			if got := f.Cells().At(1, y).Bg.RGBA; got != page {
				t.Errorf("scheme %d %q: bg %v left of the control, want the page %v", scheme, label, got, page)
			}
			inside := f.Cells().At(2, y).Bg.RGBA
			t.Logf("scheme %d %q: bg %v, page %v", scheme, label, inside, page)
			switch label {
			case "Outline", "Login with Google":
				apart := 5000
				if scheme == theme.Dark && label == "Outline" {
					apart = 15000
				}
				if d := luma(inside) - luma(page); max(d, -d) < apart {
					t.Errorf("scheme %d %q: %v is too close to the page %v, want a visible shape", scheme, label, inside, page)
				}
				if secondary, d := th.Tokens[theme.Secondary].RGBA, luma(inside)-luma(page); max(d, -d) > max(luma(secondary)-luma(page), luma(page)-luma(secondary)) {
					t.Errorf("scheme %d %q: %v stands further from the page than the Secondary badge %v", scheme, label, inside, secondary)
				}
			case "not an email":
				if int(inside.R) < int(inside.G)+20 {
					t.Errorf("scheme %d: the invalid input %v, want its red ring blended over the fill", scheme, inside)
				}
			}
		}
	}
}
