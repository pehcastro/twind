package tailwind

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

const cssFixtures = "../css/testdata/tailwind-4.3.3/"

func compileFixture(t *testing.T, dir string) (style.Sheet, []Warning) {
	t.Helper()
	src, err := os.ReadFile(dir + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, warnings, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	return sheet, warnings
}

func cells(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }

func all(l style.Length) style.Edges { return style.Edges{Top: l, Right: l, Bottom: l, Left: l} }

func TestIntegrationHello(t *testing.T) {
	sheet, warnings := compileFixture(t, cssFixtures+"hello")
	for _, w := range warnings {
		t.Errorf("warning: %s", w)
	}
	got := sheet.Compute(style.ComputedStyle{}, strings.Fields("flex flex-col gap-2 p-4 border rounded-lg bg-zinc-950 text-zinc-100"))
	t.Logf("Display=%d Direction=%d RowGap=%v ColumnGap=%v Padding=%v", got.Display, got.Direction, got.RowGap, got.ColumnGap, got.Padding)
	t.Logf("BorderWidth=%v BorderStyle=%d BorderColor=%+v Radius=%d", got.BorderWidth, got.BorderStyle, got.BorderColor, got.Radius)
	t.Logf("Background=%+v Color=%+v", got.Background, got.Color)
	if got.Display != style.DisplayFlex || got.Direction != style.Column {
		t.Errorf("display %d direction %d, want flex column", got.Display, got.Direction)
	}
	if got.RowGap != cells(2) || got.ColumnGap != cells(2) {
		t.Errorf("gap %v %v, want 2 cells", got.RowGap, got.ColumnGap)
	}
	if got.Padding != all(cells(4)) {
		t.Errorf("padding %v, want 4 cells on all sides", got.Padding)
	}
	if got.BorderWidth != all(cells(1)) || got.BorderStyle != style.BorderSingle || got.BorderColor.Kind != color.Current {
		t.Errorf("border %v %d %+v, want 1 cell, single, currentcolor", got.BorderWidth, got.BorderStyle, got.BorderColor)
	}
	if got.Radius != style.RadiusLg {
		t.Errorf("radius %d, want lg", got.Radius)
	}
	if want := (color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 9, G: 9, B: 11, A: 255}}); got.Background != want {
		t.Errorf("background %+v, want %+v", got.Background, want)
	}
	if want := (color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 244, G: 244, B: 245, A: 255}}); got.Color != want {
		t.Errorf("color %+v, want %+v", got.Color, want)
	}
}

func TestCascadeSourceOrder(t *testing.T) {
	sheet, _ := compileFixture(t, cssFixtures+"hello")
	for _, classes := range []string{"p-4 p-2", "p-2 p-4"} {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes))
		if got.Padding != all(cells(4)) {
			t.Errorf("%q: padding %v, want 4 cells: .p-4 follows .p-2 in the sheet", classes, got.Padding)
		}
	}
}

func TestCascadeBaseReset(t *testing.T) {
	sheet, _ := compileFixture(t, cssFixtures+"hello")
	got := sheet.Compute(style.ComputedStyle{}, nil)
	if got.BorderStyle != style.BorderSingle || got.BorderWidth != all(cells(0)) || got.Padding != all(cells(0)) || got.Margin != all(cells(0)) {
		t.Errorf("no classes: %+v, want the * reset: border 0 solid, no padding, no margin", got)
	}
}

func TestCascadeInheritance(t *testing.T) {
	sheet, _ := compileFixture(t, cssFixtures+"matrix")
	parent := sheet.Compute(style.ComputedStyle{}, strings.Fields("text-zinc-100 font-bold italic underline px-2 text-center"))
	child := sheet.Compute(parent, nil)
	if child.Color != parent.Color || !child.Bold || !child.Italic || !child.Underline || child.TextAlign != style.TextCenter {
		t.Errorf("child %+v lost inherited text attributes of %+v", child, parent)
	}
	if child.Padding != all(cells(0)) {
		t.Errorf("child padding %v, padding is not inherited", child.Padding)
	}
	if child.Bold != parent.Bold || child.Strikethrough {
		t.Errorf("child bold %v strike %v", child.Bold, child.Strikethrough)
	}
	struck := sheet.Compute(parent, []string{"line-through"})
	if struck.Underline || !struck.Strikethrough {
		t.Errorf("line-through after underline: underline %v strike %v, want only strike", struck.Underline, struck.Strikethrough)
	}
}

