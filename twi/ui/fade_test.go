package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

const pageLetter = "q"

func pageOfLetters(width, rows int) []twi.NodeOption {
	line := strings.TrimSpace(strings.Repeat(pageLetter+pageLetter+pageLetter+" ", width/4))
	var lines []twi.NodeOption
	for range rows {
		lines = append(lines, twi.Element(twi.Text(line)))
	}
	return lines
}

type panel struct{ top, bottom, left, right int }

func panelOf(t *testing.T, f drive.Frame, first, last string) panel {
	t.Helper()
	x, top, ok := at(f, first)
	_, bottom, found := at(f, last)
	if !ok || !found {
		t.Fatalf("no %q or %q in the settled frame:\n%s", first, last, f.Text())
	}
	row := []rune(strings.Split(f.Text(), "\n")[top])
	p := panel{top: top, bottom: bottom, left: x, right: x}
	for p.left > 0 && row[p.left] != '│' {
		p.left--
	}
	for p.right < len(row)-1 && row[p.right] != '│' {
		p.right++
	}
	return p
}

func noPageInside(t *testing.T, name, label string, d *drive.Driver, p panel) {
	t.Helper()
	const step, logged = 10 * time.Millisecond, 50 * time.Millisecond
	drawn := 0
	for elapsed := time.Duration(0); elapsed <= settleTime/5; elapsed += step {
		lines := strings.Split(d.Frame().Text(), "\n")
		if _, _, shown := at(d.Frame(), label); shown {
			drawn++
			for y := p.top; y <= p.bottom; y++ {
				if inner := string([]rune(lines[y])[p.left+2 : p.right-1]); strings.Contains(inner, pageLetter) {
					t.Errorf("%s +%v: the page shows inside the panel on row %d: %q\n%s", name, elapsed, y, inner, d.Frame().Text())
				}
			}
		}
		if elapsed == logged {
			t.Logf("%s +%v:\n%s", name, elapsed, strings.Join(lines[max(p.top-2, 0):min(p.bottom+3, len(lines))], "\n"))
		}
		d.Advance(step)
	}
	if drawn < 3 {
		t.Errorf("%s: the panel was drawn in %d sampled frames, want the whole motion sampled", name, drawn)
	}
}

func TestFadingOverlaysShowNoPageInsideThePanel(t *testing.T) {
	const width, height = 60, 20
	d := overlayDriver(t, width, height, func(rt *twi.Runtime) func() twi.Node {
		m := NewDropdownMenu(rt)
		return func() twi.Node {
			return twi.Element(append([]twi.NodeOption{
				twi.Class("flex flex-col h-full bg-background text-foreground"),
				m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Open")), m.Content(m.Item("Profile"), m.Item("Billing"), m.Item("Team"), m.Item("Log out"))),
			}, pageOfLetters(width, height-3)...)...)
		}
	})
	x, y, _ := at(d.Frame(), "Open")
	settledClick(d, x, y)
	open := panelOf(t, d.Frame(), "Profile", "Log out")
	t.Logf("dropdown open, settled:\n%s", d.Frame().Text())
	d.Press("Escape")
	noPageInside(t, "dropdown closing", "Profile", d, open)
	d.Click(x, y)
	noPageInside(t, "dropdown opening", "Profile", d, open)

	dd := overlayDriver(t, width, height, func(rt *twi.Runtime) func() twi.Node {
		dialog := NewDialog(rt)
		return func() twi.Node {
			return twi.Element(append([]twi.NodeOption{
				twi.Class("flex flex-col h-full bg-background text-foreground"),
				dialog.Trigger(Outline, SizeDefault, twi.Text("Edit")),
				dialog.Content(
					dialog.Header(dialog.Title(twi.Text("Edit profile")), dialog.Description(twi.Text("Make changes here."))),
					dialog.Footer(dialog.Close(Outline, SizeDefault, twi.Text("Cancel"))),
				),
			}, pageOfLetters(width, height-3)...)...)
		}
	})
	x, y, _ = at(dd.Frame(), "Edit")
	settledClick(dd, x, y)
	shown := panelOf(t, dd.Frame(), "Edit profile", "Cancel")
	t.Logf("dialog open, settled:\n%s", dd.Frame().Text())
	cx, cy, _ := at(dd.Frame(), "Cancel")
	dd.Click(cx, cy)
	noPageInside(t, "dialog closing", "Edit profile", dd, shown)
	dd.Click(x, y)
	noPageInside(t, "dialog opening", "Edit profile", dd, shown)
}
