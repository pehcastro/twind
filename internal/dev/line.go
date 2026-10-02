package dev

import (
	"fmt"
	"strings"

	"github.com/twind-dev/twind/internal/dev/konst"
	"github.com/twind-dev/twind/twi/text"
)

func ErrorLine(err error, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	shown := err.Error()
	for line := range strings.Lines(shown) {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "exit status") {
			shown = line
			break
		}
	}
	shown = text.Truncate(text.Sanitize(strings.Join(strings.Fields(shown), " "), text.ShowBidi), width)
	return fmt.Sprintf("%s\x1b[%d;1H%s%s%s\x1b[0m%s", konst.SaveCursor, height, konst.ErrorStyle, shown, konst.EraseRight, konst.RestoreCursor)
}
