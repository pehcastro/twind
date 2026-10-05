package render_test

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/style"
)

func column(row, word string) int {
	before, _, found := strings.Cut(row, word)
	if !found {
		return -1
	}
	return utf8.RuneCountInString(before)
}

func TestStructuralCardHeaderHasAction(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	action := true
	app := func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			header := []twi.NodeOption{
				twi.Class(sheet.ShadcnCardHeader),
				twi.Element(twi.Class(sheet.CardTitle), twi.Text("Card Title")),
				twi.Element(twi.Class(sheet.CardDescription), twi.Text("Card Description")),
			}
			if action {
				header = append(header, twi.Element(twi.Class(sheet.CardAction), twi.Data("slot", "card-action"), twi.Element(twi.Class(sheet.CardButton), twi.Text("Button"))))
			}
			return twi.Element(twi.Class(sheet.Page), twi.OnKey(func(input.KeyEvent) { action = !action; rt.Invalidate() }),
				twi.Element(twi.Class(sheet.Card), twi.Element(header...), twi.Element(twi.Class(sheet.CardContent), twi.Text("Content"))),
			)
		}
	}
	d := drive.New(app, drive.Size(60, 12), drive.Styles(styles))
	type place struct {
		word string
		x, y int
	}
	for _, step := range []struct {
		name  string
		press bool
		want  []place
	}{
		{"with an action", false, []place{{"Card Title", 7, 2}, {"Button", 43, 3}, {"Card Description", 7, 5}, {"Content", 7, 7}}},
		{"without an action", true, []place{{"Card Title", 7, 2}, {"Card Description", 7, 5}, {"Content", 7, 7}}},
		{"with the action back", true, []place{{"Card Title", 7, 2}, {"Button", 43, 3}, {"Card Description", 7, 5}, {"Content", 7, 7}}},
	} {
		if step.press {
			d.Press("space")
		}
		frame := d.Frame().Text()
		t.Logf("%s, 60x12:\n%s", step.name, frame)
		rows := strings.Split(frame, "\n")
		for _, p := range step.want {
			if got := column(rows[p.y], p.word); got != p.x {
				t.Errorf("%s: %q on row %d at column %d, want %d", step.name, p.word, p.y, got, p.x)
			}
		}
		if step.name == "without an action" && strings.Contains(frame, "Button") {
			t.Errorf("%s: the button is still drawn", step.name)
		}
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestStructuralCardHeaderRestylesOnlyTheHeader(t *testing.T) {
	f := cssFrame(t, 60)
	card := func(action bool) render.Node {
		header := node(sheet.ShadcnCardHeader, node(sheet.CardTitle, text("Title")))
		if action {
			header.Children = append(header.Children, render.Node{Classes: strings.Fields(sheet.CardAction), State: &style.NodeState{Attrs: []style.Attr{{Name: "data-slot", Value: "card-action"}}}})
		}
		return node(sheet.Page, node(sheet.Card, header, node(sheet.CardContent, text("Content"))))
	}
	var tree render.Tree
	for _, step := range []struct {
		action   bool
		title    int
		cascades int
	}{
		{false, 46, 7},
		{true, 44, 3},
		{false, 46, 2},
	} {
		before := tree.Cascades()
		root, err := tree.Scene(card(step.action), f)
		if err != nil {
			t.Fatal(err)
		}
		if got := root.Children[0].Children[0].Children[0].Bounds.W; got != step.title {
			t.Errorf("action %v: title %d wide, want %d (1fr beside an empty auto column and gap-2, or the one column)", step.action, got, step.title)
		}
		if n := tree.Cascades() - before; n != step.cascades {
			t.Errorf("action %v: %d cascades, want %d", step.action, n, step.cascades)
		}
	}
}

func TestStructuralGroupState(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	open := false
	state := func() string {
		if open {
			return "open"
		}
		return "closed"
	}
	app := func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class(sheet.Page), twi.OnKey(func(input.KeyEvent) { open = !open; rt.Invalidate() }),
				twi.Element(twi.Class(sheet.Group), twi.Data("state", state()),
					twi.Element(twi.Text("Trigger")),
					twi.Element(twi.Class(sheet.Opened), twi.Text("Closed hint")),
				),
				twi.Element(twi.Class(sheet.Opened), twi.Text("Outside")),
			)
		}
	}
	d := drive.New(app, drive.Size(20, 4), drive.Styles(styles))
	for _, step := range []struct {
		name  string
		press bool
		want  []string
	}{
		{"closed", false, []string{"Trigger", "Closed hint", "Outside", ""}},
		{"open", true, []string{"Trigger", "Outside", "", ""}},
		{"closed again", true, []string{"Trigger", "Closed hint", "Outside", ""}},
	} {
		if step.press {
			d.Press("space")
		}
		frame := d.Frame().Text()
		t.Logf("%s, 20x4:\n%s", step.name, frame)
		var rows []string
		for _, r := range strings.Split(strings.TrimSuffix(frame, "\n"), "\n") {
			rows = append(rows, strings.TrimRight(r, " "))
		}
		if !slices.Equal(rows, step.want) {
			t.Errorf("%s: rows %q, want %q", step.name, rows, step.want)
		}
	}
	if err := d.Close(); err != nil {
		t.Error(err)
	}
}

