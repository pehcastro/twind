package render_test

import (
	"fmt"
	"image"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
	"github.com/pehcastro/twind/internal/scene"
)

var palette = []string{
	"", sheet.Page, sheet.Truncate, sheet.Clip, sheet.Column, sheet.Centred, sheet.Row, sheet.Fit, sheet.Video,
	sheet.Wrap, sheet.Reverse, sheet.Shrink0, sheet.Stack, sheet.Padded, sheet.Footer, sheet.Button, sheet.Dialog,
	sheet.Upward, sheet.Leftward, sheet.Middle, sheet.Card, sheet.CardHeader, sheet.CardAction, sheet.Cells, sheet.Wide,
	sheet.Anchor, sheet.Popup, sheet.Hash, sheet.Context, sheet.Overlay, sheet.DialogBox, sheet.Shifted, sheet.Half,
	sheet.Focus + " hidden",
}

func randomNode(r *rand.Rand, depth int) render.Node {
	if depth == 0 || r.IntN(3) == 0 {
		return render.Node{Text: strings.Repeat("word ", r.IntN(6)) + "end"}
	}
	n := render.Node{Classes: strings.Fields(palette[r.IntN(len(palette))])}
	for range r.IntN(5) {
		n.Children = append(n.Children, randomNode(r, depth-1))
	}
	return n
}

func nodes(n *render.Node, into []*render.Node) []*render.Node {
	into = append(into, n)
	for i := range n.Children {
		into = nodes(&n.Children[i], into)
	}
	return into
}

func editNode(r *rand.Rand, root *render.Node) string {
	all := nodes(root, nil)
	n := all[r.IntN(len(all))]
	switch r.IntN(6) {
	case 0:
		if n.Children != nil || n.Classes != nil {
			return "none"
		}
		n.Text = strings.Repeat("longer ", r.IntN(8)) + "end"
		return "text"
	case 1:
		if n.Text != "" {
			return "none"
		}
		n.Classes = strings.Fields(palette[r.IntN(len(palette))])
		return "class"
	case 2:
		if n.Text != "" {
			return "none"
		}
		n.Children = slices.Insert(n.Children, r.IntN(len(n.Children)+1), randomNode(r, 2))
		return "add"
	case 3:
		if len(n.Children) == 0 {
			return "none"
		}
		at := r.IntN(len(n.Children))
		n.Children = slices.Delete(slices.Clone(n.Children), at, at+1)
		return "remove"
	case 4:
		n.TopLayer = r.IntN(2) * (1 + r.IntN(2))
		return "top layer"
	}
	n.Extra = nil
	if r.IntN(2) == 0 {
		n.Extra = &render.Extra{Placement: render.Placement{Positioned: true, At: image.Pt(r.IntN(30), r.IntN(12))}}
	}
	return "placed"
}

func geometry(n scene.Node, path string, into []string) []string {
	into = append(into, fmt.Sprint(path, n.Bounds, n.Padding, n.Content, n.Clip, n.ScrollContent, n.TopLayer, n.Position, n.ZIndex, n.Opacity, n.Visibility, n.Truncate, n.NoWrap, len(n.Children)))
	for i, c := range n.Children {
		into = geometry(c, fmt.Sprintf("%s/%d", path, i), into)
	}
	return into
}

func TestTopLayerReclipsRowsLaidOutInPlace(t *testing.T) {
	var tree render.Tree
	root := node("page", node("card", text("card"), menuNode(0)), node("later", text("later")))
	paintTree(t, &tree, root, nil, 0)
	menu := &root.Children[0].Children[1]
	menu.Children = append(menu.Children, text("five"))
	paintTree(t, &tree, root, nil, 0)
	menu.TopLayer = 1
	_, sc := paintTree(t, &tree, root, nil, 0)
	if rows := sc.Children[0].Children[1].Children; rows[0].Clip != (layout.Rect{W: 16, H: 8}) {
		t.Errorf("first menu row clip %+v after the menu joined the top layer, want the viewport", rows[0].Clip)
	}
}

func TestRetainedRandomEditsMatchFresh(t *testing.T) {
	styles, err := sheet.Styles()
	if err != nil {
		t.Fatal(err)
	}
	edits := 0
	for seed := range 30 {
		r := rand.New(rand.NewPCG(114, uint64(seed)))
		root := render.Node{Classes: strings.Fields(sheet.Page), Children: []render.Node{randomNode(r, 4), randomNode(r, 3)}}
		frame := render.Frame{Sheet: styles, Width: 40 + r.IntN(60), Height: layout.Length{Unit: layout.Cells, Value: 30}}
		var tree render.Tree
		if _, err := tree.Scene(root, frame); err != nil {
			t.Fatal(err)
		}
		for done := 0; done < 10; {
			kind := editNode(r, &root)
			if kind == "none" {
				continue
			}
			done, edits = done+1, edits+1
			if r.IntN(4) == 0 {
				frame.Width = 40 + r.IntN(60)
			}
			got, err := tree.Scene(root, frame)
			if err != nil {
				t.Fatalf("seed %d edit %d (%s): %v", seed, edits, kind, err)
			}
			want, err := render.Scene(root, frame)
			if err != nil {
				t.Fatal(err)
			}
			g, w := geometry(got, "", nil), geometry(want, "", nil)
			if !slices.Equal(g, w) {
				i := 0
				for i < min(len(g), len(w))-1 && g[i] == w[i] {
					i++
				}
				t.Fatalf("seed %d edit %d (%s): retained %s, fresh %s", seed, edits, kind, g[i], w[i])
			}
			if !reflect.DeepEqual(draw(got), draw(want)) {
				t.Fatalf("seed %d edit %d (%s): retained scene draws differently from a fresh one", seed, edits, kind)
			}
		}
	}
	if edits != 300 {
		t.Fatalf("%d edits, want 300", edits)
	}
}
