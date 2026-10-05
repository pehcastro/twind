package ui

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/edit"
)

func TestFieldMemoComesBack(t *testing.T) {
	long := strings.Repeat("pasted text\tthat is long ", 10)
	for _, tc := range []struct {
		why   string
		build func(e *editor)
	}{
		{"empty", func(*editor) {}},
		{"wide text across lines, selected right to left", func(e *editor) {
			e.Insert("héllo 界\nworld 👍🏽 done")
			e.Press(14, edit.Grapheme, false)
			e.Press(3, edit.Grapheme, true)
		}},
		{"a chip between words, caret inside the first word", func(e *editor) {
			e.Insert("a ")
			e.Paste(long)
			e.Insert(" b")
			e.Press(1, edit.Grapheme, false)
		}},
		{"two chips of the same length", func(e *editor) {
			e.Paste(long)
			e.Insert(" and ")
			e.Paste(strings.ToUpper(long))
		}},
	} {
		src := &editor{Buffer: edit.Buffer{Mode: edit.MultiLine}, Key: "f"}
		tc.build(src)
		dst := &editor{Buffer: edit.Buffer{Mode: edit.MultiLine}, Key: "f"}
		dst.recall(src.memo())
		if dst.Value() != src.Value() {
			t.Errorf("%s: value %q, want %q", tc.why, dst.Value(), src.Value())
		}
		if dst.Expand(dst.Value()) != src.Expand(src.Value()) {
			t.Errorf("%s: chips expand to %q, want %q", tc.why, dst.Expand(dst.Value()), src.Expand(src.Value()))
		}
		ds, de := dst.Selection()
		ss, se := src.Selection()
		dr, dc := dst.Cursor()
		sr, sc := src.Cursor()
		if ds != ss || de != se || dr != sr || dc != sc {
			t.Errorf("%s: selection %d..%d caret %d:%d, want %d..%d caret %d:%d", tc.why, ds, de, dr, dc, ss, se, sr, sc)
		}
	}
	kept := &editor{Key: "f"}
	kept.Insert("as built")
	kept.recall("{not json")
	if kept.Value() != "as built" {
		t.Errorf("a bad memo changed the field to %q", kept.Value())
	}
}

func TestOverlayMemoComesBack(t *testing.T) {
	fired := 0
	for _, open := range []bool{true, false} {
		src, dst := &overlay{Key: "o", Open: open}, &overlay{Key: "o", OnOpenChange: func(bool) { fired++ }}
		dst.recall(src.memo())
		if dst.Open != open {
			t.Errorf("open %v came back %v", open, dst.Open)
		}
	}
	if fired != 0 {
		t.Errorf("OnOpenChange fired %d times on a restore", fired)
	}
}
