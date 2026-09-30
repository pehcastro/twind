package style_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/style"
)

func corners(r style.Radius) [4]style.Radius {
	return [4]style.Radius{r.At(style.CornerTopLeft), r.At(style.CornerTopRight), r.At(style.CornerBottomRight), r.At(style.CornerBottomLeft)}
}

func TestRadiusCascade(t *testing.T) {
	sheet := appSheet(t)
	none, sm, md, lg, full := style.RadiusNone, style.RadiusSm, style.RadiusMd, style.RadiusLg, style.RadiusFull
	for _, tc := range []struct {
		classes string
		want    [4]style.Radius
	}{
		{"rounded-md", [4]style.Radius{md, md, md, md}},
		{"rounded-md rounded-t-xl", [4]style.Radius{lg, lg, md, md}},
		{"rounded-t-xl rounded-md", [4]style.Radius{lg, lg, md, md}},
		{"rounded-b-none rounded-md", [4]style.Radius{md, md, none, none}},
		{"rounded-tl-lg rounded-br-full rounded-bl-sm", [4]style.Radius{lg, none, full, sm}},
		{"rounded-ss-sm rounded-se-md rounded-ee-full rounded-es-lg", [4]style.Radius{sm, md, full, lg}},
		{"rounded-s-md rounded-e-lg", [4]style.Radius{md, lg, lg, md}},
		{"rounded-md rounded-tr-none", [4]style.Radius{md, none, md, md}},
	} {
		if got := corners(sheet.Compute(style.ComputedStyle{}, strings.Fields(tc.classes)).Radius); got != tc.want {
			t.Errorf("%s: %v, want %v (top left, top right, bottom right, bottom left)", tc.classes, got, tc.want)
		}
	}
	if got := sheet.Compute(style.ComputedStyle{}, []string{"rounded-md"}).Radius; got != style.RadiusMd {
		t.Errorf("rounded-md is %d, want the uniform RadiusMd %d", got, style.RadiusMd)
	}
}

func TestRadiusButtonGroup(t *testing.T) {
	sheet := appSheet(t)
	none, md := style.RadiusNone, style.RadiusMd
	for i, want := range [3][4]style.Radius{{md, none, none, md}, {}, {none, md, md, none}} {
		node := style.NodeState{Places: style.PlaceOf(i, 3)}
		if got := corners(sheet.ComputeState(style.ComputedStyle{}, strings.Fields("first:rounded-l-md last:rounded-r-md"), node).Radius); got != want {
			t.Errorf("button %d of 3: %v, want %v", i, got, want)
		}
	}
}
