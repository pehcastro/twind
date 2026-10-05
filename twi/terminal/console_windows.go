//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"slices"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
)

func size(fd uintptr) (width, height int, err error) {
	return consoleSize(win32{}, windows.Handle(fd))
}

func consoleSize(c console, h windows.Handle) (width, height int, err error) {
	var info windows.ConsoleScreenBufferInfo
	if err := c.bufferInfo(h, &info); err != nil {
		return 0, 0, err
	}
	return int(info.Window.Right-info.Window.Left) + 1, int(info.Window.Bottom-info.Window.Top) + 1, nil
}

func promised(fd uintptr) color.Profile {
	return consoleProfile(win32{}, windows.Handle(fd))
}

func consoleProfile(c console, h windows.Handle) color.Profile {
	var original uint32
	if c.getMode(h, &original) != nil || c.setMode(h, original|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) != nil || c.setMode(h, original) != nil {
		return color.None
	}
	return color.TrueColor
}

func EnableVirtualTerminal(f *os.File) (restore func() error, err error) {
	handle := windows.Handle(f.Fd())
	var original uint32
	if err := windows.GetConsoleMode(handle, &original); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, original|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(handle, original) }, nil
}

type inputRecord struct {
	kind    uint16
	_       uint16
	keyDown int32
	repeat  uint16
	vk      uint16
	scan    uint16
	char    uint16
	control uint32
}

type mouseRecord struct {
	kind    uint16
	_       uint16
	x, y    int16
	buttons uint32
	control uint32
	flags   uint32
}

type console interface {
	getMode(h windows.Handle, mode *uint32) error
	setMode(h windows.Handle, mode uint32) error
	bufferInfo(h windows.Handle, info *windows.ConsoleScreenBufferInfo) error
	readInput(in, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error)
	windowClass() string
	font(h windows.Handle) Font
	lacks(face, cluster string) bool
	glyph(face, cluster string, size image.Point, bold bool) []uint8
	drawable() (window, error)
	hostExe() string
	overlay(tr *trace, id Identity) (host, error)
}

type win32 struct {
	readConsoleInput, consoleWindow, currentFont                             *windows.LazyProc
	createFont, createDC, selectObject, glyphIndices, deleteDC, deleteObject *windows.LazyProc
	createDIB, bitBlt, alphaBlend                                            *windows.LazyProc
	getDC, releaseDC, clientRect, invalidateRect, threadDPI                  *windows.LazyProc
	clientToScreen, iconic, findWindow, relative, windowRect                 *windows.LazyProc
	setProp, getProp, removeProp                                             *windows.LazyProc
	registerClass, createWindow, destroyWindow, showWindow, setWindowPos     *windows.LazyProc
	updateLayered, getMessage, dispatchMessage, postThreadMessage, defProc   *windows.LazyProc
	textColor, backColor, textAlign, textOut, flush                          *windows.LazyProc
}

func loadWin32() win32 {
	kernel, gdi, user := windows.NewLazySystemDLL("kernel32.dll"), windows.NewLazySystemDLL("gdi32.dll"), windows.NewLazySystemDLL("user32.dll")
	return win32{
		readConsoleInput: kernel.NewProc("ReadConsoleInputW"),
		consoleWindow:    kernel.NewProc("GetConsoleWindow"),
		currentFont:      kernel.NewProc("GetCurrentConsoleFontEx"),
		createFont:       gdi.NewProc("CreateFontW"),
		createDC:         gdi.NewProc("CreateCompatibleDC"),
		selectObject:     gdi.NewProc("SelectObject"),
		glyphIndices:     gdi.NewProc("GetGlyphIndicesW"),
		deleteDC:         gdi.NewProc("DeleteDC"),
		deleteObject:     gdi.NewProc("DeleteObject"),
		createDIB:        gdi.NewProc("CreateDIBSection"),
		bitBlt:           gdi.NewProc("BitBlt"),
		alphaBlend:       gdi.NewProc("GdiAlphaBlend"),
		getDC:            user.NewProc("GetDC"),
		releaseDC:        user.NewProc("ReleaseDC"),
		clientRect:       user.NewProc("GetClientRect"),
		invalidateRect:   user.NewProc("InvalidateRect"),
		threadDPI:        user.NewProc("SetThreadDpiAwarenessContext"),

		clientToScreen:    user.NewProc("ClientToScreen"),
		iconic:            user.NewProc("IsIconic"),
		findWindow:        user.NewProc("FindWindowExW"),
		relative:          user.NewProc("GetWindow"),
		windowRect:        user.NewProc("GetWindowRect"),
		setProp:           user.NewProc("SetPropW"),
		getProp:           user.NewProc("GetPropW"),
		removeProp:        user.NewProc("RemovePropW"),
		registerClass:     user.NewProc("RegisterClassExW"),
		createWindow:      user.NewProc("CreateWindowExW"),
		destroyWindow:     user.NewProc("DestroyWindow"),
		showWindow:        user.NewProc("ShowWindow"),
		setWindowPos:      user.NewProc("SetWindowPos"),
		updateLayered:     user.NewProc("UpdateLayeredWindowIndirect"),
		getMessage:        user.NewProc("GetMessageW"),
		dispatchMessage:   user.NewProc("DispatchMessageW"),
		postThreadMessage: user.NewProc("PostThreadMessageW"),
		defProc:           user.NewProc("DefWindowProcW"),
		textColor:         gdi.NewProc("SetTextColor"),
		backColor:         gdi.NewProc("SetBkColor"),
		textAlign:         gdi.NewProc("SetTextAlign"),
		textOut:           gdi.NewProc("ExtTextOutW"),
		flush:             gdi.NewProc("GdiFlush"),
	}
}

