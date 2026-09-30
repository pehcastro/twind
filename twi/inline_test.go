package twi_test

import (
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/testdata/hello"
)

type writes []string

func (w *writes) Write(p []byte) (int, error) {
	*w = append(*w, string(p))
	return len(p), nil
}

func TestRenderInline(t *testing.T) {
	wt := terminal.Capabilities{Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: image.Pt(10, 20)}
	const rows = 18
	cases := []struct {
		name       string
		caps       terminal.Capabilities
		cursor     image.Point
		screenRows int
		pixels     bool
		lead, top  int
	}{
		{"room below the cursor", wt, image.Pt(0, 2), 24, true, rows, 2},
		{"cursor on the last row scrolls first", wt, image.Pt(0, 23), 24, true, rows, 5},
		{"cursor after a prompt starts on the next line", wt, image.Pt(7, 2), 24, true, rows + 1, 3},
		{"one row left for the cursor after", wt, image.Pt(0, 0), rows + 1, true, rows, 0},
		{"as tall as the screen", wt, image.Pt(0, 0), rows, false, 0, 0},
		{"silent terminal", terminal.Capabilities{}, image.Pt(0, 2), 24, false, 0, 0},
		{"sixel without a cell size", terminal.Capabilities{Graphics: terminal.GraphicsSixel}, image.Pt(0, 2), 24, false, 0, 0},
	}
	cup := regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)
	for _, tc := range cases {
		var out writes
		pixels, err := twi.RenderInline(&out, hello.App(), tc.caps, tc.cursor, tc.screenRows, sheet(t), twi.Width(80), twi.ColorProfile(color.TrueColor))
		if err != nil {
			t.Fatal(err)
		}
		if pixels != tc.pixels {
			t.Errorf("%s: pixels %v, want %v", tc.name, pixels, tc.pixels)
		}
		if !tc.pixels {
			if len(out) != 0 {
				t.Errorf("%s: wrote %d times, want nothing", tc.name, len(out))
			}
			continue
		}
		if len(out) != 1 {
			t.Fatalf("%s: %d writes, want one", tc.name, len(out))
		}
		head := "\x1b[?2026h" + strings.Repeat("\n", tc.lead) + fmt.Sprintf("\x1b[%d;%dr\x1b[?6h", tc.top+1, tc.top+rows)
		tail := fmt.Sprintf("\x1b[?6l\x1b[r\x1b[%d;1H\x1b[?2026l", tc.top+rows+1)
		frame, headOK := strings.CutPrefix(out[0], head)
		frame, tailOK := strings.CutSuffix(frame, tail)
		if !headOK || !tailOK {
			t.Errorf("%s: starts %q, ends %q; want %q and %q", tc.name, out[0][:min(len(out[0]), 40)], out[0][max(len(out[0])-40, 0):], head, tail)
			continue
		}
		if !strings.Contains(frame, "\x1bP") {
			t.Errorf("%s: no sixel in the frame", tc.name)
		}
		if strings.Contains(frame, "\x1b[?6") {
			t.Errorf("%s: the frame changes origin mode itself", tc.name)
		}
		for _, m := range cup.FindAllStringSubmatch(frame, -1) {
			if row, _ := strconv.Atoi(m[1]); row < 1 || row > rows {
				t.Errorf("%s: cursor moved to row %d of a %d row region", tc.name, row, rows)
			}
		}
	}
}
