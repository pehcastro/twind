package app

import (
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pehcastro/twind/twi/drive"
)

func TestSpringMenuAndFlipReorder(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(100, 30), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	}()
	run(d, "motion")
	frames := map[string]string{}
	snap := func(name string) {
		frames[name] = d.Frame().Text()
		t.Logf("%s:\n%s", name, frames[name])
	}
	snap("rest")
	spot := func(label string) (int, int) {
		for y, line := range strings.Split(frames["rest"], "\n") {
			if i := strings.Index(line, label); i >= 0 {
				return utf8.RuneCountInString(line[:i]), y
			}
		}
		t.Fatalf("%q is not on the motion page", label)
		return 0, 0
	}
	ys := func(name string, labels []string) []int {
		lines := strings.Split(frames[name], "\n")
		at := make([]int, len(labels))
		for i, label := range labels {
			at[i] = slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, label) })
		}
		return at
	}
	shown := func(name string, labels []string) int {
		return len(slices.DeleteFunc(ys(name, labels), func(y int) bool { return y < 0 }))
	}
	stacked := func(at []int) bool {
		for i := 1; i < len(at); i++ {
			if at[i] != at[0]+i {
				return false
			}
		}
		return at[0] >= 0
	}
	menu := []string{"Profile", "Billing", "Settings", "Log out"}
	mx, my := spot("Open menu")
	d.Click(mx, my)
	snap("menu-017ms")
	d.Advance(100 * time.Millisecond)
	snap("menu-117ms")
	d.Advance(900 * time.Millisecond)
	snap("menu-1017ms")
	d.Advance(time.Second)
	snap("menu-2017ms")
	d.Click(mx, my)
	d.Advance(100 * time.Millisecond)
	snap("close-117ms")
	d.Advance(900 * time.Millisecond)
	snap("closed")
	if n := shown("menu-017ms", menu); n >= len(menu) {
		t.Errorf("menu-017ms: %d of %d items on the first frame, the spring has not started", n, len(menu))
	}
	if n := shown("menu-117ms", menu); n == 0 || n == len(menu) {
		t.Errorf("menu-117ms: %d of %d items, want the menu mid-spring", n, len(menu))
	}
	if !stacked(ys("menu-1017ms", menu)) {
		t.Errorf("menu-1017ms: items at rows %v, want the open menu", ys("menu-1017ms", menu))
	}
	if frames["menu-1017ms"] != frames["menu-2017ms"] {
		t.Error("the menu still moves a second after it settled")
	}
	if n := shown("close-117ms", menu); n == 0 || n == len(menu) {
		t.Errorf("close-117ms: %d of %d items, want the menu mid-spring", n, len(menu))
	}
	if pageArea(frames["closed"]) != pageArea(frames["rest"]) {
		t.Error("the closed menu left cells behind")
	}

	rows := []string{"Inbox", "Draft", "Notes", "Queue"}
	before := ys("rest", rows)
	rx, ry := spot("Rotate")
	d.Click(rx, ry)
	d.Advance(100 * time.Millisecond)
	snap("rotate-117ms")
	d.Click(rx, ry)
	d.Advance(33 * time.Millisecond)
	snap("again-050ms")
	d.Advance(2 * time.Second)
	snap("rotated")
	if !stacked(before) {
		t.Fatalf("rest: rows at %v, want them stacked in order", before)
	}
	rotated := func(by int) []int {
		at := make([]int, len(before))
		for i := range at {
			at[i] = before[0] + (i+by)%len(at)
		}
		return at
	}
	mid := ys("rotate-117ms", rows)
	if slices.Equal(mid, before) || slices.Equal(mid, rotated(1)) {
		t.Errorf("rotate-117ms: rows at %v, want them between %v and %v", mid, before, rotated(1))
	}
	for i, y := range ys("again-050ms", rows) {
		if y >= 0 && mid[i] >= 0 && max(y-mid[i], mid[i]-y) > 1 {
			t.Errorf("%s jumped from row %d to %d when rotated mid-motion", rows[i], mid[i], y)
		}
	}
	if got := ys("rotated", rows); !slices.Equal(got, rotated(2)) {
		t.Errorf("rotated: rows at %v, want %v after rotating twice", got, rotated(2))
	}
}

func pageArea(frame string) string {
	lines := strings.Split(strings.TrimRight(frame, "\n"), "\n")
	return strings.Join(lines[:len(lines)-5], "\n")
}
