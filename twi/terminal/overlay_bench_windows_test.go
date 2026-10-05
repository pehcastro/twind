//go:build windows

package terminal

import (
	"image"
	"math"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func benchOverlay(b *testing.B) *overlay {
	k := loadWin32()
	runtime.LockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, err := k.createWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, konst.PopupStyle, 0, 0, 320, 200, 0, 0, 0, 0)
	if owner == 0 {
		b.Fatalf("no hidden owner window: %v", err)
	}
	h, err := k.overlayOn(owner)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		h.release()
		_, _, _ = k.destroyWindow.Call(owner)
		runtime.UnlockOSThread()
	})
	grid, width, height := image.Pt(187, 40), 7.797, 16.9
	cols, rows := make([]int, grid.X+1), make([]int, grid.Y+1)
	for c := range cols {
		cols[c] = int(math.Floor(float64(c) * width))
	}
	for r := range rows {
		rows[r] = int(math.Round(float64(r) * height))
	}
	o := &overlay{win: h, grid: grid, cell: image.Pt(8, 17), origin: image.Pt(-4000, -4000)}
	o.size = image.Pt(cols[grid.X], rows[grid.Y])
	o.columns, o.lines = remap(cols, o.cell.X), remap(rows, o.cell.Y)
	o.fresh, o.visible = true, true
	if !o.paint(Pixels{Cell: o.cell, Grid: grid, Clear: true}) {
		b.Fatal("the first clear paint was refused")
	}
	return o
}

func benchTiles(o *overlay, area image.Rectangle) []Tile {
	var tiles []Tile
	for y := area.Min.Y; y < area.Max.Y; y += 6 {
		for x := area.Min.X; x < area.Max.X; x += 8 {
			cells := image.Rect(x, y, min(x+8, area.Max.X), min(y+6, area.Max.Y))
			pix := make([]byte, cells.Dx()*o.cell.X*cells.Dy()*o.cell.Y*graphicskonst.GDIBytes)
			for i := range pix {
				pix[i] = byte(i)
			}
			tiles = append(tiles, Tile{Cells: cells, Pix: pix})
		}
	}
	return tiles
}

func BenchmarkOverlayTaken(b *testing.B) {
	h := benchOverlay(b).win.(*layered)
	other, err := loadWin32().overlayOn(h.term)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(other.release)
	other.pixels(image.Pt(4, 4))
	other.draw(image.Pt(-4000, -4000), image.Pt(4, 4), image.Rect(0, 0, 4, 4))
	h.take()
	other.take()
	for b.Loop() {
		if !h.taken() {
			b.Fatal("not taken by the later claim")
		}
	}
}

func BenchmarkOverlayPaint(b *testing.B) {
	for _, c := range []struct {
		name  string
		area  image.Rectangle
		clear bool
	}{
		{"row", image.Rect(0, 4, 32, 6), false},
		{"content", image.Rect(31, 2, 187, 40), false},
		{"full", image.Rect(0, 0, 187, 40), true},
	} {
		b.Run(c.name, func(b *testing.B) {
			o := benchOverlay(b)
			p := Pixels{Cell: o.cell, Grid: o.grid, Clear: c.clear, Tiles: benchTiles(o, c.area)}
			b.ResetTimer()
			for b.Loop() {
				if !o.paint(p) {
					b.Fatal("paint refused")
				}
			}
		})
	}
}