func TestCascadeMatrix(t *testing.T) {
	sheet, _ := compileFixture(t, cssFixtures+"matrix")
	got := sheet.Compute(style.ComputedStyle{}, strings.Fields(
		"flex flex-row grow shrink-0 basis-4 items-center self-end justify-between gap-1 w-10 h-full min-w-0 max-w-80 min-h-1 max-h-10 m-1 absolute inset-0 top-1 right-2 left-3 overflow-hidden overflow-x-auto z-10 opacity-50 translate-x-1 cursor-pointer select-none invisible border-2 border-dashed border-zinc-700 rounded-full hover:bg-zinc-800 data-[selected=true]:bg-zinc-800 md:flex-row"))
	want := style.ComputedStyle{
		Display: style.DisplayFlex, Direction: style.Row, Grow: 1, Shrink: 0, Basis: cells(4),
		AlignItems: style.AlignCenter, AlignSelf: style.AlignEnd, Justify: style.JustifyBetween,
		JustifyItems: style.AlignStretch, AlignContent: style.JustifyStretch,
		RowGap: cells(1), ColumnGap: cells(1),
		Width: cells(10), Height: style.Length{Unit: style.Percent, Value: 100},
		MinWidth: cells(0), MaxWidth: cells(80), MinHeight: cells(1), MaxHeight: cells(10),
		Margin: all(cells(1)), Position: style.PositionAbsolute,
		Inset:     style.Edges{Top: cells(1), Right: cells(2), Bottom: cells(0), Left: cells(3)},
		OverflowX: style.OverflowAuto, OverflowY: style.OverflowHidden, ZIndex: 10,
		TranslateX: cells(1), ScaleX: 1, ScaleY: 1,
		BorderWidth: all(cells(2)), BorderStyle: style.BorderDashed,
		BorderColor: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 63, G: 63, B: 70, A: 255}},
		Radius:      style.RadiusFull, Opacity: 0.5,
		Visibility: style.Hidden, Cursor: style.CursorPointer, UserSelect: style.SelectNone,
		Gradient:   style.Gradient{From: style.GradientStop{Color: color.Color{Kind: color.Literal}}, Via: style.GradientStop{Color: color.Color{Kind: color.Literal}, Position: 0.5}, To: style.GradientStop{Color: color.Color{Kind: color.Literal}, Position: 1}},
		Ring:       style.Ring{Color: color.Color{Kind: color.Current}, OffsetColor: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255}}},
		Transition: style.Transition{Properties: style.TransitionAll, Easing: style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}},
		Animation:  style.Animation{Iterations: 1, Easing: style.Easing{X1: 0.25, Y1: 0.1, X2: 0.25, Y2: 1}, Enter: still(), Exit: still()},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("matrix classes:\n got %+v\nwant %+v", got, want)
	}
	alpha := sheet.Compute(style.ComputedStyle{}, []string{"bg-zinc-950/90"})
	if want := (color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 9, G: 9, B: 11, A: 230}}); alpha.Background != want {
		t.Errorf("bg-zinc-950/90: %+v, want %+v", alpha.Background, want)
	}
	arbitrary := sheet.Compute(style.ComputedStyle{}, []string{"w-[37px]"})
	if arbitrary.Width != cells(37) {
		t.Errorf("w-[37px]: %v, want 37 cells", arbitrary.Width)
	}
}

func TestIntegrationMatrixWarnings(t *testing.T) {
	_, warnings := compileFixture(t, cssFixtures+"matrix")
	var got []string
	for _, w := range warnings {
		t.Log(w)
		got = append(got, w.Class+" "+w.Category.String()+" "+w.Reason)
	}
	want := []string{"blur-sm unsupported no terminal rendering", "md:flex-row unsupported breakpoint 48rem is not a whole number of cells"}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("warnings %q, want %q", got, want)
	}
}

func TestConditionsKept(t *testing.T) {
	src, err := os.ReadFile("../css/testdata/tailwind-4.3.3/matrix/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	conditions := map[string]style.Condition{}
	for _, r := range rules {
		conditions[r.Class] = r.When
	}
	if c := conditions["hover:bg-zinc-800"]; c.States != style.StateHover {
		t.Errorf("hover: %+v", c)
	}
	if c := conditions["data-[selected=true]:bg-zinc-800"]; !slices.Equal(c.Attrs, []style.Attr{{Name: "data-selected", Value: "true"}}) {
		t.Errorf("data attribute: %+v", c)
	}
}

func TestCompileKeepsHalfCells(t *testing.T) {
	rules, warnings, err := Compile("@layer utilities { .p-half { padding: 0.5px; } .h-x { height: calc(var(--s, 1px) * 1.5); } .b-half { border-top-width: 0.5px; } }")
	if err != nil || len(warnings) != 0 {
		t.Fatalf("half cells: error %v warnings %v, want none: rounding is the layout's, per terminal", err, warnings)
	}
	sheet, err := style.NewSheet(1, rules)
	if err != nil {
		t.Fatal(err)
	}
	got := sheet.Compute(style.ComputedStyle{}, []string{"p-half", "h-x", "b-half"})
	if half := (style.Length{Unit: style.Cells, Value: 0.5}); got.Padding.Top != half || got.Padding.Left != half || got.BorderWidth.Top != half || got.Height != (style.Length{Unit: style.Cells, Value: 1.5}) {
		t.Errorf("padding %+v, border top %+v, height %+v, want 0.5, 0.5 and 1.5 cells", got.Padding, got.BorderWidth.Top, got.Height)
	}
}

func TestCompileRejects(t *testing.T) {
	cases := map[string]string{
		"rem length":       ".w-4 { width: 1rem; }",
		"mixed calc":       ".w-x { width: calc(100% - 2px); }",
		"undefined var":    ".p-x { padding: var(--nope); }",
		"adjacent sibling": ".a + .x { color: #fff; }",
		"unknown value":    ".d-x { display: table; }",
		"unknown prop":     ".s-x { mask-image: none; }",
		"percent shadow":   ".s-x { box-shadow: 10% 1px #000; }",
		"two colours":      ".s-x { box-shadow: 1px 1px #000 #fff; }",
		"two ring shadows": ".r-x { --tw-ring-shadow: 0 0 0 2px #000, 0 0 0 1px #fff; box-shadow: var(--tw-ring-shadow); }",
		"inset ring":       ".r-x { --tw-inset-ring-shadow: inset 0 0 0 1px #000; box-shadow: var(--tw-inset-ring-shadow); }",
	}
	for name, src := range cases {
		_, warnings, err := Compile("@layer utilities { " + src + " }")
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(warnings) != 1 {
			t.Errorf("%s: %d warnings %v, want 1", name, len(warnings), warnings)
			continue
		}
		t.Logf("%s: %s", name, warnings[0])
	}
}
