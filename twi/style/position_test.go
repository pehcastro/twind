package style_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
)

func TestPositionOf(t *testing.T) {
	first, last, odd, even := style.PlaceFirst, style.PlaceLast, style.PlaceOdd, style.PlaceEven
	for _, tc := range []struct {
		index, count int
		want         style.Place
	}{
		{0, 1, first | last | odd},
		{0, 3, first | odd},
		{1, 3, even},
		{2, 3, last | odd},
		{3, 4, last | even},
	} {
		if got := style.PlaceOf(tc.index, tc.count); got != tc.want {
			t.Errorf("PlaceOf(%d, %d) = %04b, want %04b", tc.index, tc.count, got, tc.want)
		}
	}
}

func TestPositionComputeThreeSiblings(t *testing.T) {
	sheet := appSheet(t)
	muted := sheet.Compute(style.ComputedStyle{}, []string{"bg-muted"}).Background
	type row struct {
		bottom, left float64
		background   color.Color
		padding      float64
	}
	classes := strings.Fields("border-b last:border-0 first:border-l odd:bg-muted only:p-1 not-first:p-1")
	for count, want := range map[int][]row{
		3: {
			{bottom: 1, left: 1, background: muted},
			{bottom: 1, padding: 1},
			{bottom: 0, background: muted, padding: 1},
		},
		1: {{bottom: 0, left: 0, background: muted, padding: 1}},
	} {
		for i, w := range want {
			got := sheet.ComputeState(style.ComputedStyle{}, classes, style.NodeState{Places: style.PlaceOf(i, count)})
			if g := (row{got.BorderWidth.Bottom.Value, got.BorderWidth.Left.Value, got.Background, got.Padding.Top.Value}); g != w {
				t.Errorf("child %d of %d: %+v, want %+v", i+1, count, g, w)
			}
		}
	}
	got := sheet.ComputeState(style.ComputedStyle{}, classes, style.NodeState{})
	if unknown := (row{got.BorderWidth.Bottom.Value, got.BorderWidth.Left.Value, got.Background, got.Padding.Top.Value}); unknown != (row{bottom: 1}) {
		t.Errorf("no position given: %+v, want only border-b: an unknown place passes no positional test, negated or not", unknown)
	}
}

func TestPositionNegatedStateAndAttribute(t *testing.T) {
	sheet := appSheet(t)
	for _, tc := range []struct {
		classes string
		node    style.NodeState
		want    style.Display
	}{
		{"not-data-[state=open]:hidden", style.NodeState{}, style.DisplayNone},
		{"not-data-[state=open]:hidden", style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "closed"}}}, style.DisplayNone},
		{"not-data-[state=open]:hidden", style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "open"}}}, style.DisplayBlock},
	} {
		if got := sheet.ComputeState(style.ComputedStyle{}, strings.Fields(tc.classes), tc.node).Display; got != tc.want {
			t.Errorf("%s with %+v: display %v, want %v", tc.classes, tc.node, got, tc.want)
		}
	}
	for states, want := range map[style.State]float64{0: 0.5, style.StateDisabled: 1, style.StateHover: 0.5} {
		if got := sheet.ComputeState(style.ComputedStyle{}, []string{"not-disabled:opacity-50"}, style.NodeState{States: states}).Opacity; got != want {
			t.Errorf("not-disabled:opacity-50 with states %07b: opacity %v, want %v", states, got, want)
		}
	}
}