func TestStructuralGroupRestylesOnlyItsMatches(t *testing.T) {
	f := cssFrame(t, 30)
	attrs := func(state string) *style.NodeState {
		return &style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: state}}}
	}
	page := func(outer, inner string) render.Node {
		deep := node(sheet.Opened+" "+sheet.OpenedItem, text("deep"))
		innerGroup := render.Node{Classes: []string{sheet.Group}, State: attrs(inner), Children: []render.Node{deep, node("", text("plain"))}}
		outerGroup := render.Node{Classes: []string{sheet.GroupItem}, State: attrs(outer), Children: []render.Node{innerGroup}}
		return node(sheet.Page, outerGroup, node(sheet.Opened, text("outside")))
	}
	var tree render.Tree
	for _, step := range []struct {
		name         string
		outer, inner string
		shown        []string
		cascades     int
	}{
		{"both closed", "closed", "closed", []string{"deep", "plain", "outside"}, 9},
		{"outer named group opens", "open", "closed", []string{"plain", "outside"}, 3},
		{"inner group opens too", "open", "open", []string{"plain", "outside"}, 3},
		{"outer closes, inner still open", "closed", "open", []string{"plain", "outside"}, 2},
		{"both closed again", "closed", "closed", []string{"deep", "plain", "outside"}, 3},
	} {
		before := tree.Cascades()
		root, err := tree.Scene(page(step.outer, step.inner), f)
		if err != nil {
			t.Fatal(err)
		}
		var shown []string
		for _, r := range frameRows(root) {
			if r != "" {
				shown = append(shown, r)
			}
		}
		if !slices.Equal(shown, step.shown) {
			t.Errorf("%s: rows %q, want %q", step.name, shown, step.shown)
		}
		if n := tree.Cascades() - before; n != step.cascades {
			t.Errorf("%s: %d cascades, want %d", step.name, n, step.cascades)
		}
	}
}

func TestStructuralPeer(t *testing.T) {
	f := cssFrame(t, 20)
	row := func(disabled bool) render.Node {
		var states style.State
		if disabled {
			states = style.StateDisabled
		}
		return node(sheet.Page,
			node(sheet.PeerLabel, text("before")),
			render.Node{Classes: []string{sheet.Peer}, State: &style.NodeState{States: states}, Text: "peer"},
			node(sheet.PeerLabel, text("after")),
		)
	}
	var tree render.Tree
	for _, step := range []struct {
		disabled bool
		want     []string
	}{
		{false, []string{"before", "peer", "after"}},
		{true, []string{"before", "peer"}},
		{false, []string{"before", "peer", "after"}},
	} {
		root, err := tree.Scene(row(step.disabled), f)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.DeleteFunc(frameRows(root), func(r string) bool { return r == "" }); !slices.Equal(got, step.want) {
			t.Errorf("disabled %v: rows %q, want %q", step.disabled, got, step.want)
		}
	}
}