type consoleFontInfo struct {
	size   uint32
	index  uint32
	cell   windows.Coord
	family uint32
	weight uint32
	face   [konst.FaceLength]uint16
}

func (k win32) font(h windows.Handle) Font {
	info := consoleFontInfo{size: uint32(unsafe.Sizeof(consoleFontInfo{}))}
	if ok, _, _ := k.currentFont.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return Font{}
	}
	return Font{Face: windows.UTF16ToString(info.face[:]), Size: image.Pt(int(info.cell.X), int(info.cell.Y))}
}

func (k win32) lacks(face, cluster string) bool {
	name, err := windows.UTF16PtrFromString(face)
	if err != nil {
		return false
	}
	font, _, _ := k.createFont.Call(0, 0, 0, 0, 0, 0, 0, 0, konst.DefaultCharset, 0, 0, 0, 0, uintptr(unsafe.Pointer(name)))
	if font == 0 {
		return false
	}
	units := utf16.Encode([]rune(cluster))
	glyphs := make([]uint16, len(units))
	dc, _, _ := k.createDC.Call(0)
	old, _, _ := k.selectObject.Call(dc, font)
	n, _, _ := k.glyphIndices.Call(dc, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)), uintptr(unsafe.Pointer(&glyphs[0])), konst.MarkMissingGlyphs)
	_, _, _ = k.selectObject.Call(dc, old)
	_, _, _ = k.deleteDC.Call(dc)
	_, _, _ = k.deleteObject.Call(font)
	return uint32(n) != konst.GDIError && slices.Contains(glyphs, konst.MissingGlyph)
}

func (k win32) windowClass() string {
	window, _, _ := k.consoleWindow.Call()
	if window == 0 || !windows.IsWindowVisible(windows.HWND(window)) {
		return ""
	}
	return classOf(window)
}

func classOf(window uintptr) string {
	class := make([]uint16, konst.WindowClassLength)
	n, _ := windows.GetClassName(windows.HWND(window), &class[0], int32(len(class)))
	return windows.UTF16ToString(class[:n])
}

func (win32) getMode(h windows.Handle, mode *uint32) error {
	return windows.GetConsoleMode(h, mode)
}

func (win32) setMode(h windows.Handle, mode uint32) error {
	return windows.SetConsoleMode(h, mode)
}

func (win32) bufferInfo(h windows.Handle, info *windows.ConsoleScreenBufferInfo) error {
	return windows.GetConsoleScreenBufferInfo(h, info)
}

func (k win32) readInput(in, cancel windows.Handle, records []inputRecord, timeout uint32) (int, error) {
	event, err := windows.WaitForMultipleObjects([]windows.Handle{cancel, in}, false, timeout)
	switch {
	case err != nil:
		return 0, err
	case event == uint32(windows.WAIT_TIMEOUT):
		return 0, errQuiet
	case event == windows.WAIT_OBJECT_0:
		return 0, io.EOF
	}
	var n uint32
	ok, _, err := k.readConsoleInput.Call(uintptr(in), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&n)))
	if ok == 0 {
		return 0, err
	}
	return int(n), nil
}

type consoleTTY struct {
	console         console
	in, out         windows.Handle
	inMode, outMode uint32
	cancelled       windows.Handle
	records         []inputRecord
	high            rune
	buttons         uint16
	reportMatched   int
}

func openTTY(in, out *os.File, opt Options) (tty, error) {
	return openConsole(loadWin32(), windows.Handle(in.Fd()), windows.Handle(out.Fd()), opt)
}

func openConsole(c console, in, out windows.Handle, opt Options) (tty, error) {
	t := &consoleTTY{console: c, in: in, out: out, records: make([]inputRecord, konst.ReadBuffer/konst.ConsoleRecordBytes)}
	if err := errors.Join(c.getMode(in, &t.inMode), c.getMode(out, &t.outMode)); err != nil {
		return nil, err
	}
	cancelled, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	t.cancelled = cancelled
	mode := uint32(windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_WINDOW_INPUT | windows.ENABLE_EXTENDED_FLAGS)
	if opt.NoMouse {
		mode |= t.inMode & windows.ENABLE_QUICK_EDIT_MODE
	} else {
		mode |= windows.ENABLE_MOUSE_INPUT
	}
	err = errors.Join(c.setMode(out, t.outMode|windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING), c.setMode(in, mode))
	if err != nil {
		return nil, errors.Join(err, t.restore())
	}
	return t, nil
}

