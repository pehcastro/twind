package layout

import "testing"

func BenchmarkLayout1000(b *testing.B) {
	root := cardsTree()
	b.ReportAllocs()
	for b.Loop() {
		full(root, 80, Length{})
	}
}

func BenchmarkGrid1000(b *testing.B) {
	root := gridTree()
	b.ReportAllocs()
	for b.Loop() {
		full(root, 80, Length{})
	}
}

func BenchmarkRelayoutOneText(b *testing.B) {
	badge := func(root *Box) *Box { return root.Children[25].Children[0].Children[2].Children[0] }
	paragraph := func(root *Box) *Box { return root.Children[25].Children[1] }
	for _, bc := range []struct {
		name    string
		lay     func(*Box, int, Length)
		changed func(root *Box) *Box
		other   *Box
	}{
		{"width", Layout, badge, text("badgebadgebadge!")},
		{"height", Layout, paragraph, wrap(380)},
		{"width/full", full, badge, text("badgebadgebadge!")},
		{"height/full", full, paragraph, wrap(380)},
	} {
		b.Run(bc.name, func(b *testing.B) {
			root := cardsTree()
			changed := bc.changed(root)
			measures := [2]Measure{changed.Measure, bc.other.Measure}
			Layout(root, 80, Length{})
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				i++
				changed.Measure = measures[i%2]
				changed.Invalidate()
				bc.lay(root, 80, Length{})
			}
		})
	}
}
