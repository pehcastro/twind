//go:build windows

package terminal

import (
	"image"
	"testing"
)

func BenchmarkGDIGlyph(b *testing.B) {
	k := loadWin32()
	for b.Loop() {
		k.glyph("Courier New", "ש", image.Pt(10, 20), false)
	}
}

func TestGDIDrawsAHebrewLetterInsideItsCell(t *testing.T) {
	k, size := loadWin32(), image.Pt(10, 20)
	mask := k.glyph("Courier New", "ש", size, false)
	if len(mask) != size.X*size.Y {
		t.Fatalf("mask of %d pixels, want %d", len(mask), size.X*size.Y)
	}
	inked, box := 0, image.Rectangle{Min: size}
	for i, a := range mask {
		if a > 0 {
			inked++
			box = box.Union(image.Rect(i%size.X, i/size.X, i%size.X+1, i/size.X+1))
		}
	}
	if inked < size.X || box.Dx() < size.X/2 || box.Min.Y < 1 || box.Max.Y > size.Y {
		t.Errorf("shin drew %d pixels in %v, want a letter at least half the cell wide inside the cell", inked, box)
	}
	if blank := k.glyph("Courier New", " ", size, false); len(blank) != len(mask) || max(blank[0], blank[len(blank)-1]) != 0 {
		t.Errorf("a space drew ink")
	}
}
