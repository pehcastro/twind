//go:build windows

package terminal

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestConsoleGlyphCoverageOfConsolas(t *testing.T) {
	k := loadWin32()
	for _, c := range []struct {
		glyph  string
		lacked bool
	}{{"A", false}, {"─", false}, {"☾", true}, {"⑂", true}, {"\U0001FB00", true}} {
		if got := k.lacks("Consolas", c.glyph); got != c.lacked {
			t.Errorf("Consolas lacks %U: %v, want %v", []rune(c.glyph)[0], got, c.lacked)
		}
	}
}

func TestConsoleFontCoverage(t *testing.T) {
	if os.Getenv("TWIND_FONT") == "" {
		t.Skip("TWIND_FONT=1 reads the console font; run inside the console under test")
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close() //nolint:errcheck
	k := loadWin32()
	font := k.font(windows.Handle(out.Fd()))
	if font.Face == "" {
		t.Fatal("the console reported no font")
	}
	t.Logf("console font %q, cell %dx%d", font.Face, font.Size.X, font.Size.Y)
	for _, glyph := range []string{"☾", "☼", "▤", "▣", "✕", "A"} {
		t.Logf("%U %s lacked %v", []rune(glyph)[0], glyph, k.lacks(font.Face, glyph))
	}
	if k.lacks(font.Face, "A") {
		t.Errorf("%q reports A as missing", font.Face)
	}
}
