package style_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/style/testdata/uikit"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

func TestLeadingUniversalOverridesInheritance(t *testing.T) {
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Decls: []style.Declaration{{Property: style.PropBold, Flag: true}, {Property: style.PropPaddingTop, Length: style.Length{Value: 2}}}},
		{Class: "p-1", Decls: []style.Declaration{{Property: style.PropPaddingTop, Length: style.Length{Value: 1}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := sheet.Compute(style.ComputedStyle{}, []string{"p-1"})
	if !got.Bold || got.Padding.Top.Value != 1 {
		t.Errorf("bold %v, padding %v: want the universal bold and the class padding", got.Bold, got.Padding.Top.Value)
	}
	if bare := sheet.Compute(style.ComputedStyle{}, nil); !bare.Bold || bare.Padding.Top.Value != 2 {
		t.Errorf("bold %v, padding %v: want the universal bold and padding", bare.Bold, bare.Padding.Top.Value)
	}
}

func TestComputeMatchesFixture(t *testing.T) {
	src, err := os.ReadFile("../tailwind/testdata/tailwind-4.3.3/app/output.css")
	if err != nil {
		t.Fatal(err)
	}
	app, _, err := tailwind.Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/identity.txt")
	if err != nil {
		t.Fatal(err)
	}
	parent := style.ComputedStyle{Color: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 9, G: 8, B: 7, A: 255}}, Bold: true, Underline: true, TextAlign: style.TextCenter, WhiteSpace: style.WhiteSpacePre}
	var got []string
	for _, source := range []struct {
		name  string
		rules []style.Rule
	}{{"app", app}, {"uikit", uikit.Rules()}} {
		sheet, err := style.NewSheet(konst.IRVersion, source.rules)
		if err != nil {
			t.Fatal(err)
		}
		var classes []string
		var attrs []style.Attr
		for _, r := range source.rules {
			for _, c := range []string{r.Class, r.Near.Class, r.Target.Class} {
				if c != "" && !slices.Contains(classes, c) {
					classes = append(classes, c)
				}
			}
			for _, a := range slices.Concat(r.When.Attrs, r.Near.Attrs, r.Target.Attrs) {
				if !slices.Contains(attrs, a) {
					attrs = append(attrs, a)
				}
			}
		}
		var shading []string
		for _, r := range source.rules {
			if r.Class != "" && !slices.Contains(shading, r.Class) && slices.ContainsFunc(r.Decls, func(d style.Declaration) bool {
				return slices.Contains([]style.Property{style.PropShadow, style.PropInsetShadow, style.PropShadowColor, style.PropInsetShadowColor, style.PropRingWidth, style.PropRingColor, style.PropRingInset, style.PropRingOffsetWidth, style.PropRingOffsetColor}, d.Property)
			}) {
				shading = append(shading, r.Class)
			}
		}
		states := []style.State{0, 1<<7 - 1}
		for bit := style.StateHover; bit <= style.StateChecked; bit <<= 1 {
			states = append(states, bit)
		}
		for _, th := range []*theme.Theme{nil, builtin(t, "zinc", theme.Light), builtin(t, "zinc", theme.Dark)} {
			for _, columns := range []int{40, 120, 240} {
				themed := sheet.WithTheme(th).WithColumns(columns)
				h := sha256.New()
				for _, states := range states {
					for _, node := range []style.NodeState{{States: states}, {States: states, Attrs: attrs}} {
						for i, c := range classes {
							one := []string{c}
							related := themed.Hands(one, node, themed.Near(one, nil))
							h.Write(fmt.Appendf(nil, "%#v\n%#v\n", themed.ComputeState(parent, one, node), themed.ComputeRelated(parent, nil, node, related)))
							trio := []string{classes[(i+2)%len(classes)], classes[(i+1)%len(classes)], c}
							h.Write(fmt.Appendf(nil, "%#v\n", themed.ComputeRelated(parent, trio, node, themed.Near(trio, nil))))
						}
					}
				}
				for _, node := range []style.NodeState{{}, {States: 1<<7 - 1, Attrs: attrs}} {
					for _, a := range shading {
						for _, b := range shading {
							got := themed.ComputeState(parent, []string{a, b}, node)
							h.Write(fmt.Appendf(nil, "%v %v\n", got.Shadows, got.InsetShadows))
						}
					}
				}
				name := "none"
				if th != nil {
					name = fmt.Sprint(th.Name, th.Scheme)
				}
				got = append(got, fmt.Sprintf("%s %s %d %x", source.name, name, columns, h.Sum(nil)))
			}
		}
	}
	if joined := strings.Join(got, "\n") + "\n"; joined != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("computed styles differ from the b170c89 fixture:\n%s", joined)
	}
}
