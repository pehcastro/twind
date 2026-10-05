package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	ekonst "github.com/pehcastro/twind/internal/konst/edit"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
)

type prompt struct {
	*drive.Driver
	area *Textarea
	line *Input
	sent []string
	keys []input.Key
}

func promptDriver(t *testing.T, width string, submits bool) *prompt {
	t.Helper()
	p := &prompt{}
	p.Driver = overlayDriver(t, 40, 16, func(rt *twi.Runtime) func() twi.Node {
		p.area, p.line = NewTextarea(rt), NewInput(rt)
		if submits {
			p.area.OnSubmit = func(s string) { p.sent = append(p.sent, s) }
			p.line.OnSubmit = p.area.OnSubmit
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				twi.OnKeyDown(func(e *twi.Event) { p.keys = append(p.keys, e.Key.Key) }),
				p.area.Node(twi.Class(width), twi.AutoFocus()), p.line.Node(twi.Class(width)))
		}
	})
	return p
}

func (p *prompt) typed(s string, keys string) {
	p.Type(s)
	hit(p.Driver, keys)
}

func (p *prompt) rows() []string {
	var rows []string
	for line := range strings.SplitSeq(p.Frame().Text(), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "│") {
			rows = append(rows, strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "│")))
		}
	}
	return rows
}

func TestTextareaEnterSubmitsShiftEnterAndCtrlJBreak(t *testing.T) {
	p := promptDriver(t, "w-24", true)
	hit(p.Driver, "type:one shift+enter type:two ctrl+j type:three")
	if got := p.area.Value(); got != "one\ntwo\nthree" {
		t.Fatalf("the textarea holds %q, want three lines:\n%s", got, p.Frame().Text())
	}
	t.Logf("three lines, broken with shift+enter and ctrl+j:\n%s", p.Frame().Text())
	hit(p.Driver, "enter")
	if !slices.Equal(p.sent, []string{"one\ntwo\nthree"}) || p.area.Value() != "" {
		t.Fatalf("enter sent %q and left %q, want the three lines sent and an empty field", p.sent, p.area.Value())
	}
	p.typed("   ", "enter")
	if len(p.sent) != 1 || p.area.Value() != "   " {
		t.Fatalf("enter on spaces sent %q and left %q", p.sent, p.area.Value())
	}
}

func TestTextareaEnterWithoutSubmitReachesTheParent(t *testing.T) {
	p := promptDriver(t, "w-24", false)
	hit(p.Driver, "type:ab enter")
	if p.area.Value() != "ab" || !slices.Contains(p.keys, input.KeyEnter) {
		t.Fatalf("enter with no Submit left %q and the parent heard %v", p.area.Value(), p.keys)
	}
}

func TestInputHistoryWalksSentText(t *testing.T) {
	p := promptDriver(t, "w-24", true)
	hit(p.Driver, "tab type:first enter type:second enter type:dr up")
	if got := p.line.Value(); got != "second" {
		t.Fatalf("up holds %q, want second", got)
	}
	hit(p.Driver, "up")
	if got := p.line.Value(); got != "first" {
		t.Fatalf("up up holds %q, want first", got)
	}
	hit(p.Driver, "down down")
	if got := p.line.Value(); got != "dr" {
		t.Fatalf("down past the newest holds %q, want the draft", got)
	}
}

