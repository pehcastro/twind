package layout

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

func clone(b *Box) *Box {
	c := &Box{Style: b.Style, Measure: b.Measure, ScrollX: b.ScrollX, ScrollY: b.ScrollY, ScrollWidth: b.ScrollWidth, ScrollHeight: b.ScrollHeight}
	for _, child := range b.Children {
		c.Children = append(c.Children, clone(child))
	}
	return c
}

func boxes(b *Box, into []*Box) []*Box {
	into = append(into, b)
	for _, c := range b.Children {
		into = boxes(c, into)
	}
	return into
}

func tries(lay func(*Box, int, Length), root *Box, width int, height Length) (failed bool) {
	defer func() { failed = recover() != nil }()
	lay(root, width, height)
	return false
}

func outputs(b *Box) string {
	return fmt.Sprint(b.BorderBox, b.PaddingBox, b.ContentBox, b.Clip, b.ScrollX, b.ScrollY, b.ScrollWidth, b.ScrollHeight)
}

func unmarked(b *Box, before map[*Box]string, path string) string {
	if !b.Moved && outputs(b) != before[b] {
		return path + " changed without Moved"
	}
	for i, c := range b.Children {
		if c.Moved && !b.Moved {
			return fmt.Sprintf("%s/%d moved, its parent is not Moved", path, i)
		}
		if d := unmarked(c, before, fmt.Sprintf("%s/%d", path, i)); d != "" {
			return d
		}
	}
	return ""
}

func differ(got, want *Box, path string) string {
	if g, w := outputs(got), outputs(want); g != w {
		return fmt.Sprintf("box %s: %s, want %s", path, g, w)
	}
	for i := range got.Children {
		if d := differ(got.Children[i], want.Children[i], fmt.Sprintf("%s/%d", path, i)); d != "" {
			return d
		}
	}
	return ""
}

func edit(r *rand.Rand, root *Box) string {
	all := boxes(root, nil)
	b := all[r.IntN(len(all))]
	switch r.IntN(8) {
	case 0:
		if len(b.Children) > 0 {
			return "none"
		}
		b.Measure = text(strings.Repeat("w", r.IntN(14))).Measure
		if r.IntN(2) == 0 {
			b.Measure = wrap(r.IntN(80)).Measure
		}
		b.Invalidate()
		return "text"
	case 1:
		b.Style = randomTree(r, 1).Style
		b.Invalidate()
		return "class"
	case 2:
		if b.Measure != nil {
			return "none"
		}
		at := r.IntN(len(b.Children) + 1)
		b.Children = slices.Insert(b.Children, at, randomTree(r, r.IntN(3)))
		b.Invalidate()
		return "add"
	case 3:
		if len(b.Children) == 0 {
			return "none"
		}
		at := r.IntN(len(b.Children))
		b.Children = slices.Delete(b.Children, at, at+1)
		b.Invalidate()
		return "remove"
	case 4:
		s, from := &b.Style, randomTree(r, 0).Style
		s.Position, s.Inset, s.Column, s.Row, s.Margin = from.Position, from.Inset, from.Column, from.Row, from.Margin
		if r.IntN(3) == 0 {
			s.Position, s.Inset = PositionSticky, Insets{randomLength(r, 6), randomLength(r, 6), randomLength(r, 6), randomLength(r, 6)}
		}
		b.ScrollX, b.ScrollY = r.IntN(20), r.IntN(20)
		b.Invalidate()
		return "place"
	case 5:
		hidden := slices.DeleteFunc(all, visible)
		if len(hidden) > 0 && r.IntN(2) == 0 {
			b = hidden[r.IntN(len(hidden))]
		}
		shown := visible(b)
		b.Style.Display = DisplayFlex
		if shown {
			b.Style.Display = DisplayNone
		}
		b.Invalidate()
		return "display"
	case 6:
		b.Style.Width, b.Style.Height, b.Style.Overflow = randomLength(r, 3), randomLength(r, 3), Overflow(r.IntN(3))
		b.Invalidate()
		return "boundary"
	}
	from, to := slices.IndexFunc(all, func(p *Box) bool { return slices.Contains(p.Children, b) }), all[r.IntN(len(all))]
	if from < 0 || to.Measure != nil || slices.Contains(boxes(b, nil), to) {
		return "none"
	}
	parent := all[from]
	parent.Children = slices.DeleteFunc(parent.Children, func(c *Box) bool { return c == b })
	to.Children = append(to.Children, b)
	parent.Invalidate()
	to.Invalidate()
	return "move"
}

