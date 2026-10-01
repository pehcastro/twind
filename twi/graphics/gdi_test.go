package graphics

import (
	"image"
	"slices"
	"testing"
)

func TestGDITextCellsAreTransparent(t *testing.T) {
	cell := image.Pt(2, 2)
	red := Run{End: 6, Pixel: 0xff0000ff}
	got := GDIPixels(nil, [][]Run{{red}, {red}}, cell, 0b010)
	want := []byte{
		0, 0, 255, 255, 0, 0, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 0, 0, 255, 255,
		0, 0, 255, 255, 0, 0, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 0, 0, 255, 255,
	}
	if !slices.Equal(got, want) {
		t.Errorf("text in the middle cell:\n got %v\nwant %v", got, want)
	}
}

func TestGDIAlphaIsAllOrNothing(t *testing.T) {
	cell := image.Pt(4, 1)
	line := []Run{{End: 1, Pixel: 200<<24 | 50<<16 | 100}, {End: 2, Pixel: 127<<24 | 127}, {End: 3, Pixel: 0}, {End: 4, Pixel: 128<<24 | 128<<8}}
	got := GDIPixels(nil, [][]Run{line}, cell, 0)
	want := []byte{64, 0, 128, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 0, 255}
	if !slices.Equal(got, want) {
		t.Errorf("partly covered pixels:\n got %v\nwant %v", got, want)
	}
	again := GDIPixels(got[:0:0], [][]Run{line}, cell, 0)
	if !slices.Equal(again, want) {
		t.Errorf("encoding twice differs: %v", again)
	}
}
