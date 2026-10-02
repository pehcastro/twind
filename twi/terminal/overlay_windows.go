//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

type point struct{ x, y int32 }

type windowClass struct {
	size, style                        uint32
	proc                               uintptr
	classExtra, windowExtra            int32
	instance, icon, cursor, background uintptr
	menu, name                         *uint16
	smallIcon                          uintptr
}

type message struct {
	hwnd           uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	pt             point
	private        uint32
}

type layeredInfo struct {
	size   uint32
	dst    uintptr
	at     *point
	extent *point
	src    uintptr
	from   *point
	key    uint32
	blend  *uint32
	flags  uint32
	dirty  *windows.Rect
}

type layered struct {
	k      win32
	zed    uintptr
	hwnd   uintptr
	thread uint32
	ended  chan struct{}
	dib    gdiWindow
	shot   gdiWindow
}

func OverlayHost() string {
	k := loadWin32()
	if zed := k.ancestor(nil); zed != 0 {
		return k.describe(zed)
	}
	return fmt.Sprintf("no host: no ancestor within %d has a visible unowned window", konst.OverlayAncestors)
}

func (k win32) overlay(tr *trace) (host, error) {
	for _, p := range []*windows.LazyProc{k.createDC, k.deleteDC, k.selectObject, k.deleteObject, k.createDIB, k.clientRect, k.getDC, k.releaseDC, k.bitBlt, k.clientToScreen, k.iconic, k.findWindow, k.relative, k.registerClass, k.createWindow, k.destroyWindow, k.showWindow, k.setWindowPos, k.updateLayered, k.getMessage, k.dispatchMessage, k.postThreadMessage, k.defProc} {
		if err := p.Find(); err != nil {
			return nil, err
		}
	}
	zed := k.ancestor(tr)
	if zed == 0 {
		return nil, errors.New("no host window")
	}
	tr.log("host: %s", k.describe(zed))
	h, err := k.overlayOn(zed)
	if err != nil {
		return nil, err
	}
	tr.log("overlay window %#x created, owned by %#x", h.hwnd, zed)
	return h, nil
}

func (k win32) ancestor(tr *trace) uintptr {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		tr.log("host: process snapshot: %v", err)
		return 0
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()
	parents := map[uint32]uint32{}
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		parents[entry.ProcessID] = entry.ParentProcessID
	}
	shown := map[uint32][]candidate{}
	k.aware(func() {
		for w, _, _ := k.findWindow.Call(0, 0, 0, 0); w != 0; w, _, _ = k.findWindow.Call(0, w, 0, 0) {
			var pid uint32
			var client windows.Rect
			_, _ = windows.GetWindowThreadProcessId(windows.HWND(w), &pid)
			if owner, _, _ := k.relative.Call(w, konst.OwnerWindow); owner == 0 && windows.IsWindowVisible(windows.HWND(w)) {
				_, _, _ = k.clientRect.Call(w, uintptr(unsafe.Pointer(&client)))
				shown[pid] = append(shown[pid], candidate{w, int(client.Right) * int(client.Bottom)})
			}
		}
	})
	return hostOf(parents, shown, windows.GetCurrentProcessId(), uintptr(windows.GetForegroundWindow()), tr)
}

func (k win32) describe(zed uintptr) string {
	class := make([]uint16, konst.WindowClassLength)
	n, _ := windows.GetClassName(windows.HWND(zed), &class[0], int32(len(class)))
	var pid uint32
	_, _ = windows.GetWindowThreadProcessId(windows.HWND(zed), &pid)
	var origin point
	var client windows.Rect
	k.aware(func() {
		_, _, _ = k.clientToScreen.Call(zed, uintptr(unsafe.Pointer(&origin)))
		_, _, _ = k.clientRect.Call(zed, uintptr(unsafe.Pointer(&client)))
	})
	return fmt.Sprintf("host hwnd %#x pid %d class %s client %dx%d at %d,%d, foreground %t", zed, pid, windows.UTF16ToString(class[:n]), client.Right, client.Bottom, origin.x, origin.y, uintptr(windows.GetForegroundWindow()) == zed)
}

func (k win32) overlayOn(zed uintptr) (*layered, error) {
	dc, _, err := k.createDC.Call(0)
	shot, _, _ := k.createDC.Call(0)
	h := &layered{k: k, zed: zed, ended: make(chan struct{}), dib: gdiWindow{k: k, dc: dc}, shot: gdiWindow{k: k, dc: shot}}
	if dc == 0 || shot == 0 {
		h.dib.release()
		h.shot.release()
		return nil, fmt.Errorf("CreateCompatibleDC: %w", err)
	}
	made := make(chan error)
	go h.pump(made)
	if err := <-made; err != nil {
		h.dib.release()
		h.shot.release()
		return nil, err
	}
	return h, nil
}

