package style_test

import (
	"strconv"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
)

func TestStructuralNeedsATree(t *testing.T) {
	sheet := appSheet(t)
	open := style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "open"}}}
	for _, class := range []string{"group-data-[state=open]:bg-accent", "has-data-[slot=card-action]:grid-cols-[1fr_auto]", "peer-data-[state=checked]:bg-primary", "[&>*]:p-1", "*:p-2"} {
		got := sheet.ComputeState(style.ComputedStyle{}, []string{class}, open)
		if got.Background.Kind != color.Unset || got.GridColumns != nil || got.Padding.Top.Value != 0 {
			t.Errorf("%s applied without a tree: %+v", class, got)
		}
	}
}

func TestStructuralNear(t *testing.T) {
	sheet := appSheet(t)
	open := style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: "open"}}}
	accent := rgba(244, 244, 245, 255)
	for _, tc := range []struct {
		classes  string
		ancestor []string
		node     style.NodeState
		want     color.Color
	}{
		{"group-data-[state=open]:bg-accent", []string{"group"}, open, accent},
		{"group-data-[state=open]:bg-accent", []string{"group"}, style.NodeState{}, color.Color{}},
		{"group-data-[state=open]:bg-accent", []string{"group/item"}, open, color.Color{}},
		{"group-data-[state=open]:bg-accent", []string{"peer"}, open, color.Color{}},
		{"group-data-[state=open]/item:bg-muted", []string{"group/item"}, open, accent},
		{"group-data-[state=open]/item:bg-muted", []string{"group"}, open, color.Color{}},
		{"group-data-[state=open]:bg-accent bg-card", []string{"group"}, open, accent},
	} {
		near := sheet.Near(strings.Fields(tc.classes), nil)
		if len(near) == 0 {
			t.Fatalf("%s: no near rule", tc.classes)
		}
		var related []int
		for _, i := range near {
			if sheet.Rule(i).Near.Accepts(style.ElementAny, sheet.Marks(tc.ancestor), tc.node) {
				related = append(related, i)
			}
		}
		if got := sheet.ComputeRelated(style.ComputedStyle{}, strings.Fields(tc.classes), style.NodeState{}, related).Background; got != tc.want {
			t.Errorf("%s under %v with %+v: background %+v, want %+v", tc.classes, tc.ancestor, tc.node, got, tc.want)
		}
	}
}

func TestStructuralHands(t *testing.T) {
	sheet := appSheet(t)
	hover := style.NodeState{States: style.StateHover}
	anchor := []string{"hover:[&>svg]:shrink-0"}
	if got := sheet.Hands(anchor, style.NodeState{}, nil); len(got) != 0 {
		t.Errorf("hover:[&>svg] handed %v down from an anchor that is not hovered", got)
	}
	hands := sheet.Hands(anchor, hover, nil)
	if len(hands) != 1 {
		t.Fatalf("hover:[&>svg] handed %v down from a hovered anchor, want one rule", hands)
	}
	target := &sheet.Rule(hands[0]).Target
	if target.Relation != style.RelationChild || !target.Accepts(style.ElementSVG, 0, style.NodeState{}) || target.Accepts(style.ElementSpan, 0, style.NodeState{}) {
		t.Errorf("hover:[&>svg] target %+v", *target)
	}
	if got := sheet.ComputeRelated(style.ComputedStyle{}, nil, style.NodeState{}, hands).Shrink; got != 0 {
		t.Errorf("the anchor's hover was checked again on the child: shrink %v", got)
	}
	link := sheet.Hands([]string{"[&>a:hover]:underline"}, style.NodeState{}, nil)
	if a := &sheet.Rule(link[0]).Target; a.Accepts(style.ElementA, 0, style.NodeState{}) || !a.Accepts(style.ElementA, 0, hover) {
		t.Errorf("[&>a:hover] must test the child's hover")
	}
	sr := sheet.Hands([]string{"[&>.sr-only]:hidden"}, style.NodeState{}, nil)
	if m := &sheet.Rule(sr[0]).Target; m.Accepts(style.ElementAny, sheet.Marks([]string{"p-2"}), style.NodeState{}) || !m.Accepts(style.ElementAny, sheet.Marks([]string{"p-2", "sr-only"}), style.NodeState{}) {
		t.Errorf("[&>.sr-only] must test the child's class")
	}
	padded := sheet.Hands([]string{"[&>*]:p-1"}, style.NodeState{}, nil)
	if got := sheet.ComputeRelated(style.ComputedStyle{}, []string{"p-2"}, style.NodeState{}, padded).Padding.Top.Value; got != 1 {
		t.Errorf("[&>*]:p-1 from the parent against the child's own p-2: padding %v, want 1 (sheet order)", got)
	}
}

func TestStructuralTooManyMarkers(t *testing.T) {
	var rules []style.Rule
	for i := range konst.MaxMarkers + 1 {
		rules = append(rules, style.Rule{Class: "x", Near: style.Match{Relation: style.RelationAncestor, Class: "group/" + strconv.Itoa(i)}, Decls: []style.Declaration{{Property: style.PropBold, Flag: true}}})
	}
	if _, err := style.NewSheet(konst.IRVersion, rules[:konst.MaxMarkers]); err != nil {
		t.Errorf("%d markers: %v", konst.MaxMarkers, err)
	}
	if _, err := style.NewSheet(konst.IRVersion, rules); err == nil {
		t.Errorf("%d markers: no error", konst.MaxMarkers+1)
	}
}