func TestStructuralChildSvg(t *testing.T) {
	svg := func(s string) render.Node { return render.Node{Element: style.ElementSVG, Text: s} }
	span := func(children ...render.Node) render.Node {
		return render.Node{Element: style.ElementSpan, Children: children}
	}
	f := cssFrame(t, 30)
	for _, c := range []struct {
		name string
		node render.Node
		want []layout.Rect
	}{
		{"child svg sized, grandchild not", node(sheet.Icons, svg("★"), span(svg("☆")), text("t")), []layout.Rect{{X: 0, Y: 0, W: 4, H: 4}, {X: 5, Y: 0, W: 1, H: 4}, {X: 7, Y: 0, W: 1, H: 4}}},
		{"descendant svg sized", node(sheet.DeepIcons, svg("★"), span(svg("☆"))), []layout.Rect{{X: 0, Y: 0, W: 4, H: 4}, {X: 5, Y: 0, W: 4, H: 4}}},
		{"span is not svg", node(sheet.Icons, span(text("x"))), []layout.Rect{{X: 0, Y: 0, W: 1, H: 1}}},
	} {
		root, err := render.Scene(c.node, f)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s:\n%s", c.name, strings.Join(frameRows(root), "\n"))
		var got []layout.Rect
		for _, child := range root.Children {
			got = append(got, child.Bounds)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: children at %+v, want %+v", c.name, got, c.want)
		}
	}
	root, err := render.Scene(node(sheet.Spaced, node("", text("a")), text("b")), f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := []layout.Rect{root.Children[0].Bounds, root.Children[1].Bounds}, []layout.Rect{{X: 0, Y: 0, W: 3, H: 1}, {X: 3, Y: 0, W: 1, H: 1}}; !slices.Equal(got, want) {
		t.Errorf("*:px-1 pads elements, never text: %+v, want %+v", got, want)
	}
}

func TestStructuralHandsFollowTheAnchor(t *testing.T) {
	svg := render.Node{Element: style.ElementSVG, Text: "★"}
	var tree render.Tree
	for _, step := range []struct {
		name   string
		width  int
		states style.State
		shown  bool
	}{
		{"idle", 90, 0, true},
		{"hovered", 90, style.StateHover, false},
		{"released", 90, 0, true},
		{"wide", 110, 0, false},
		{"narrow again", 90, 0, true},
	} {
		page := node("",
			render.Node{Classes: strings.Fields(sheet.HoverIcons), State: &style.NodeState{States: step.states}, Children: []render.Node{svg}},
			node(sheet.WideIcons, svg),
		)
		root, err := tree.Scene(page, cssFrame(t, step.width))
		if err != nil {
			t.Fatal(err)
		}
		hover, wide := root.Children[0].Children[0].Bounds.W > 0, root.Children[1].Children[0].Bounds.W > 0
		if hover != (step.states == 0) || wide != (step.width < 100) {
			t.Errorf("%s: hover icon shown %v, md icon shown %v", step.name, hover, wide)
		}
	}
}

func TestStructuralBreakMinContent(t *testing.T) {
	f := cssFrame(t, 20)
	f.Height = layout.Length{Unit: layout.Cells, Value: 3}
	for _, c := range []struct {
		classes string
		want    []string
	}{
		{"", []string{"              Open", "            Keyboard"}},
		{sheet.BreakWords, []string{"              Open", "            Keyboard"}},
		{sheet.BreakAll, []string{"              Open", "              Keyboa", "              rd"}},
		{sheet.Anywhere, []string{"              Open", "              Keyboa", "              rd"}},
	} {
		buf, err := render.Render(node(sheet.End, node(sheet.Anchor, text("Open"), node(sheet.Popup+" "+c.classes, text("Keyboard")))), f)
		if err != nil {
			t.Fatal(err)
		}
		if got := slices.DeleteFunc(rowsOf(buf), func(r string) bool { return r == "" }); !slices.Equal(got, c.want) {
			t.Errorf("%q: rows %q, want %q", c.classes, got, c.want)
		}
	}
}