func (t *consoleTTY) read(p []byte, wait time.Duration) (int, bool, error) {
	timeout := uint32(windows.INFINITE)
	if wait > 0 {
		timeout = uint32(wait.Milliseconds())
	}
	n, err := t.console.readInput(t.in, t.cancelled, t.records, timeout)
	if err != nil {
		return 0, false, err
	}
	p, resized := p[:0], false
	for _, r := range t.records[:n] {
		switch r.kind {
		case windows.KEY_EVENT:
			c := rune(r.char)
			switch {
			case r.keyDown == 0 || c == 0:
			case utf16.IsSurrogate(c) && t.high == 0:
				t.high = c
			default:
				if t.high != 0 {
					c, t.high = utf16.DecodeRune(t.high, c), 0
				}
				p = utf8.AppendRune(p, c)
				t.matchReport(c)
			}
		case windows.MOUSE_EVENT:
			if t.reportMatched < len(konst.MouseReport) {
				p = t.mouse(p, (*mouseRecord)(unsafe.Pointer(&r)))
			}
		case windows.WINDOW_BUFFER_SIZE_EVENT:
			resized = true
		case windows.FOCUS_EVENT, windows.MENU_EVENT:
		}
	}
	return len(p), resized, nil
}

func (t *consoleTTY) matchReport(c rune) {
	switch {
	case t.reportMatched == len(konst.MouseReport):
	case c == rune(konst.MouseReport[t.reportMatched]):
		t.reportMatched++
	case c == konst.ESC:
		t.reportMatched = 1
	default:
		t.reportMatched = 0
	}
}

func (t *consoleTTY) mouse(p []byte, m *mouseRecord) []byte {
	var info windows.ConsoleScreenBufferInfo
	if t.console.bufferInfo(t.out, &info) != nil {
		return p
	}
	held := uint16(m.buttons)
	pressed, released := held&^t.buttons, t.buttons&^held
	t.buttons = held
	wheel, code, final := int16(m.buttons>>16), 0, byte('M')
	switch {
	case m.flags&windows.MOUSE_WHEELED != 0 && wheel > 0:
		code = konst.WheelUpReport
	case m.flags&windows.MOUSE_WHEELED != 0:
		code = konst.WheelDownReport
	case m.flags&windows.MOUSE_HWHEELED != 0 && wheel > 0:
		code = konst.WheelRightReport
	case m.flags&windows.MOUSE_HWHEELED != 0:
		code = konst.WheelLeftReport
	case m.flags&windows.MOUSE_MOVED != 0:
		code = buttonReport(held) | konst.MotionReport
	case pressed != 0:
		code = buttonReport(pressed)
	case released != 0:
		code, final = buttonReport(released), 'm'
	default:
		return p
	}
	if m.control&windows.SHIFT_PRESSED != 0 {
		code |= int(input.ModShift) << konst.ModifierShift
	}
	if m.control&(windows.LEFT_ALT_PRESSED|windows.RIGHT_ALT_PRESSED) != 0 {
		code |= int(input.ModAlt) << konst.ModifierShift
	}
	if m.control&(windows.LEFT_CTRL_PRESSED|windows.RIGHT_CTRL_PRESSED) != 0 {
		code |= int(input.ModCtrl) << konst.ModifierShift
	}
	return fmt.Appendf(p, "%s%d;%d;%d%c", konst.MouseReport, code, m.x-info.Window.Left+1, m.y-info.Window.Top+1, final)
}

func buttonReport(buttons uint16) int {
	switch {
	case buttons&windows.FROM_LEFT_1ST_BUTTON_PRESSED != 0:
		return konst.LeftReport
	case buttons&windows.FROM_LEFT_2ND_BUTTON_PRESSED != 0:
		return konst.MiddleReport
	case buttons&windows.RIGHTMOST_BUTTON_PRESSED != 0:
		return konst.RightReport
	}
	return konst.NoButtonReport
}

func (t *consoleTTY) size() (width, height int, err error) {
	return consoleSize(t.console, t.out)
}

func (t *consoleTTY) conhost() bool {
	return t.console.windowClass() == konst.ConhostWindowClass
}

func (t *consoleTTY) font() Font {
	return t.console.font(t.out)
}

func (t *consoleTTY) lacks(face, cluster string) bool {
	return t.console.lacks(face, cluster)
}

func (t *consoleTTY) glyph(face, cluster string, size image.Point, bold bool) []uint8 {
	return t.console.glyph(face, cluster, size, bold)
}

func (t *consoleTTY) drawable() (window, error) {
	return t.console.drawable()
}

func (t *consoleTTY) hostExe() string {
	return t.console.hostExe()
}

func (t *consoleTTY) overlay(tr *trace, id Identity) (host, error) {
	return t.console.overlay(tr, id)
}

func (t *consoleTTY) cancel() {
	_ = windows.SetEvent(t.cancelled)
}

func (t *consoleTTY) restore() error {
	return errors.Join(t.console.setMode(t.in, t.inMode), t.console.setMode(t.out, t.outMode), windows.CloseHandle(t.cancelled))
}
