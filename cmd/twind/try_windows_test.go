//go:build windows

package main

import (
	"os"
	"testing"
	"time"
)

func TestTryInConsole(t *testing.T) {
	exe := devExe(t, t.TempDir())
	for demo, want := range map[string]string{"landing": "deploying", "portfolio": "November"} {
		c := openConsole(t, t.TempDir(), os.Environ(), exe, "try", demo)
		if _, ok := c.awaitText(t, want, 15*time.Second); !ok {
			t.Fatalf("twind try %s: no %q on the screen:\n%s", demo, want, c.text())
		}
		time.Sleep(500 * time.Millisecond)
		t.Logf("twind try %s, first frame at %dx%d:\n%s", demo, screenCols, screenRows, c.text())
		c.type_("q")
		if !c.exited(5 * time.Second) {
			t.Errorf("twind try %s still runs after q", demo)
		}
	}
}