func TestTextareaWrapsAndMovesByRow(t *testing.T) {
	p := promptDriver(t, "w-24", false)
	p.typed("the quick brown fox jumps over the lazy dog", "")
	t.Logf("a wrapped line:\n%s", p.Frame().Text())
	if got, want := p.rows()[:3], []string{"the quick brown fox", "jumps over the lazy", "dog"}; !slices.Equal(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
	hit(p.Driver, "up")
	if row, column := p.area.Cursor(); row != 1 || column != 3 {
		t.Fatalf("up from the end lands on row %d column %d, want 1 3", row, column)
	}
}

func TestTextareaCaretAtAWrapIsDrawnOnce(t *testing.T) {
	p := promptDriver(t, "w-10", false)
	p.typed("ab cdefgh", "ctrl+left")
	caret := zinc(t, theme.Light).Tokens[theme.Foreground].RGBA
	cells, carets := p.Frame(), 0
	for y := range cells.Height() {
		for x := range cells.Width() {
			if cells.At(x, y).Bg.RGBA == caret {
				carets++
			}
		}
	}
	if carets != 1 {
		t.Fatalf("%d caret cells, want 1:\n%s", carets, p.Frame().Text())
	}
}

func TestTextareaGrowsToEightRowsThenScrolls(t *testing.T) {
	p := promptDriver(t, "w-24", false)
	for i := range 10 {
		if i > 0 {
			p.Press("shift+enter")
		}
		p.Type(fmt.Sprintf("row%02d", i+1))
	}
	hit(p.Driver, "end")
	text := p.Frame().Text()
	if strings.Contains(text, "row02") || !strings.Contains(text, "row03") || !strings.Contains(text, "row10") {
		t.Fatalf("ten lines do not show rows 3 to 10:\n%s", text)
	}
	hit(p.Driver, "ctrl+home")
	text = p.Frame().Text()
	if !strings.Contains(text, "row01") || strings.Contains(text, "row09") {
		t.Fatalf("ctrl+home does not scroll back to rows 1 to 8:\n%s", text)
	}
}

func wrappedRows(f drive.Frame) []string {
	var rows []string
	for line := range strings.SplitSeq(f.Text(), "\n") {
		if row := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "│")); strings.HasPrefix(strings.TrimSpace(line), "│") && row != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

func TestTextareaWrapsTextSetByTheProgramAndOnResize(t *testing.T) {
	d := overlayDriver(t, 40, 12, func(rt *twi.Runtime) func() twi.Node {
		area := NewTextarea(rt)
		area.Insert("the quick brown fox jumps over the lazy dog and the cat")
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), area.Node())
		}
	})
	t.Logf("set by the program, 40 columns:\n%s", d.Frame().Text())
	if got, want := wrappedRows(d.Frame()), []string{"the quick brown fox jumps over", "the lazy dog and the cat"}; !slices.Equal(got, want) {
		t.Fatalf("rows %q, want %q", got, want)
	}
	d.Resize(24, 12)
	d.Advance(settleTime)
	t.Logf("after a resize to 24 columns:\n%s", d.Frame().Text())
	if got, want := wrappedRows(d.Frame()), []string{"the quick brown", "fox jumps over", "the lazy dog and", "the cat"}; !slices.Equal(got, want) {
		t.Fatalf("rows after the resize %q, want %q", got, want)
	}
}

func TestTextareaPasteChipIsTinted(t *testing.T) {
	d := overlayDriver(t, 60, 8, func(rt *twi.Runtime) func() twi.Node {
		area := NewTextarea(rt)
		area.Insert("look ")
		area.Paste(strings.Repeat("the gate reads the policy before the wire. ", 10))
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col p-1 h-full bg-background text-foreground"), area.Node())
		}
	})
	x, y, ok := at(d.Frame(), "look [Text 430 characters]")
	if !ok {
		t.Fatalf("no chip in the field:\n%s", d.Frame().Text())
	}
	cells := d.Frame()
	if text, chip := cells.At(x, y).Fg.RGBA, cells.At(x+len("look "), y).Fg.RGBA; text == chip {
		t.Fatalf("the chip is drawn in the text colour %v", text)
	}
}

func TestTextareaCopiesTheSelection(t *testing.T) {
	p := promptDriver(t, "w-24", false)
	p.typed("copy me", "ctrl+g ctrl+shift+c")
	if got := p.Clipboard(); got != "copy me" {
		t.Fatalf("the clipboard holds %q, want the selection", got)
	}
}

func TestEditorsUndoOneBurstAfterAPause(t *testing.T) {
	p := promptDriver(t, "w-24", false)
	for _, field := range []interface{ Value() string }{p.area, p.line} {
		p.Type("one")
		p.Advance(ekonst.UndoPause)
		p.Type("two")
		p.Press("ctrl+z")
		if got := field.Value(); got != "one" {
			t.Errorf("one undo after a pause left %q, want only the second burst gone", got)
		}
		p.Press("tab")
	}
}