func TestPositionInRelationalRules(t *testing.T) {
	sheet := appSheet(t)
	first, last := style.NodeState{Places: style.PlaceOf(0, 3)}, style.NodeState{Places: style.PlaceOf(2, 3)}
	target := func(class string) *style.Match {
		hands := sheet.Hands([]string{class}, style.NodeState{}, nil)
		if len(hands) != 1 {
			t.Fatalf("%s handed %v", class, hands)
		}
		return &sheet.Rule(hands[0]).Target
	}
	if m := target("*:last:border-b-0"); !m.Accepts(style.ElementAny, 0, last) || m.Accepts(style.ElementAny, 0, first) || m.Accepts(style.ElementAny, 0, style.NodeState{}) {
		t.Errorf("*:last: must accept only the last child")
	}
	if m := target("[&>*:not(:first-child)]:p-1"); m.Accepts(style.ElementAny, 0, first) || !m.Accepts(style.ElementAny, 0, last) || m.Accepts(style.ElementAny, 0, style.NodeState{}) {
		t.Errorf("[&>*:not(:first-child)] must accept a later child only")
	}
	anchor := []string{"[&:last-child[data-selected=true]_button]:border-r"}
	selected := []style.Attr{{Name: "data-selected", Value: "true"}}
	if got := sheet.Hands(anchor, style.NodeState{Attrs: selected, Places: style.PlaceOf(0, 3)}, nil); len(got) != 0 {
		t.Errorf("a first anchor handed %v down", got)
	}
	if got := sheet.Hands(anchor, style.NodeState{Attrs: selected, Places: style.PlaceOf(2, 3)}, nil); len(got) != 1 {
		t.Errorf("the last anchor handed %v down, want one rule", got)
	}
	near := sheet.Rule(sheet.Near([]string{"group-first:p-1"}, nil)[0]).Near
	if group := sheet.Marks([]string{"group"}); !near.Accepts(style.ElementAny, group, first) || near.Accepts(style.ElementAny, group, last) {
		t.Errorf("group-first: must test the group's place")
	}
}

func TestPseudoElementPlaceholder(t *testing.T) {
	sheet := appSheet(t)
	classes := strings.Fields("text-foreground placeholder:text-muted-foreground")
	foreground := sheet.Compute(style.ComputedStyle{}, []string{"text-foreground"}).Color
	muted := sheet.Compute(style.ComputedStyle{}, []string{"text-muted-foreground"}).Color
	node := sheet.Compute(style.ComputedStyle{}, classes)
	if node.Color != foreground {
		t.Errorf("the input's own text %+v, want foreground %+v: a placeholder rule styled the node", node.Color, foreground)
	}
	if got := sheet.ComputePart(style.PartPlaceholder, node, classes, style.NodeState{}, nil).Color; got != muted {
		t.Errorf("placeholder %+v, want muted %+v", got, muted)
	}
	hover := []string{"text-foreground", "hover:placeholder:text-muted-foreground"}
	for states, want := range map[style.State]color.Color{0: foreground, style.StateHover: muted} {
		if got := sheet.ComputePart(style.PartPlaceholder, node, hover, style.NodeState{States: states}, nil).Color; got != want {
			t.Errorf("hover:placeholder with states %07b: %+v, want %+v", states, got, want)
		}
	}
	if got := sheet.ComputePart(style.PartSelection, node, classes, style.NodeState{}, nil); got.Color != foreground || got.Background.Kind != color.Unset {
		t.Errorf("selection of a node with only placeholder rules: %+v %+v, want the node's colour and no background", got.Color, got.Background)
	}
}

func TestPseudoElementSelection(t *testing.T) {
	sheet := appSheet(t)
	classes := []string{"selection:bg-primary", "selection:text-primary-foreground"}
	bg := sheet.Compute(style.ComputedStyle{}, []string{"bg-primary"}).Background
	fg := sheet.Compute(style.ComputedStyle{}, []string{"text-primary-foreground"}).Color
	node := sheet.Compute(style.ComputedStyle{}, classes)
	if node.Background.Kind != color.Unset || node.Color.Kind != color.Unset {
		t.Errorf("selection rules styled the node: %+v %+v", node.Background, node.Color)
	}
	if got := sheet.ComputePart(style.PartSelection, node, classes, style.NodeState{}, nil); got.Background != bg || got.Color != fg {
		t.Errorf("own selection %+v on %+v, want %+v on %+v", got.Color, got.Background, fg, bg)
	}
	hands := sheet.Hands(classes, style.NodeState{}, nil)
	if len(hands) != 2 {
		t.Fatalf("selection: handed %v to descendants, want the two descendant rules", hands)
	}
	child := sheet.ComputeRelated(node, nil, style.NodeState{}, hands)
	if child.Background.Kind != color.Unset || child.Color.Kind != color.Unset {
		t.Errorf("a descendant's box took its ancestor's selection: %+v %+v", child.Background, child.Color)
	}
	if got := sheet.ComputePart(style.PartSelection, child, nil, style.NodeState{}, hands); got.Background != bg || got.Color != fg {
		t.Errorf("descendant selection %+v on %+v, want %+v on %+v", got.Color, got.Background, fg, bg)
	}
	if got := sheet.ComputePart(style.PartPlaceholder, child, nil, style.NodeState{}, hands); got.Background.Kind != color.Unset {
		t.Errorf("a selection rule reached the placeholder: %+v", got.Background)
	}
}
