package tailwind

import (
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/style"
)

func TestGridCompiles(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	fr, auto, zero := style.Breadth{Kind: style.SizeFr, Value: 1}, style.Breadth{}, style.Breadth{Kind: style.SizeCells}
	share := style.Track{Min: zero, Max: fr}
	flexible, automatic := style.Track{Min: auto, Max: fr}, style.Track{}
	minContent := style.Track{Min: style.Breadth{Kind: style.SizeMinContent}, Max: style.Breadth{Kind: style.SizeMinContent}}
	maxContent := style.Track{Min: style.Breadth{Kind: style.SizeMaxContent}, Max: style.Breadth{Kind: style.SizeMaxContent}}
	type get func(style.ComputedStyle) any
	columns := func(s style.ComputedStyle) any { return s.GridColumns }
	rows := func(s style.ComputedStyle) any { return s.GridRows }
	autoRows := func(s style.ComputedStyle) any { return s.GridAutoRows }
	autoColumns := func(s style.ComputedStyle) any { return s.GridAutoColumns }
	column := func(s style.ComputedStyle) any { return s.GridColumn }
	row := func(s style.ComputedStyle) any { return s.GridRow }
	flow := func(s style.ComputedStyle) any { return s.GridFlow }
	items := func(s style.ComputedStyle) any { return [2]style.Align{s.AlignItems, s.JustifyItems} }
	self := func(s style.ComputedStyle) any { return [2]style.Align{s.AlignSelf, s.JustifySelf} }
	content := func(s style.ComputedStyle) any { return [2]style.Justify{s.AlignContent, s.Justify} }
	gaps := func(s style.ComputedStyle) any { return [2]style.Length{s.RowGap, s.ColumnGap} }
	for _, tc := range []struct {
		classes string
		field   get
		want    any
	}{
		{"grid", func(s style.ComputedStyle) any { return s.Display }, style.DisplayGrid},
		{"grid-cols-1", columns, []style.Track{share}},
		{"grid-cols-2", columns, []style.Track{share, share}},
		{"grid-cols-3", columns, []style.Track{share, share, share}},
		{"grid-cols-3 grid-cols-none", columns, []style.Track(nil)},
		{"grid-cols-[1fr_auto]", columns, []style.Track{flexible, automatic}},
		{"grid-cols-[0_1fr]", columns, []style.Track{{Min: zero, Max: zero}, flexible}},
		{"grid-cols-[calc(var(--spacing)*4)_1fr]", columns, []style.Track{{Min: style.Breadth{Kind: style.SizeCells, Value: 4}, Max: style.Breadth{Kind: style.SizeCells, Value: 4}}, flexible}},
		{"grid-cols-[minmax(0,1fr)_max-content]", columns, []style.Track{share, maxContent}},
		{"grid-cols-[min-content_1fr]", columns, []style.Track{minContent, flexible}},
		{"grid-rows-[auto_auto]", rows, []style.Track{automatic, automatic}},
		{"grid-rows-[auto_1fr]", rows, []style.Track{automatic, flexible}},
		{"grid-rows-[auto_auto_1fr]", rows, []style.Track{automatic, automatic, flexible}},
		{"grid-rows-2", rows, []style.Track{share, share}},
		{"auto-rows-min", autoRows, []style.Track{minContent}},
		{"auto-rows-max", autoRows, []style.Track{maxContent}},
		{"auto-rows-fr", autoRows, []style.Track{share}},
		{"auto-rows-auto", autoRows, []style.Track{automatic}},
		{"auto-cols-min", autoColumns, []style.Track{minContent}},
		{"auto-cols-fr", autoColumns, []style.Track{share}},
		{"col-span-2", column, style.GridPlacement{Start: style.GridLine{Span: 2}, End: style.GridLine{Span: 2}}},
		{"col-span-full", column, style.GridPlacement{Start: style.GridLine{Line: 1}, End: style.GridLine{Line: -1}}},
		{"col-start-2", column, style.GridPlacement{Start: style.GridLine{Line: 2}}},
		{"col-end-3", column, style.GridPlacement{End: style.GridLine{Line: 3}}},
		{"col-span-2 col-start-2", column, style.GridPlacement{Start: style.GridLine{Line: 2}, End: style.GridLine{Span: 2}}},
		{"col-auto", column, style.GridPlacement{}},
		{"col-start-2 col-auto", column, style.GridPlacement{Start: style.GridLine{Line: 2}}},
		{"row-span-2", row, style.GridPlacement{Start: style.GridLine{Span: 2}, End: style.GridLine{Span: 2}}},
		{"row-start-1", row, style.GridPlacement{Start: style.GridLine{Line: 1}}},
		{"row-end-3", row, style.GridPlacement{End: style.GridLine{Line: 3}}},
		{"col-start-2 row-span-2 row-start-1", row, style.GridPlacement{Start: style.GridLine{Line: 1}, End: style.GridLine{Span: 2}}},
		{"row-start-1 row-auto", row, style.GridPlacement{Start: style.GridLine{Line: 1}}},
		{"grid-flow-col", flow, style.FlowColumn},
		{"grid-flow-col grid-flow-row", flow, style.FlowRow},
		{"grid-flow-dense", flow, style.FlowRowDense},
		{"grid-flow-row-dense", flow, style.FlowRowDense},
		{"", items, [2]style.Align{style.AlignStretch, style.AlignStretch}},
		{"justify-items-start", items, [2]style.Align{style.AlignStretch, style.AlignStart}},
		{"justify-items-end", items, [2]style.Align{style.AlignStretch, style.AlignEnd}},
		{"justify-items-center", items, [2]style.Align{style.AlignStretch, style.AlignCenter}},
		{"justify-items-start justify-items-stretch", items, [2]style.Align{style.AlignStretch, style.AlignStretch}},
		{"place-items-center", items, [2]style.Align{style.AlignCenter, style.AlignCenter}},
		{"place-items-start", items, [2]style.Align{style.AlignStart, style.AlignStart}},
		{"items-start", items, [2]style.Align{style.AlignStart, style.AlignStretch}},
		{"", self, [2]style.Align{style.AlignAuto, style.AlignAuto}},
		{"self-start justify-self-end", self, [2]style.Align{style.AlignStart, style.AlignEnd}},
		{"justify-self-start", self, [2]style.Align{style.AlignAuto, style.AlignStart}},
		{"justify-self-center", self, [2]style.Align{style.AlignAuto, style.AlignCenter}},
		{"justify-self-stretch", self, [2]style.Align{style.AlignAuto, style.AlignStretch}},
		{"justify-self-end justify-self-auto", self, [2]style.Align{style.AlignAuto, style.AlignEnd}},
		{"justify-self-auto", self, [2]style.Align{style.AlignAuto, style.AlignAuto}},
		{"place-self-center", self, [2]style.Align{style.AlignCenter, style.AlignCenter}},
		{"", content, [2]style.Justify{style.JustifyStretch, style.JustifyStretch}},
		{"place-content-center", content, [2]style.Justify{style.JustifyCenter, style.JustifyCenter}},
		{"content-center", content, [2]style.Justify{style.JustifyCenter, style.JustifyStretch}},
		{"gap-2", gaps, [2]style.Length{cells(2), cells(2)}},
		{"gap-x-6", gaps, [2]style.Length{{}, cells(6)}},
		{"gap-x-3 gap-y-1", gaps, [2]style.Length{cells(1), cells(3)}},
		{"gap-y-0.5", gaps, [2]style.Length{cells(0.5), {}}},
		{"gap-1.5", gaps, [2]style.Length{cells(1.5), cells(1.5)}},
	} {
		if got := tc.field(sheet.Compute(style.ComputedStyle{}, strings.Fields(tc.classes))); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: %+v, want %+v", tc.classes, got, tc.want)
		}
	}
}

