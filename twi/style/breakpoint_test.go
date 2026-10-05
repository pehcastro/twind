package style_test

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/style/testdata/responsive"
)

func TestBreakpointColumns(t *testing.T) {
	sheet, err := responsive.Styles()
	if err != nil {
		t.Fatal(err)
	}
	stack, swapped := strings.Fields(responsive.Stack), []string{"md:flex-row", "flex", "flex-col"}
	if got := sheet.Compute(style.ComputedStyle{}, stack).Direction; got != style.Column {
		t.Errorf("no viewport: direction %d, want column", got)
	}
	for _, tc := range []struct {
		cols      int
		direction style.Direction
		padding   float64
	}{
		{0, style.Column, 1},
		{79, style.Column, 1},
		{99, style.Column, 1},
		{100, style.Row, 2},
		{139, style.Row, 2},
		{140, style.Row, 4},
		{400, style.Row, 4},
	} {
		wide := sheet.WithColumns(tc.cols)
		if got := wide.Compute(style.ComputedStyle{}, stack).Direction; got != tc.direction {
			t.Errorf("%d columns: %s direction %d, want %d", tc.cols, responsive.Stack, got, tc.direction)
		}
		if got := wide.Compute(style.ComputedStyle{}, swapped).Direction; got != tc.direction {
			t.Errorf("%d columns: %v direction %d, want %d", tc.cols, swapped, got, tc.direction)
		}
		classes := strings.Fields(responsive.Padding + " " + responsive.Narrow)
		if got := wide.Compute(style.ComputedStyle{}, classes).Padding.Left.Value; got != tc.padding {
			t.Errorf("%d columns: %v padding %v, want %v", tc.cols, classes, got, tc.padding)
		}
	}
}

func TestBreakpointBands(t *testing.T) {
	sheet, err := responsive.Styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		from, to int
		crossed  bool
	}{
		{0, 99, false},
		{80, 99, false},
		{99, 100, true},
		{100, 99, true},
		{100, 139, false},
		{139, 140, true},
		{99, 200, true},
		{140, 1000, false},
	} {
		if got := sheet.Band(tc.from) != sheet.Band(tc.to); got != tc.crossed {
			t.Errorf("%d to %d columns: crossed %v, want %v", tc.from, tc.to, got, tc.crossed)
		}
	}
	for classes, want := range map[string]bool{
		"flex flex-col":        false,
		responsive.Stack:       true,
		responsive.Narrow:      true,
		"lg:p-4":               true,
		"hover:md:p-2 unknown": false,
		"":                     false,
	} {
		if got := sheet.Responsive(strings.Fields(classes)); got != want {
			t.Errorf("Responsive(%q) = %v, want %v", classes, got, want)
		}
	}
	universal, err := style.NewSheet(1, []style.Rule{{When: style.Condition{MinCols: 100}, Decls: []style.Declaration{{Property: style.PropBold, Flag: true}}}})
	if err != nil {
		t.Fatal(err)
	}
	if !universal.Responsive(nil) {
		t.Error("a universal breakpoint rule leaves an unclassed node unresponsive")
	}
	if !universal.WithColumns(100).Compute(style.ComputedStyle{}, nil).Bold || universal.WithColumns(99).Compute(style.ComputedStyle{}, nil).Bold {
		t.Error("a universal breakpoint rule ignores the viewport")
	}
}
