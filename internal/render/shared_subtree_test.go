package render_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/twi/motion"
	"github.com/pehcastro/twind/twi/style"
	twitext "github.com/pehcastro/twind/twi/text"
)

func cloned(n render.Node) render.Node {
	n.Children = slices.Clone(n.Children)
	for i := range n.Children {
		n.Children[i] = cloned(n.Children[i])
	}
	return n
}

func describe(n *scene.Node, path string, out *strings.Builder) {
	fmt.Fprintf(out, "%s %v %v %v %v %v %v %.3f %v %q\n", path, n.Bounds, n.Clip, n.Padding, n.Content, n.Foreground, n.Background, n.Opacity, n.Visibility, n.Lines(twitext.Widths{}))
	for i := range n.Children {
		describe(&n.Children[i], fmt.Sprint(path, ".", i), out)
	}
}

func TestSharedChildrenDrawLikeFreshOnes(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	words := strings.Fields
	enter := &motion.Presence{Duration: time.Second}
	opened := []render.Node{
		{Classes: words(sheet.Opened), Text: "opened"},
		{Classes: words(sheet.Stack), Children: []render.Node{{Classes: words(sheet.Padded), Text: "pad"}, {Text: "two words"}, {Classes: words(sheet.Opened), Text: "deep"}}},
		{Key: "in", Classes: words(sheet.Button), Enter: enter, Text: "entering"},
	}
	labelled := []render.Node{{Text: "label"}}
	inked := []render.Node{{Text: "ink"}, {Classes: words(sheet.Row), Children: []render.Node{{Text: "deep"}}}}
	pulsing := []render.Node{{Classes: words(sheet.Pulsing)}}
	type step struct {
		open, disabled, dark, restyle, reduced, pulse bool
		label                                         string
		width                                         int
		now                                           time.Duration
	}
	page := func(s step) render.Node {
		state := "closed"
		if s.open {
			state = "open"
		}
		var peer style.NodeState
		if s.disabled {
			peer.States = style.StateDisabled
		}
		ink := words(sheet.Page)
		if s.dark {
			ink = words(sheet.Button)
		}
		children := []render.Node{
			{Classes: words(sheet.Group), State: &style.NodeState{Attrs: []style.Attr{{Name: "data-state", Value: state}}}, Children: opened},
			{Classes: words(sheet.Button), Text: s.label},
			{Classes: words(sheet.Peer), State: &peer, Text: "peer"},
			{Classes: words(sheet.PeerLabel), Children: labelled},
			{Classes: ink, Children: inked},
		}
		if s.pulse {
			children = append(children, render.Node{Children: pulsing})
		}
		return render.Node{Classes: words(sheet.Column), Children: children}
	}
	steps := []step{
		{label: "a", width: 60, pulse: true},
		{label: "a", width: 60, now: 100 * time.Millisecond, pulse: true},
		{label: "b", width: 60, now: 200 * time.Millisecond, pulse: true},
		{label: "b", width: 60, now: 2 * time.Second, pulse: true},
		{label: "b", width: 60, now: 2050 * time.Millisecond},
		{label: "b", width: 60, now: 2100 * time.Millisecond},
		{open: true, label: "b", width: 60, now: 2200 * time.Millisecond},
		{label: "b", width: 60, now: 2300 * time.Millisecond},
		{label: "b", width: 100, now: 2400 * time.Millisecond},
		{label: "b", width: 100, now: 2500 * time.Millisecond},
		{disabled: true, label: "b", width: 100, now: 2600 * time.Millisecond},
		{disabled: true, dark: true, label: "b", width: 100, now: 2700 * time.Millisecond},
		{disabled: true, dark: true, label: "b", width: 100, now: 2800 * time.Millisecond, restyle: true},
		{disabled: true, dark: true, label: "b", width: 100, now: 2900 * time.Millisecond, reduced: true},
		{disabled: true, label: "b", width: 60, now: 3 * time.Second, reduced: true},
	}
	var shared, reference render.Tree
	for i, s := range steps {
		f := render.Frame{Sheet: styles, Width: s.width, Height: layout.Length{Unit: layout.Cells, Value: 20}, Now: s.now, ReducedMotion: s.reduced, Graphics: true}
		if s.restyle {
			shared.Restyle()
			reference.Restyle()
		}
		got, err := shared.Scene(page(s), f)
		if err != nil {
			t.Fatal(err)
		}
		want, err := reference.Scene(cloned(page(s)), f)
		if err != nil {
			t.Fatal(err)
		}
		var a, b strings.Builder
		describe(&got, "", &a)
		describe(&want, "", &b)
		if a.String() != b.String() {
			t.Errorf("step %d: shared children drew\n%s\nfresh children drew\n%s", i, a.String(), b.String())
		}
	}
	if shared.Visits() >= reference.Visits() {
		t.Errorf("shared children were built %d times, fresh ones %d: nothing was skipped", shared.Visits(), reference.Visits())
	}
}
