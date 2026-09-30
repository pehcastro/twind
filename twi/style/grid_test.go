package style_test

import (
	"reflect"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/style"
)

func TestGridDefaults(t *testing.T) {
	sheet, err := style.NewSheet(konst.IRVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := sheet.Compute(style.ComputedStyle{}, nil)
	if got.Justify != style.JustifyStretch || got.AlignContent != style.JustifyStretch {
		t.Errorf("justify-content %d, align-content %d, want normal (stretch) for both", got.Justify, got.AlignContent)
	}
	if got.JustifyItems != style.AlignStretch || got.JustifySelf != style.AlignAuto {
		t.Errorf("justify-items %d, justify-self %d, want stretch and auto", got.JustifyItems, got.JustifySelf)
	}
	if got.GridColumns != nil || got.GridRows != nil || got.GridAutoColumns != nil || got.GridAutoRows != nil || got.GridColumn != (style.GridPlacement{}) || got.GridRow != (style.GridPlacement{}) || got.GridFlow != style.FlowRow {
		t.Errorf("grid defaults %+v %+v %+v %+v %+v %+v %d, want none, auto, row", got.GridColumns, got.GridRows, got.GridAutoColumns, got.GridAutoRows, got.GridColumn, got.GridRow, got.GridFlow)
	}
}

func TestGridApplies(t *testing.T) {
	fr := style.Track{Min: style.Breadth{Kind: style.SizeCells}, Max: style.Breadth{Kind: style.SizeFr, Value: 1}}
	content := style.Track{Min: style.Breadth{Kind: style.SizeMinContent}, Max: style.Breadth{Kind: style.SizeMaxContent}}
	sheet, err := style.NewSheet(konst.IRVersion, []style.Rule{
		{Class: "a", Decls: []style.Declaration{
			{Property: style.PropGridColumns, Tracks: []style.Track{fr, content}},
			{Property: style.PropGridRows, Tracks: []style.Track{content}},
			{Property: style.PropGridAutoColumns, Tracks: []style.Track{content, fr}},
			{Property: style.PropGridAutoRows, Tracks: []style.Track{fr}},
			{Property: style.PropGridColumnStart, GridLine: style.GridLine{Line: -1}},
			{Property: style.PropGridColumnEnd, GridLine: style.GridLine{Span: 3}},
			{Property: style.PropGridRowStart, GridLine: style.GridLine{Span: 2}},
			{Property: style.PropGridRowEnd, GridLine: style.GridLine{Line: 4}},
			{Property: style.PropGridFlow, Flow: style.FlowColumnDense},
			{Property: style.PropJustifyItems, Align: style.AlignEnd},
			{Property: style.PropJustifySelf, Align: style.AlignCenter},
			{Property: style.PropAlignContent, Justify: style.JustifyBetween},
		}},
		{Class: "b", Decls: []style.Declaration{
			{Property: style.PropGridColumns},
			{Property: style.PropGridColumnStart},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := sheet.Compute(style.ComputedStyle{}, []string{"a"})
	want := style.ComputedStyle{
		GridColumns:     []style.Track{fr, content},
		GridRows:        []style.Track{content},
		GridAutoColumns: []style.Track{content, fr},
		GridAutoRows:    []style.Track{fr},
		GridColumn:      style.GridPlacement{Start: style.GridLine{Line: -1}, End: style.GridLine{Span: 3}},
		GridRow:         style.GridPlacement{Start: style.GridLine{Span: 2}, End: style.GridLine{Line: 4}},
		GridFlow:        style.FlowColumnDense,
		JustifyItems:    style.AlignEnd,
		JustifySelf:     style.AlignCenter,
		AlignContent:    style.JustifyBetween,
	}
	pick := func(s style.ComputedStyle) style.ComputedStyle {
		return style.ComputedStyle{GridColumns: s.GridColumns, GridRows: s.GridRows, GridAutoColumns: s.GridAutoColumns, GridAutoRows: s.GridAutoRows, GridColumn: s.GridColumn, GridRow: s.GridRow, GridFlow: s.GridFlow, JustifyItems: s.JustifyItems, JustifySelf: s.JustifySelf, AlignContent: s.AlignContent}
	}
	if !reflect.DeepEqual(pick(got), want) {
		t.Errorf("grid properties\n got %+v\nwant %+v", pick(got), want)
	}
	later := sheet.Compute(style.ComputedStyle{}, []string{"a", "b"})
	if later.GridColumns != nil || later.GridColumn.Start != (style.GridLine{}) || later.GridColumn.End != (style.GridLine{Span: 3}) {
		t.Errorf("a later rule: columns %+v, placement %+v, want none and auto / span 3", later.GridColumns, later.GridColumn)
	}
}
