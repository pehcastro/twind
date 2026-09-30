package tailwind

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/style"
)

func appRules(t *testing.T) map[string][]style.Rule {
	t.Helper()
	src, err := os.ReadFile(appFixture + "/output.css")
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := Compile(string(src))
	if err != nil {
		t.Fatal(err)
	}
	byClass := map[string][]style.Rule{}
	for _, r := range rules {
		byClass[r.Class] = append(byClass[r.Class], r)
	}
	return byClass
}

type shape struct {
	when         style.Condition
	near, target style.Match
	part         style.Part
}

func checkShapes(t *testing.T, want map[string]shape) {
	t.Helper()
	rules := appRules(t)
	warned := appWarnings(t, "")
	for class, w := range want {
		if len(rules[class]) == 0 {
			t.Errorf("%s: no rule", class)
		}
		for _, r := range rules[class] {
			r.When.Scheme = style.SchemeAny
			if len(r.Decls) == 0 || !reflect.DeepEqual(shape{r.When, r.Near, r.Target, r.Part}, w) {
				t.Errorf("%s:\n got when %+v near %+v target %+v part %d decls %d\nwant when %+v near %+v target %+v part %d", class, r.When, r.Near, r.Target, r.Part, len(r.Decls), w.when, w.near, w.target, w.part)
			}
		}
		if c, ok := warned[class]; ok {
			t.Errorf("%s warned %s", class, c)
		}
	}
}

func attr(name, value string) []style.Attr { return []style.Attr{{Name: name, Value: value}} }

func TestGroup(t *testing.T) {
	ancestor := style.RelationAncestor
	checkShapes(t, map[string]shape{
		"group-hover:bg-accent":                 {near: style.Match{Relation: ancestor, Class: "group", States: style.StateHover}},
		"group-focus-visible:ring-2":            {near: style.Match{Relation: ancestor, Class: "group", States: style.StateFocusVisible}},
		"group-data-[state=open]:bg-accent":     {near: style.Match{Relation: ancestor, Class: "group", Attrs: attr("data-state", "open")}},
		"group-data-[state=open]/item:bg-muted": {near: style.Match{Relation: ancestor, Class: "group/item", Attrs: attr("data-state", "open")}},
		"group-data-[disabled=true]:opacity-50": {near: style.Match{Relation: ancestor, Class: "group", Attrs: attr("data-disabled", "true")}},
		"in-data-[slot=x]:p-1":                  {near: style.Match{Relation: ancestor, Attrs: attr("data-slot", "x")}},
	})
}

func TestPeer(t *testing.T) {
	previous := style.RelationPrevious
	checkShapes(t, map[string]shape{
		"peer-disabled:opacity-50":             {near: style.Match{Relation: previous, Class: "peer", States: style.StateDisabled}},
		"peer-hover:bg-muted":                  {near: style.Match{Relation: previous, Class: "peer", States: style.StateHover}},
		"peer-data-[state=checked]:bg-primary": {near: style.Match{Relation: previous, Class: "peer", Attrs: attr("data-state", "checked")}},
		"peer-checked/name:bg-accent":          {near: style.Match{Relation: previous, Class: "peer/name", States: style.StateChecked}},
	})
}

func TestHas(t *testing.T) {
	child, descendant := style.RelationChild, style.RelationDescendant
	checkShapes(t, map[string]shape{
		"has-data-[slot=card-action]:grid-cols-[1fr_auto]": {near: style.Match{Relation: descendant, Attrs: attr("data-slot", "card-action")}},
		"has-[>svg]:px-3":                     {near: style.Match{Relation: child, Element: style.ElementSVG}},
		"has-[svg]:gap-2":                     {near: style.Match{Relation: descendant, Element: style.ElementSVG}},
		"has-disabled:opacity-50":             {near: style.Match{Relation: descendant, States: style.StateDisabled}},
		"has-aria-invalid:border-destructive": {near: style.Match{Relation: descendant, Attrs: attr("aria-invalid", "true")}},
		"has-[[data-slot=x]]:p-2":             {near: style.Match{Relation: descendant, Attrs: attr("data-slot", "x")}},
		"has-[select:disabled]:opacity-50":    {near: style.Match{Relation: descendant, Element: style.ElementSelect, States: style.StateDisabled}},
		"[&:has([role=checkbox])]:pr-0":       {near: style.Match{Relation: descendant, Attrs: attr("role", "checkbox")}},
	})
}

