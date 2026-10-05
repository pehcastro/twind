package render_test

import (
	"testing"
	"time"

	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
)

func TestClosingTwinKeepsItsSiblingsFill(t *testing.T) {
	page := func(second bool) render.Node {
		closing := func(on bool) render.Node {
			w := render.Node{Classes: classes(sheet.Button)}
			if on {
				w.Children = []render.Node{{Classes: classes(sheet.Closing), Children: []render.Node{{Text: "gone"}}}}
			}
			return w
		}
		plain := render.Node{Classes: classes(sheet.Button)}
		return screen(plain, closing(true), plain, closing(second), plain)
	}
	kept, removed := newMotionRun(t), newMotionRun(t)
	for _, now := range []time.Duration{0, 50 * time.Millisecond, time.Second} {
		kept.at(now, page(true))
		removed.at(now, page(true))
	}
	for _, now := range []time.Duration{time.Second + 10*time.Millisecond, 2 * time.Second, 3 * time.Second} {
		want := kept.at(now, page(true)).Children[1].Children[0]
		got := removed.at(now, page(false)).Children[1].Children[0]
		if got.Opacity != want.Opacity || got.Bounds != want.Bounds {
			t.Errorf("%v: the twin of a closing node drawn at opacity %v %+v, want %v %+v", now, got.Opacity, got.Bounds, want.Opacity, want.Bounds)
		}
	}
}
