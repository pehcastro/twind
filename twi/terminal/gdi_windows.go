//go:build windows

package terminal

import (
	"image"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

type bitmapHeader struct {
	size          uint32
	width, height int32
	planes, bits  uint16
	compression   uint32
	image         uint32
	xPels, yPels  int32
	used          uint32
	important     uint32
}

type gdiWindow struct {
	k                win32
	hwnd, dc         uintptr
	bitmap, previous uintptr
	bits             []byte
	size             image.Point
}

func (k win32) drawable() (window, error) {
	for _, p := range []*windows.LazyProc{k.createDC, k.selectObject, k.deleteDC, k.deleteObject, k.createDIB, k.bitBlt, k.alphaBlend, k.getDC, k.releaseDC, k.clientRect, k.invalidateRect} {
		if p.Find() != nil {
			return nil, errNoWindow
		}
	}
	hwnd, _, _ := k.consoleWindow.Call()
	if hwnd == 0 {
		return nil, errNoWindow
	}
	w := &gdiWindow{k: k, hwnd: hwnd}
	k.aware(func() {
		if dc, _, _ := k.getDC.Call(hwnd); dc != 0 {
			w.dc, _, _ = k.createDC.Call(dc)
			_, _, _ = k.releaseDC.Call(hwnd, dc)
		}
	})
	if w.dc == 0 {
		return nil, errNoWindow
	}
	return w, nil
}

func (k win32) aware(do func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if k.threadDPI.Find() == nil {
		if previous, _, _ := k.threadDPI.Call(konst.DPIPerMonitorAware); previous != 0 {
			defer func() { _, _, _ = k.threadDPI.Call(previous) }()
		}
	}
	do()
}

func (w *gdiWindow) client() (image.Point, error) {
	var r windows.Rect
	var ok uintptr
	w.k.aware(func() { ok, _, _ = w.k.clientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r))) })
	if ok == 0 {
		return image.Point{}, errNoWindow
	}
	return image.Pt(int(r.Right), int(r.Bottom)), nil
}

func (w *gdiWindow) sized(size image.Point) bool {
	if size == w.size {
		return true
	}
	w.free()
	header := bitmapHeader{width: int32(size.X), height: -int32(size.Y), planes: 1, bits: konst.DIBBitCount}
	header.size = uint32(unsafe.Sizeof(header))
	var bits unsafe.Pointer
	bitmap, _, _ := w.k.createDIB.Call(w.dc, uintptr(unsafe.Pointer(&header)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		return false
	}
	w.previous, _, _ = w.k.selectObject.Call(w.dc, bitmap)
	w.bitmap, w.size, w.bits = bitmap, size, unsafe.Slice((*byte)(bits), size.X*size.Y*graphicskonst.GDIBytes)
	return true
}

func (w *gdiWindow) free() {
	if w.bitmap == 0 {
		return
	}
	_, _, _ = w.k.selectObject.Call(w.dc, w.previous)
	_, _, _ = w.k.deleteObject.Call(w.bitmap)
	w.bitmap, w.bits, w.size = 0, nil, image.Point{}
}

func (w *gdiWindow) blit(pix []byte, size image.Point, r image.Rectangle) {
	w.k.aware(func() {
		if !w.sized(size) {
			return
		}
		stride := size.X * graphicskonst.GDIBytes
		for y := r.Min.Y; y < r.Max.Y; y++ {
			copy(w.bits[y*stride+r.Min.X*graphicskonst.GDIBytes:y*stride+r.Max.X*graphicskonst.GDIBytes], pix[y*stride+r.Min.X*graphicskonst.GDIBytes:])
		}
		w.onWindow(func(dc uintptr) {
			_, _, _ = w.k.alphaBlend.Call(dc, uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Dx()), uintptr(r.Dy()), w.dc, uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Dx()), uintptr(r.Dy()), konst.BlendPremultiplied)
		})
	})
}

func (w *gdiWindow) read(size image.Point, r image.Rectangle) []byte {
	var seen []byte
	w.k.aware(func() {
		if !w.sized(size) {
			return
		}
		w.onWindow(func(dc uintptr) {
			if ok, _, _ := w.k.bitBlt.Call(w.dc, uintptr(r.Min.X), uintptr(r.Min.Y), uintptr(r.Dx()), uintptr(r.Dy()), dc, uintptr(r.Min.X), uintptr(r.Min.Y), konst.SourceCopy); ok != 0 {
				seen = w.bits
			}
		})
	})
	return seen
}

func (w *gdiWindow) onWindow(do func(dc uintptr)) {
	dc, _, _ := w.k.getDC.Call(w.hwnd)
	if dc == 0 {
		return
	}
	do(dc)
	_, _, _ = w.k.releaseDC.Call(w.hwnd, dc)
}

func (w *gdiWindow) invalidate(r image.Rectangle) {
	rect := windows.Rect{Left: int32(r.Min.X), Top: int32(r.Min.Y), Right: int32(r.Max.X), Bottom: int32(r.Max.Y)}
	w.k.aware(func() { _, _, _ = w.k.invalidateRect.Call(w.hwnd, uintptr(unsafe.Pointer(&rect)), 0) })
}

func (w *gdiWindow) release() {
	w.free()
	_, _, _ = w.k.deleteDC.Call(w.dc)
}
