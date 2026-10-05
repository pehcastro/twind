package dev

import (
	"errors"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/dev/konst"
	"github.com/pehcastro/twind/twi/text"
)

func TestErrorLineIsSanitisedAndFits(t *testing.T) {
	compiler := errors.New("exit status 1\n# github.com/pehcastro/twind/twi/ui\ntwi\\ui\\input.go:32:5:\tundefined: \x1b]0;pwned\x07fieldEdgeX \x1b[2J" + strings.Repeat("界", 80) + "\nsecond error")
	line := ErrorLine(compiler, 61, 24)
	if !strings.HasPrefix(line, konst.SaveCursor+"\x1b[24;1H"+konst.ErrorStyle) || !strings.HasSuffix(line, konst.EraseRight+"\x1b[0m"+konst.RestoreCursor) {
		t.Fatalf("line is not framed to row 24: %q", line)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(line, konst.SaveCursor+"\x1b[24;1H"+konst.ErrorStyle), konst.EraseRight+"\x1b[0m"+konst.RestoreCursor)
	if strings.ContainsAny(body, "\x1b\x07\t\n") {
		t.Errorf("control bytes reach the screen: %q", body)
	}
	if !strings.Contains(body, "input.go:32:5: undefined: fieldEdgeX") {
		t.Errorf("the first compiler error is not shown: %q", body)
	}
	if strings.Contains(body, "pwned") {
		t.Errorf("an OSC title payload is shown: %q", body)
	}
	if w := text.Width(body); w > 61 {
		t.Errorf("width %d over 61: %q", w, body)
	}
	if got := ErrorLine(errors.New("x"), 0, 0); got != "" {
		t.Errorf("no size gave %q, want nothing", got)
	}
}