func TestChild(t *testing.T) {
	child, descendant := style.RelationChild, style.RelationDescendant
	checkShapes(t, map[string]shape{
		"[&>svg]:size-4":                  {target: style.Match{Relation: child, Element: style.ElementSVG}},
		"[&_svg]:size-4":                  {target: style.Match{Relation: descendant, Element: style.ElementSVG}},
		"[&>*]:p-1":                       {target: style.Match{Relation: child}},
		"[&>a:hover]:underline":           {target: style.Match{Relation: child, Element: style.ElementA, States: style.StateHover}},
		"[&>[data-slot=x]]:flex":          {target: style.Match{Relation: child, Attrs: attr("data-slot", "x")}},
		"[&>.sr-only]:hidden":             {target: style.Match{Relation: child, Class: "sr-only"}},
		"*:data-[slot=select-value]:flex": {target: style.Match{Relation: child, Attrs: attr("data-slot", "select-value")}},
		"*:p-2":                           {target: style.Match{Relation: child}},
		"**:data-[slot=x]:p-1":            {target: style.Match{Relation: descendant, Attrs: attr("data-slot", "x")}},
		"hover:[&>svg]:shrink-0":          {when: style.Condition{States: style.StateHover}, target: style.Match{Relation: child, Element: style.ElementSVG}},
	})
}

func TestChildUnsupportedFormsWarn(t *testing.T) {
	rules := appRules(t)
	warned := appWarnings(t, "")
	for _, class := range []string{
		"group-[.foo]:bg-accent",
		"group-has-data-[slot=x]/item:gap-2",
		"has-[>a,>button]:p-2",
		"[&_svg:not([class*='size-'])]:size-4",
		"[&>video]:hidden",
		"group-hover:[&>svg]:shrink-0",
	} {
		if warned[class] != Unsupported {
			t.Errorf("%s: warned %v, want unsupported", class, warned[class])
		}
		if len(rules[class]) != 0 {
			t.Errorf("%s: compiled to %+v, want no rule", class, rules[class])
		}
	}
	for class := range warned {
		if strings.ContainsAny(class, "\\ ") || strings.HasPrefix(class, ":") || strings.HasPrefix(class, ".") {
			t.Errorf("warning names %q, not the class", class)
		}
	}
}

func TestPointerEvents(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for classes, want := range map[string]style.PointerEvents{
		"":                    style.PointerAuto,
		"pointer-events-none": style.PointerNone,
		"pointer-events-auto pointer-events-none": style.PointerNone,
		"pointer-events-none pointer-events-auto": style.PointerNone,
		"disabled:pointer-events-none":            style.PointerAuto,
	} {
		if got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes)).PointerEvents; got != want {
			t.Errorf("%q: pointer-events %v, want %v", classes, got, want)
		}
	}
	disabled := style.NodeState{States: style.StateDisabled}
	if got := sheet.ComputeState(style.ComputedStyle{}, []string{"disabled:pointer-events-none"}, disabled).PointerEvents; got != style.PointerNone {
		t.Errorf("disabled:pointer-events-none on a disabled node: %v", got)
	}
	if got := sheet.Compute(sheet.Compute(style.ComputedStyle{}, []string{"pointer-events-none"}), nil).PointerEvents; got != style.PointerNone {
		t.Errorf("pointer-events does not inherit: %v", got)
	}
	quiet(t, hasPrefix("pointer-events"))
}

func TestBreak(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	type breaks struct {
		wrap  style.OverflowWrap
		words style.WordBreak
	}
	for classes, want := range map[string]breaks{
		"":                                   {},
		"break-words":                        {wrap: style.OverflowWrapBreakWord},
		"wrap-break-word":                    {wrap: style.OverflowWrapBreakWord},
		"wrap-anywhere":                      {wrap: style.OverflowWrapAnywhere},
		"wrap-anywhere wrap-normal":          {},
		"break-all":                          {words: style.WordBreakAll},
		"break-keep":                         {words: style.WordBreakKeepAll},
		"break-normal":                       {},
		"break-normal break-all break-words": {wrap: style.OverflowWrapBreakWord, words: style.WordBreakAll},
	} {
		got := sheet.Compute(style.ComputedStyle{}, strings.Fields(classes))
		if (breaks{got.OverflowWrap, got.WordBreak}) != want {
			t.Errorf("%q: overflow-wrap %v word-break %v, want %+v", classes, got.OverflowWrap, got.WordBreak, want)
		}
	}
	parent := sheet.Compute(style.ComputedStyle{}, strings.Fields("break-all wrap-anywhere"))
	if child := sheet.Compute(parent, nil); child.WordBreak != style.WordBreakAll || child.OverflowWrap != style.OverflowWrapAnywhere {
		t.Errorf("word-break and overflow-wrap do not inherit: %+v", child)
	}
	quiet(t, hasPrefix("break-", "wrap-"))
}

func TestBreakpoint2xl(t *testing.T) {
	sheet, _ := compileFixture(t, appFixture)
	for columns, want := range map[int]style.Display{219: style.DisplayBlock, 220: style.DisplayFlex} {
		if got := sheet.WithColumns(columns).Compute(style.ComputedStyle{}, []string{"2xl:flex"}).Display; got != want {
			t.Errorf("2xl:flex at %d columns: display %v, want %v", columns, got, want)
		}
	}
	quiet(t, hasPrefix("2xl:"))
}
