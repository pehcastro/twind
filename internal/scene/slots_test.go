package scene

import (
	"image"
	"testing"
	"unsafe"

	konst "github.com/pehcastro/twind/internal/konst/scene"
	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/twi/color"
)

func TestNewFrameHoldsNoSlotTables(t *testing.T) {
	if size := unsafe.Sizeof(Frame{}); size > 4096 {
		t.Errorf("a zero Frame is %d bytes, want the look and stamp tables allocated as they fill", size)
	}
}

func TestLooksKeepTheirEntriesAcrossGrowth(t *testing.T) {
	var m looks
	ops := func(i int) []raster.Op {
		return []raster.Op{{Kind: raster.Fill, Box: raster.Box{Rect: raster.Rect{X: 1, Y: 2, W: float64(i + 1), H: 3}}, Color: color.RGBA{R: uint8(i), G: uint8(i * 7), A: 255}}}
	}
	n := konst.LookSlots / 2
	looked := make([]uint64, n)
	for i := range n {
		looked[i] = m.look(ops(i), image.Rect(0, 0, i+1, 3))
	}
	if len(m.slots) <= konst.FirstSlots {
		t.Fatalf("%d looks left the table at %d slots", n, len(m.slots))
	}
	kept := 0
	for i := range m.slots {
		if m.slots[i].to != 0 {
			kept++
		}
	}
	if kept < n*3/4 {
		t.Errorf("%d looks kept %d slots; growth dropped what the table held", n, kept)
	}
	for i := range n {
		if m.look(ops(i), image.Rect(0, 0, i+1, 3)) != looked[i] {
			t.Fatalf("look %d changed after the table grew", i)
		}
	}
}
