package present

import (
	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/terminal"
)

type rings struct {
	encoder graphics.Sixel
	runs    []run
	starts  []int
	lines   [][]run
}

func (s *Screen) edgedCell(x, y int) bool {
	if !s.edged(x, y) {
		return false
	}
	r, c, at := s.area(x, y)
	middle := c.line(r.Min.Y + s.Cell.Y/2)
	for py := r.Min.Y; py < r.Max.Y; py++ {
		if !same(c.line(py), middle, at, at+s.Cell.X) {
			return true
		}
	}
	return false
}

func (s *Screen) restoreRings() {
	start := s.out.Len()
	for _, r := range s.runs {
		if r.Y == s.rows-1 && s.Cell.Y%graphicskonst.SixelBand != 0 {
			continue
		}
		row := s.want.Row(r.Y)
		for x := r.X; x < r.X+r.Len; x++ {
			from := x
			for x < r.X+r.Len && (row[x].Grapheme != "" || row[x].Width == buffer.Continuation) && s.edgedCell(x, r.Y) {
				x++
			}
			if x > from {
				s.ring(from, x, r.Y)
			}
		}
	}
	if s.out.Len() > start {
		s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	}
}

func (s *Screen) ring(from, to, y int) {
	g := &s.rings
	g.runs, g.starts, g.lines = g.runs[:0], g.starts[:0], g.lines[:0]
	middle := y*s.Cell.Y + s.Cell.Y/2
	for py := y * s.Cell.Y; py < (y+1)*s.Cell.Y; py++ {
		g.starts = append(g.starts, len(g.runs))
		for x := from; x < to; x++ {
			_, c, at := s.area(x, y)
			line, left, right := c.line(py), int32((x-from)*s.Cell.X-at), int32(at+s.Cell.X)
			if same(line, c.line(middle), at, at+s.Cell.X) {
				g.runs = append(g.runs, run{End: left + right})
				continue
			}
			for i := find(line, at); i < len(line) && (i == 0 || line[i-1].End < right); i++ {
				pixel := line[i].Pixel
				if pixel>>24 == 0 {
					pixel = s.page
				}
				g.runs = append(g.runs, run{End: left + min(line[i].End, right), Pixel: pixel})
			}
		}
	}
	g.starts = append(g.starts, len(g.runs))
	for i := 1; i < len(g.starts); i++ {
		g.lines = append(g.lines, g.runs[g.starts[i-1]:g.starts[i]])
	}
	at := graphics.Placement{Col: from, Row: y, Cols: to - from, Rows: 1}
	s.out.Write(g.encoder.Encode(s.out.AvailableBuffer(), g.lines, at))
	if s.owners != nil {
		s.own(at)
	}
}
