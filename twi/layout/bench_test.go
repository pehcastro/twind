package layout

import "testing"

func BenchmarkLayout1000(b *testing.B) {
	root := cardsTree()
	b.ReportAllocs()
	for b.Loop() {
		Layout(root, 80, Length{})
	}
}

func BenchmarkGrid1000(b *testing.B) {
	root := gridTree()
	b.ReportAllocs()
	for b.Loop() {
		Layout(root, 80, Length{})
	}
}
