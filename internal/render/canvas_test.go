package render_test

import (
	"image"
	"testing"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/internal/render/testdata/sheet"
)

func TestCanvasRedrawsOnlyWhenItsKeyOrSizeChanges(t *testing.T) {
	r := newMotionRun(t)
	r.frame.Graphics, r.frame.Cell = true, image.Pt(10, 20)
	paints := 0
	page := func(key uint64, note string) render.Node {
		canvas := render.Node{Classes: classes(sheet.Dialog), Canvas: &render.Canvas{Key: key, Paint: func(dst *image.RGBA, cell image.Point) {
			paints++
			if cell != r.frame.Cell {
				t.Errorf("painted at cell %v, want %v", cell, r.frame.Cell)
			}
			dst.Pix[3] = 255
		}}, Children: []render.Node{{Text: "cells"}}}
		return screen(canvas, render.Node{Text: note})
	}
	var last *image.RGBA
	painted := 0
	for _, step := range []struct {
		why    string
		key    uint64
		note   string
		width  int
		paints int
	}{
		{"first frame", 1, "a", 80, 1},
		{"same key and size", 1, "a", 80, 1},
		{"a sibling's text changed", 1, "b", 80, 1},
		{"a new key", 2, "b", 80, 2},
		{"a narrower screen", 2, "b", 60, 3},
	} {
		r.frame.Width = step.width
		n := r.at(0, page(step.key, step.note)).Children[0]
		if paints != step.paints {
			t.Errorf("%s: %d paints, want %d", step.why, paints, step.paints)
		}
		if n.Pixels == nil {
			t.Fatalf("%s: no pixels on the canvas node", step.why)
		}
		r.wake(false)
		img := n.Pixels.Image
		if want := image.Pt(n.Content.W*10, n.Content.H*20); img.Rect.Size() != want || n.Pixels.Key != step.key {
			t.Errorf("%s: canvas %v key %d, want the content box %v key %d", step.why, img.Rect.Size(), n.Pixels.Key, want, step.key)
		}
		if len(n.Children) != 0 {
			t.Errorf("%s: the fallback cells are in the scene over the pixels", step.why)
		}
		if fresh := img != last; fresh != (paints != painted) {
			t.Errorf("%s: a new image %v after %d paints, want a new image exactly when painted", step.why, fresh, paints-painted)
		}
		last, painted = img, paints
	}
	r.frame.Graphics = false
	n := r.at(0, page(2, "b")).Children[0]
	if n.Pixels != nil || len(n.Children) != 1 || paints != 3 {
		t.Errorf("without graphics: pixels %v, %d children, %d paints; want no pixels, the fallback cells and no paint", n.Pixels != nil, len(n.Children), paints)
	}
}