func (h *layered) pump(made chan<- error) {
	runtime.LockOSThread()
	defer close(h.ended)
	if h.k.threadDPI.Find() == nil {
		_, _, _ = h.k.threadDPI.Call(konst.DPIPerMonitorAware)
	}
	var instance windows.Handle
	_ = windows.GetModuleHandleEx(0, nil, &instance)
	name, _ := windows.UTF16PtrFromString(konst.OverlayClass)
	class := windowClass{proc: h.k.defProc.Addr(), instance: uintptr(instance), name: name}
	class.size = uint32(unsafe.Sizeof(class))
	_, _, _ = h.k.registerClass.Call(uintptr(unsafe.Pointer(&class)))
	hwnd, _, err := h.k.createWindow.Call(konst.OverlayExStyle, uintptr(unsafe.Pointer(name)), 0, konst.PopupStyle, 0, 0, 0, 0, h.zed, 0, uintptr(instance), 0)
	h.hwnd, h.thread = hwnd, windows.GetCurrentThreadId()
	if hwnd == 0 {
		made <- fmt.Errorf("CreateWindowEx: %w", err)
		return
	}
	made <- nil
	var msg message
	for {
		if got, _, _ := h.k.getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0); int32(got) <= 0 {
			break
		}
		_, _, _ = h.k.dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
	_, _, _ = h.k.destroyWindow.Call(h.hwnd)
}

func (h *layered) place() (hostPlace, error) {
	at, err := hostPlace{}, error(errNoWindow)
	h.k.aware(func() {
		var origin point
		var client windows.Rect
		moved, _, _ := h.k.clientToScreen.Call(h.zed, uintptr(unsafe.Pointer(&origin)))
		sized, _, _ := h.k.clientRect.Call(h.zed, uintptr(unsafe.Pointer(&client)))
		if !windows.IsWindow(windows.HWND(h.zed)) || moved == 0 || sized == 0 {
			return
		}
		iconic, _, _ := h.k.iconic.Call(h.zed)
		at = hostPlace{
			origin: image.Pt(int(origin.x), int(origin.y)),
			client: image.Pt(int(client.Right), int(client.Bottom)),
			shown:  windows.IsWindowVisible(windows.HWND(h.zed)) && iconic == 0,
			front:  uintptr(windows.GetForegroundWindow()) == h.zed,
		}
		err = nil
	})
	return at, err
}

func (h *layered) capture(r image.Rectangle) []byte {
	var seen []byte
	h.k.aware(func() {
		if !h.shot.sized(r.Size()) {
			return
		}
		screen, _, _ := h.k.getDC.Call(0)
		if screen == 0 {
			return
		}
		if ok, _, _ := h.k.bitBlt.Call(h.shot.dc, 0, 0, uintptr(r.Dx()), uintptr(r.Dy()), screen, uintptr(r.Min.X), uintptr(r.Min.Y), konst.SourceCopy); ok != 0 {
			seen = h.shot.bits
		}
		_, _, _ = h.k.releaseDC.Call(0, screen)
	})
	return seen
}

func (h *layered) pixels(size image.Point) []byte {
	if !h.dib.sized(size) {
		return nil
	}
	return h.dib.bits
}

func (h *layered) draw(at, size image.Point, dirty image.Rectangle) bool {
	var drawn uintptr
	h.k.aware(func() {
		if size != h.dib.size {
			return
		}
		dst, extent, from, blend := point{int32(at.X), int32(at.Y)}, point{int32(size.X), int32(size.Y)}, point{}, uint32(konst.BlendPremultiplied)
		rect := windows.Rect{Left: int32(dirty.Min.X), Top: int32(dirty.Min.Y), Right: int32(dirty.Max.X), Bottom: int32(dirty.Max.Y)}
		info := layeredInfo{at: &dst, extent: &extent, src: h.dib.dc, from: &from, blend: &blend, flags: konst.LayeredAlpha, dirty: &rect}
		info.size = uint32(unsafe.Sizeof(info))
		drawn, _, _ = h.k.updateLayered.Call(h.hwnd, uintptr(unsafe.Pointer(&info)))
	})
	return drawn != 0
}

func (h *layered) move(at image.Point) {
	h.k.aware(func() {
		_, _, _ = h.k.setWindowPos.Call(h.hwnd, 0, uintptr(at.X), uintptr(at.Y), 0, 0, konst.MoveOnly)
	})
}

func (h *layered) show() {
	_, _, _ = h.k.showWindow.Call(h.hwnd, konst.ShowNoActivate)
}

func (h *layered) hide() {
	_, _, _ = h.k.showWindow.Call(h.hwnd, konst.HideWindow)
}

func (h *layered) release() {
	_, _, _ = h.k.postThreadMessage.Call(uintptr(h.thread), konst.QuitMessage, 0, 0)
	<-h.ended
	h.dib.release()
	h.shot.release()
}
