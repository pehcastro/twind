package render_test

import (
	"image"
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
)

func limited(classes string, limits *render.Placement, children ...render.Node) render.Node {
	n := node(classes, children...)
	if limits != nil {
		n.Extra = &render.Extra{Placement: *limits}
	}
	return n
}

func tenRows() []render.Node {
	rows := make([]render.Node, 10)
	for i := range rows {
		rows[i] = text("row")
	}
	return rows
}

func TestMaxSizeAndMinSizeOverTheClassSizes(t *testing.T) {
	var tree render.Tree
	f := cssFrame(t, 40)
	for _, step := range []struct {
		name  string
		page  render.Node
		sizes []layout.Rect
	}{
		{
			"a capped column, a widened fit box, an uncapped twin",
			screen(
				limited(sheet.Column, &render.Placement{Max: image.Pt(8, 3)}, tenRows()...),
				limited(sheet.Fit, &render.Placement{Min: image.Pt(10, 2)}, text("ab")),
				limited(sheet.Column, nil, tenRows()...),
			),
			[]layout.Rect{{W: 8, H: 3}, {W: 10, H: 2}, {W: 20, H: 10}},
		},
		{
			"the same tree with the limits swapped",
			screen(
				limited(sheet.Column, nil, tenRows()...),
				limited(sheet.Fit, nil, text("ab")),
				limited(sheet.Column, &render.Placement{Max: image.Pt(0, 4)}, tenRows()...),
			),
			[]layout.Rect{{W: 20, H: 10}, {W: 2, H: 1}, {W: 20, H: 4}},
		},
	} {
		root, err := tree.Scene(step.page, f)
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range step.sizes {
			if got := root.Children[i].Bounds; got.W != want.W || got.H != want.H {
				t.Errorf("%s: child %d is %dx%d, want %dx%d", step.name, i, got.W, got.H, want.W, want.H)
			}
		}
	}
}