func TestGridWarnings(t *testing.T) {
	want := map[string]Category{
		"inline-grid":       Approximated,
		"grid-cols-subgrid": Unsupported,
		"grid-cols-[repeat(auto-fill,minmax(10px,1fr))]": Unsupported,
	}
	got := appWarnings(t, "")
	for class, c := range want {
		if got[class] != c {
			t.Errorf("%s: warned %v, want %s", class, got[class], c)
		}
	}
	quiet(t, func(class string) bool {
		_, expected := want[class]
		return !expected && hasPrefix("grid", "col-", "row-", "auto-rows", "auto-cols", "justify-", "place-", "content-", "gap-")(class)
	})
	for name, src := range map[string]string{
		"a zero line":            ".a { grid-column-start: 0; }",
		"a named line":           ".a { grid-row: header; }",
		"a span of zero":         ".a { grid-column: span 0 / auto; }",
		"fr as a minimum":        ".a { grid-template-columns: minmax(1fr, 2fr); }",
		"fit-content()":          ".a { grid-template-columns: fit-content(10px) 1fr; }",
		"line names":             ".a { grid-template-columns: [start] 1fr [end]; }",
		"three values in a line": ".a { grid-column: 1 / 2 / 3; }",
		"legacy justify-items":   ".a { justify-items: legacy; }",
		"too many repeats":       ".a { grid-template-columns: repeat(100000, 1fr); }",
	} {
		if _, warnings, err := Compile("@layer utilities { " + src + " }"); err != nil || len(warnings) != 1 || warnings[0].Category != Unsupported {
			t.Errorf("%s: error %v warnings %v, want one unsupported warning", name, err, warnings)
		}
	}
}
