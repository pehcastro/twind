package edit

import (
	"strings"
	"testing"
)

var long = strings.Repeat("the gate reads the policy before the wire. ", 10)

const longToken = "[Text 430 characters]"

func sent(b *Buffer) {
	b.Remember(b.Value())
	b.Set("")
}

func TestPasteLineBreaksAndTabsInOneStep(t *testing.T) {
	b := &Buffer{Mode: MultiLine}
	typeText(t, b, "x")
	b.Paste("a\r\nb\rc\td")
	want(t, b, "xa\nb\nc    d", 11, 11)
	press(t, b, ctrl('z'))
	want(t, b, "x", 1, 1)
	var single Buffer
	single.Paste("a\r\nb")
	want(t, &single, "ab", 2, 2)
}

func TestPasteLongBecomesAChip(t *testing.T) {
	var b Buffer
	typeText(t, &b, "look ")
	b.Paste(long)
	typeText(t, &b, " ok")
	if got := b.Value(); got != "look "+longToken+" ok" {
		t.Fatalf("the field holds %q, want the chip at the pasted place", got)
	}
	if got := b.Expand(b.Value()); got != "look "+long+" ok" {
		t.Fatalf("expanded %q", got)
	}
	short := long[:159]
	b = Buffer{}
	b.Paste(short)
	if b.Value() != short {
		t.Fatalf("159 characters became %q, want the text", b.Value())
	}
	b = Buffer{}
	b.Paste(long[:160])
	if b.Value() != "[Text 160 characters]" {
		t.Fatalf("160 characters became %q, want a chip", b.Value())
	}
}

func TestPasteBrokenChipStaysText(t *testing.T) {
	var b Buffer
	b.Paste(long)
	press(t, &b, backspace)
	if got := b.Expand(b.Value()); got != longToken[:len(longToken)-1] {
		t.Fatalf("a chip missing its bracket expanded to %q", got)
	}
}

func TestHistoryKeepsTheChip(t *testing.T) {
	var b Buffer
	b.Paste(long)
	sent(&b)
	press(t, &b, up)
	if b.Value() != longToken || b.Expand(b.Value()) != long {
		t.Fatalf("up recalled %q, expanding to %q", b.Value(), b.Expand(b.Value()))
	}
}

func TestHistoryChipNeverLeaks(t *testing.T) {
	var b Buffer
	b.Paste(long)
	sent(&b)
	typeText(t, &b, longToken)
	sent(&b)
	press(t, &b, up, up, down)
	if got := b.Expand(b.Value()); got != longToken {
		t.Fatalf("the typed entry expanded to %q, want it as typed", got[:min(len(got), 40)])
	}
}

func TestHistoryDraftChipComesBack(t *testing.T) {
	var b Buffer
	typeText(t, &b, "first")
	sent(&b)
	b.Paste(long)
	press(t, &b, up, down)
	if got := b.Expand(b.Value()); got != long {
		t.Fatalf("the draft expanded to %q, want the pasted text", got)
	}
}
