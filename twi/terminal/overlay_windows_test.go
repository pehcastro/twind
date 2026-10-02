//go:build windows

package terminal

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func hiddenOwner(t *testing.T, k win32) uintptr {
	t.Helper()
	runtime.LockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	owner, _, err := k.createWindow.Call(0, uintptr(unsafe.Pointer(class)), 0, konst.PopupStyle, 0, 0, 320, 200, 0, 0, 0, 0)
	if owner == 0 {
		t.Fatalf("no hidden owner window: %v", err)
	}
	t.Cleanup(func() {
		_, _, _ = k.destroyWindow.Call(owner)
		runtime.UnlockOSThread()
	})
	return owner
}

func TestOverlayWindowIsAHiddenClickThroughToolWindow(t *testing.T) {
	k := loadWin32()
	owner := hiddenOwner(t, k)
	h, err := k.overlayOn(owner)
	if err != nil {
		t.Fatal(err)
	}
	exStyle, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("GetWindowLongPtrW").Call(h.hwnd, ^uintptr(19))
	if exStyle&konst.OverlayExStyle != konst.OverlayExStyle {
		t.Errorf("extended style %#x, want layered, transparent, no-activate and tool window %#x", exStyle, konst.OverlayExStyle)
	}
	if got, _, _ := k.relative.Call(h.hwnd, konst.OwnerWindow); got != owner {
		t.Errorf("owner %#x, want the host %#x", got, owner)
	}
	pix := make([]byte, 4*4*4)
	for i := range pix {
		pix[i] = 255
	}
	if !h.draw(image.Pt(-4000, -4000), image.Pt(4, 4), pix, image.Rect(1, 1, 3, 3)) {
		t.Errorf("UpdateLayeredWindowIndirect refused the first draw")
	}
	if !h.draw(image.Pt(-4000, -4000), image.Pt(4, 4), pix, image.Rect(1, 1, 3, 3)) {
		t.Errorf("UpdateLayeredWindowIndirect refused a partial draw")
	}
	if windows.IsWindowVisible(windows.HWND(h.hwnd)) {
		t.Errorf("the overlay became visible without show")
	}
	at, err := h.place()
	if err != nil || at.client != image.Pt(320, 200) || at.shown || at.front {
		t.Errorf("host place %+v %v, want a hidden 320x200 client in the background", at, err)
	}
	if shot := h.capture(image.Rect(0, 0, 4, 4)); len(shot) != 4*4*4 {
		t.Errorf("reading a 4x4 corner of the screen gave %d bytes, want 64", len(shot))
	}
	h.release()
	if windows.IsWindow(windows.HWND(h.hwnd)) {
		t.Errorf("the overlay window outlived release")
	}
}

func TestOverlayHostLineNamesTheWindowOrWhy(t *testing.T) {
	line := OverlayHost()
	t.Log(line)
	if !strings.HasPrefix(line, "host hwnd ") && !strings.HasPrefix(line, "no host: ") {
		t.Errorf("host line %q names neither a window nor a reason", line)
	}
}

func TestOverlayWindowDiesWithItsProcess(t *testing.T) {
	if os.Getenv("TWIND_OVERLAY_CHILD") != "" {
		k := loadWin32()
		h, err := k.overlayOn(hiddenOwner(t, k))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("overlay %d\n", h.hwnd)
		os.Exit(3)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestOverlayWindowDiesWithItsProcess$")
	cmd.Env = append(os.Environ(), "TWIND_OVERLAY_CHILD=1")
	out, _ := cmd.Output()
	if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != 3 {
		t.Fatalf("child did not stop with its overlay open: %q", out)
	}
	_, after, found := strings.Cut(string(out), "overlay ")
	hwnd, err := strconv.ParseUint(strings.TrimSpace(after), 10, 64)
	if !found || err != nil || hwnd == 0 {
		t.Fatalf("child printed no overlay window: %q", out)
	}
	if windows.IsWindow(windows.HWND(uintptr(hwnd))) {
		t.Errorf("overlay window %#x outlived its process", hwnd)
	}
}
