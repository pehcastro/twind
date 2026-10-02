package terminal

import (
	"strings"
	"testing"

	devkonst "github.com/twind-dev/twind/internal/dev/konst"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func TestDevChildKeepsTheScreen(t *testing.T) {
	for _, env := range []map[string]string{{devkonst.DevEnv: "1"}, {}} {
		o, err := offered(func(k string) string { return env[k] })
		if err != nil {
			t.Fatal(err)
		}
		dev := env[devkonst.DevEnv] != ""
		if o.dev != dev {
			t.Errorf("%v: offer dev %v, want %v", env, o.dev, dev)
		}
		term := newFake("\x1b[?2026;1$y", "\x1b[?1u", "\x1b[?62;22c")
		b, err := enter(term, term.tty, Options{}, o)
		if err != nil {
			t.Fatal(err)
		}
		entered := term.out()
		if got := strings.Contains(entered, devkonst.AltScreen); got == dev {
			t.Errorf("%v: alternate screen entered %v at startup: %q", env, got, entered)
		}
		if !strings.Contains(entered, konst.EnterScreen[len(devkonst.AltScreen):]) {
			t.Errorf("%v: startup lost the modes after the alternate screen: %q", env, entered)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		left := strings.TrimPrefix(term.out(), entered)
		if dev && left != "" {
			t.Errorf("dev child wrote %q on exit, want nothing", left)
		}
		if !dev && (!strings.Contains(left, konst.LeaveScreen) || !strings.Contains(left, konst.MouseOff) || !strings.Contains(left, konst.KittyPop)) {
			t.Errorf("plain exit wrote %q, want the full reset", left)
		}
	}
	if !strings.HasPrefix(konst.EnterScreen, devkonst.AltScreen) {
		t.Errorf("EnterScreen %q no longer starts with %q", konst.EnterScreen, devkonst.AltScreen)
	}
}
