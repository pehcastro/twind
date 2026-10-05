package docsapp

import (
	"strings"
	"testing"
)

func TestDocsStartWithEveryGroupOpen(t *testing.T) {
	d := open(t)
	text := d.Frame().Text()
	t.Logf("start:\n%s", text)
	for _, row := range []string{"Introduction", "Layout", "Accordion"} {
		if !strings.Contains(text, row) {
			t.Errorf("the start frame has no %q in the sidebar:\n%s", row, text)
		}
	}
}

func TestKeptViewsFollowCopyFoldsAndSidebar(t *testing.T) {
	d := open(t)
	left, top := spot(t, d, "Docs ›")
	for range 40 {
		if strings.Contains(d.Frame().Text(), "Copy") {
			break
		}
		d.Wheel(left+2, top+2, 1)
	}
	x, y := spot(t, d, "Copy")
	d.Click(x+1, y)
	if _, at := spot(t, d, "Copied"); at != y {
		t.Errorf("Copied on row %d after a click on the introduction's code block on row %d", at, y)
	}
	t.Logf("after the click:\n%s", d.Frame().Text())
	d.Advance(copiedFor)
	if text := d.Frame().Text(); strings.Contains(text, "Copied") {
		t.Errorf("the introduction's code block still says Copied %v later:\n%s", copiedFor, text)
	}
	gx, gy := spot(t, d, "Components")
	d.Click(gx+1, gy)
	d.Advance(settle)
	if text := d.Frame().Text(); strings.Contains(text, "Accordion") {
		t.Errorf("a click on Components left its pages shown:\n%s", text)
	}
	d.Click(gx+1, gy)
	d.Advance(settle)
	if text := d.Frame().Text(); !strings.Contains(text, "Accordion") {
		t.Errorf("a second click on Components left its pages hidden:\n%s", text)
	}
	d.Press("ctrl+b")
	d.Advance(settle)
	if text := d.Frame().Text(); strings.Contains(text, "Accordion") {
		t.Errorf("ctrl+b left the sidebar open:\n%s", text)
	}
}
