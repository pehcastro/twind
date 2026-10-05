package twi_test

import (
	"slices"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/runtime/testdata/relate"
	"github.com/pehcastro/twind/twi/tailwind"
)

func TestElementTagSizesAnSvgChild(t *testing.T) {
	sheet, err := relate.Styles()
	if err != nil {
		t.Fatal(err)
	}
	var frame []string
	for _, row := range decode(t, twi.RenderString(relate.Icons(), twi.Styles(sheet), twi.Width(12), twi.ColorProfile(color.TrueColor))) {
		var line strings.Builder
		for _, c := range row {
			switch {
			case c.glyph != " ":
				line.WriteString(c.glyph)
			case c.bg != "":
				line.WriteString("#")
			default:
				line.WriteString(".")
			}
		}
		frame = append(frame, line.String())
	}
	got := strings.Join(frame, "\n")
	t.Logf("[&>svg]:size-4 over a tagged svg, a span holding a tagged svg, an element and a text, svg background as #:\n%s", got)
	if want := "★###.☆.s.t\n####\n####\n####"; got != want {
		t.Errorf("got\n%s\nwant the child svg 4 by 4, the grandchild svg, the element and the text 1 by 1:\n%s", got, want)
	}
}

func TestElementClassesSplitsTheCallersClasses(t *testing.T) {
	options := []twi.NodeOption{twi.Class("flex  gap-1"), twi.Data("state", "open"), twi.Element(twi.Class("child"), twi.Text("x")), twi.Class("p-1"), twi.Text("y")}
	classes, rest := twi.Classes(options)
	if want := []string{"flex", "gap-1", "p-1"}; !slices.Equal(classes, want) {
		t.Errorf("classes %q, want %q", classes, want)
	}
	if len(rest) != 3 {
		t.Fatalf("%d other options, want the attribute and two children", len(rest))
	}
	if _, child := rest[1].(twi.Node); !child {
		t.Errorf("rest[1] is %T, want the child element in its place", rest[1])
	}
	sheet, err := relate.Styles()
	if err != nil {
		t.Fatal(err)
	}
	render := func(options ...twi.NodeOption) string {
		return twi.RenderString(twi.Element(options...), twi.Styles(sheet), twi.Width(10), twi.ColorProfile(color.None))
	}
	if merged, whole := render(append([]twi.NodeOption{twi.Class(classes...)}, rest...)...), render(options...); merged != whole {
		t.Errorf("the split options render %q, the originals %q", merged, whole)
	}
}

func TestElementFixtureFresh(t *testing.T) {
	stale, err := tailwind.Stale("runtime/testdata/relate", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twi/runtime/testdata/relate IR is stale: run go generate there")
	}
}