func TestRelayoutMatchesFullLayout(t *testing.T) {
	edits := 0
	for seed := range 100 {
		r := rand.New(rand.NewPCG(114, uint64(seed)))
		root := randomTree(r, 1+r.IntN(4))
		width, height := 1+r.IntN(100), Length{}
		tries(Layout, root, width, height)
		for done := 0; done < 30; {
			kind := edit(r, root)
			if kind == "none" {
				continue
			}
			done, edits = done+1, edits+1
			if r.IntN(3) == 0 {
				width, height = 1+r.IntN(100), Length{}
				if r.IntN(2) == 0 {
					height = cells(r.IntN(50))
				}
			}
			fresh, before := clone(root), map[*Box]string{}
			for _, b := range boxes(root, nil) {
				b.Moved, before[b] = false, outputs(b)
			}
			failed, freshFailed := tries(Layout, root, width, height), tries(full, fresh, width, height)
			if failed != freshFailed {
				t.Fatalf("seed %d edit %d (%s): relayout panics %v, full layout panics %v", seed, edits, kind, failed, freshFailed)
			}
			if failed {
				continue
			}
			if d := differ(root, fresh, ""); d != "" {
				t.Fatalf("seed %d edit %d (%s) at %d x %v: %s", seed, edits, kind, width, height, d)
			}
			if d := unmarked(root, before, ""); d != "" {
				t.Fatalf("seed %d edit %d (%s): box %s", seed, edits, kind, d)
			}
		}
	}
	if edits != 3000 {
		t.Fatalf("%d edits, want 3000", edits)
	}
}

func TestRelayoutFixedBoxStopsOnlyWhenClipped(t *testing.T) {
	for _, overflow := range []Overflow{OverflowVisible, OverflowHidden} {
		for _, direction := range []Direction{Row, Column} {
			inner := wrap(3)
			fixed := box(Style{Width: cells(10), Height: cells(6), Shrink: 1, Overflow: overflow}, inner)
			root := box(Style{Direction: direction, Width: cells(4), Height: cells(3)}, fixed, box(Style{Width: cells(4), Height: cells(3), Shrink: 1}))
			Layout(root, 20, Length{})
			inner.Measure = func(int) (int, int) { return 7, 4 }
			inner.Invalidate()
			fresh := clone(root)
			Layout(root, 20, Length{})
			full(fresh, 20, Length{})
			if d := differ(root, fresh, ""); d != "" {
				t.Errorf("overflow %d, direction %d: %s", overflow, direction, d)
			}
		}
	}
}

func TestRelayoutAfterAPanic(t *testing.T) {
	changed, broken := text("a"), box(Style{})
	root := box(Style{Direction: Column}, changed, box(Style{Width: cells(5), Height: cells(2), Overflow: OverflowHidden}, broken))
	Layout(root, 20, Length{})
	changed.Measure = func(int) (int, int) { return 3, 4 }
	changed.Invalidate()
	broken.Style.Display = DisplayNone + 1
	broken.Invalidate()
	if !tries(Layout, root, 20, Length{}) {
		t.Fatal("an unknown display did not panic")
	}
	broken.Style.Display = DisplayFlex
	broken.Invalidate()
	fresh := clone(root)
	Layout(root, 20, Length{})
	full(fresh, 20, Length{})
	if d := differ(root, fresh, ""); d != "" {
		t.Error(d)
	}
}
