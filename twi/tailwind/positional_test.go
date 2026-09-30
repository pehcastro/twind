package tailwind

import (
	"reflect"
	"slices"
	"testing"

	"github.com/twind-dev/twind/twi/style"
)

func TestFirst(t *testing.T) {
	first := style.PlaceFirst
	checkShapes(t, map[string]shape{
		"first:border-l":                  {when: style.Condition{Places: first}},
		"data-[spacing=0]:first:border-l": {when: style.Condition{Attrs: attr("data-spacing", "0"), Places: first}},
		"group-first:p-1":                 {near: style.Match{Relation: style.RelationAncestor, Class: "group", Places: first}},
		"has-[>:first-child]:p-1":         {near: style.Match{Relation: style.RelationChild, Places: first}},
	})
}

func TestLast(t *testing.T) {
	last := style.PlaceLast
	checkShapes(t, map[string]shape{
		"last:border-0":                 {when: style.Condition{Places: last}},
		"last:hover:bg-muted":           {when: style.Condition{States: style.StateHover, Places: last}},
		"*:last:border-b-0":             {target: style.Match{Relation: style.RelationChild, Places: last}},
		"[&>span:last-child]:underline": {target: style.Match{Relation: style.RelationChild, Element: style.ElementSpan, Places: last}},
		"peer-last:p-1":                 {near: style.Match{Relation: style.RelationPrevious, Class: "peer", Places: last}},
		"has-[:last-child]:p-1":         {near: style.Match{Relation: style.RelationDescendant, Places: last}},
	})
	checkShapes(t, map[string]shape{
		"[&:last-child[data-selected=true]_button]:border-r": {when: style.Condition{Attrs: attr("data-selected", "true"), Places: last}, target: style.Match{Relation: style.RelationDescendant, Element: style.ElementButton}},
	})
}

func TestOdd(t *testing.T) {
	checkShapes(t, map[string]shape{
		"odd:bg-muted":  {when: style.Condition{Places: style.PlaceOdd}},
		"even:bg-muted": {when: style.Condition{Places: style.PlaceEven}},
	})
}

func TestOnly(t *testing.T) {
	checkShapes(t, map[string]shape{"only:p-1": {when: style.Condition{Places: style.PlaceFirst | style.PlaceLast}}})
}

func TestNot(t *testing.T) {
	notFirst := style.Negation{Places: style.PlaceFirst}
	checkShapes(t, map[string]shape{
		"not-first:p-1":                     {when: style.Condition{Not: notFirst}},
		"not-last:border-b":                 {when: style.Condition{Not: style.Negation{Places: style.PlaceLast}}},
		"not-disabled:opacity-50":           {when: style.Condition{Not: style.Negation{States: style.StateDisabled}}},
		"not-hover:bg-muted":                {when: style.Condition{Not: style.Negation{States: style.StateHover}}},
		"not-data-[state=open]:hidden":      {when: style.Condition{Not: style.Negation{Attrs: attr("data-state", "open")}}},
		"[&>*:not(:first-child)]:p-1":       {target: style.Match{Relation: style.RelationChild, Not: notFirst}},
		"[&>*:not(:last-child)]:border-r-0": {target: style.Match{Relation: style.RelationChild, Not: style.Negation{Places: style.PlaceLast}}},
		"has-[>*:not(:first-child)]:p-1":    {near: style.Match{Relation: style.RelationChild, Not: notFirst}},
	})
}

func TestNotUnsupportedFormsWarn(t *testing.T) {
	rules := appRules(t)
	warned := appWarnings(t, "")
	for _, class := range []string{
		"not-only:p-1",
		"nth-3:p-1",
		"nth-last-2:-mt-1",
		"first-of-type:p-1",
		"file:p-1",
		"[&>tr]:last:p-1",
		"[&_svg:not([class*='size-'])]:size-4",
	} {
		if warned[class] != Unsupported {
			t.Errorf("%s: warned %v, want unsupported", class, warned[class])
		}
		if len(rules[class]) != 0 {
			t.Errorf("%s: compiled to %+v, want no rule", class, rules[class])
		}
	}
}

func TestPlaceholder(t *testing.T) {
	checkShapes(t, map[string]shape{
		"placeholder:text-muted-foreground":       {part: style.PartPlaceholder},
		"hover:placeholder:text-muted-foreground": {when: style.Condition{States: style.StateHover}, part: style.PartPlaceholder},
	})
}

func TestSelection(t *testing.T) {
	rules := appRules(t)
	warned := appWarnings(t, "selection:")
	own, descendants := shape{part: style.PartSelection}, shape{part: style.PartSelection, target: style.Match{Relation: style.RelationDescendant}}
	for _, class := range []string{"selection:bg-primary", "selection:text-primary-foreground"} {
		var got []shape
		for _, r := range rules[class] {
			r.When.Scheme = style.SchemeAny
			if s := (shape{r.When, r.Near, r.Target, r.Part}); len(r.Decls) > 0 && !slices.ContainsFunc(got, func(x shape) bool { return reflect.DeepEqual(x, s) }) {
				got = append(got, s)
			}
		}
		if !reflect.DeepEqual(got, []shape{descendants, own}) {
			t.Errorf("%s: shapes %+v, want the descendants' and the node's own selection", class, got)
		}
		if c, ok := warned[class]; ok {
			t.Errorf("%s warned %s", class, c)
		}
	}
}
