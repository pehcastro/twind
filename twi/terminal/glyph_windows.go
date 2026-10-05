//go:build windows

package terminal

import (
	"image"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func (k win32) glyph(face, cluster string, size image.Point, bold bool) []uint8 {
	name, err := windows.UTF16PtrFromString(face)
	units := utf16.Encode([]rune(cluster))
	if err != nil || len(units) == 0 {
		return nil
	}
	weight := konst.WeightNormal
	if bold {
		weight = konst.WeightBold
	}
	font, _, _ := k.createFont.Call(uintptr(size.Y), 0, 0, 0, uintptr(weight), 0, 0, 0, konst.DefaultCharset, 0, 0, konst.AntialiasQuality, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return nil
	}
	defer func() { _, _, _ = k.deleteObject.Call(font) }()
	dc, _, _ := k.createDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer func() { _, _, _ = k.deleteDC.Call(dc) }()
	header := bitmapHeader{width: int32(size.X), height: -int32(size.Y), planes: 1, bits: konst.DIBBitCount}
	header.size = uint32(unsafe.Sizeof(header))
	var bits unsafe.Pointer
	bitmap, _, _ := k.createDIB.Call(dc, uintptr(unsafe.Pointer(&header)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		return nil
	}
	defer func() { _, _, _ = k.deleteObject.Call(bitmap) }()
	oldBitmap, _, _ := k.selectObject.Call(dc, bitmap)
	oldFont, _, _ := k.selectObject.Call(dc, font)
	_, _, _ = k.textColor.Call(dc, konst.WhiteText)
	_, _, _ = k.backColor.Call(dc, 0)
	_, _, _ = k.textAlign.Call(dc, konst.CentreAlign)
	box := windows.Rect{Right: int32(size.X), Bottom: int32(size.Y)}
	_, _, _ = k.textOut.Call(dc, uintptr(size.X/2), 0, konst.OpaqueText, uintptr(unsafe.Pointer(&box)), uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)), 0)
	_, _, _ = k.flush.Call()
	pix := unsafe.Slice((*byte)(bits), size.X*size.Y*graphicskonst.GDIBytes)
	mask := make([]uint8, size.X*size.Y)
	for i := range mask {
		mask[i] = max(pix[i*graphicskonst.GDIBytes], pix[i*graphicskonst.GDIBytes+1], pix[i*graphicskonst.GDIBytes+2])
	}
	_, _, _ = k.selectObject.Call(dc, oldFont)
	_, _, _ = k.selectObject.Call(dc, oldBitmap)
	return mask
}
