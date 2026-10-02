package docsapp

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

type screenRows [][]rune

func (s screenRows) at(x, y int) rune {
	if y < 0 || y >= len(s) || x < 0 || x >= len(s[y]) {
		return ' '
	}
	return s[y][x]
}

func (s screenRows) box(from, left, right int) (top, l, r, bottom int, whole bool) {
	l = -1
	for top = from; top < len(s); top++ {
		line := s[top][:min(len(s[top]), right)]
		if found := slices.Index(line[min(len(line), left):], '╭'); found >= 0 {
			l = left + found
			r = l + slices.Index(s[top][l:], '╮')
			break
		}
	}
	for bottom = top + 1; l >= 0 && bottom < len(s); bottom++ {
		if s.at(l, bottom) == '╰' {
			return top, l, r, bottom, s.at(r, bottom) == '╯'
		}
	}
	return top, l, r, bottom, false
}

func TestPreviewCardInsideItsFrameWhenNarrow(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	app := func(rt *twi.Runtime) func() twi.Node {
		view, err := New(rt, Start{Page: "card", Theme: "twind-dark"})
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	for _, width := range []int{60, 80, 120} {
		d := drive.New(app, drive.Size(width, 35), drive.Styles(sheet))
		t.Cleanup(func() {
			if err := d.Err(); err != nil {
				t.Error(err)
			}
			if err := d.Close(); err != nil {
				t.Error(err)
			}
		})
		var rows screenRows
		var top, left, right, bottom int
		for notches := 0; ; notches++ {
			text := d.Frame().Text()
			t.Logf("%d columns, %d wheel notches down:\n%s", width, notches, text)
			rows = nil
			for line := range strings.SplitSeq(text, "\n") {
				rows = append(rows, []rune(line))
			}
			var whole bool
			if top, left, right, bottom, whole = rows.box(0, 0, width); whole {
				break
			}
			if notches == 4 || left < 0 {
				t.Fatalf("%d columns: no whole preview frame on screen after %d wheel notches (top %d, columns %d to %d)", width, notches, top, left, right)
			}
			d.Wheel(width/2, top, 1)
		}
		for y := top + 1; y < bottom; y++ {
			if rows.at(left, y) != '│' || rows.at(right, y) != '│' {
				t.Errorf("%d columns: row %d of the preview frame lost a side border: %q at %d, %q at %d", width, y, rows.at(left, y), left, rows.at(right, y), right)
			}
		}
		cardTop, cardLeft, cardRight, cardBottom, whole := rows.box(top+1, left+1, right)
		if !whole || cardBottom >= bottom || cardLeft <= left || cardRight >= right {
			t.Errorf("%d columns: the card (rows %d to %d, columns %d to %d) is not inside the frame (rows %d to %d, columns %d to %d)", width, cardTop, cardBottom, cardLeft, cardRight, top, bottom, left, right)
			continue
		}
		for y := cardTop + 1; y < cardBottom; y++ {
			if rows.at(cardLeft, y) != '│' || rows.at(cardRight, y) != '│' {
				t.Errorf("%d columns: row %d of the card lost a side border: %q at %d, %q at %d", width, y, rows.at(cardLeft, y), cardLeft, rows.at(cardRight, y), cardRight)
			}
		}
	}
}
