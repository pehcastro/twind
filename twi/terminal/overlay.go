package terminal

import (
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"slices"
	"sync"
	"time"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
)

type Mark struct {
	Cell  image.Point
	Color color.RGBA
}

type hostPlace struct {
	origin, client image.Point
	shown, front   bool
}

type host interface {
	place() (hostPlace, error)
	capture(screen image.Rectangle) []byte
	draw(at, size image.Point, pix []byte, dirty image.Rectangle) bool
	move(at image.Point)
	show()
	hide()
	release()
}

type trace struct {
	mu sync.Mutex
	w  io.Writer
}

func (t *trace) log(format string, args ...any) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, _ = fmt.Fprintf(t.w, "%s "+format+"\n", append([]any{time.Now().Format(konst.TraceTime)}, args...)...)
}

type candidate struct {
	hwnd uintptr
	area int
}

func hostOf(parents map[uint32]uint32, shown map[uint32][]candidate, pid uint32, front uintptr, tr *trace) uintptr {
	for range konst.OverlayAncestors {
		if pid = parents[pid]; pid == 0 {
			tr.log("host: no parent left, no window")
			return 0
		}
		found := shown[pid]
		tr.log("host: ancestor pid %d has %d visible unowned windows", pid, len(found))
		if len(found) == 0 {
			continue
		}
		if slices.ContainsFunc(found, func(c candidate) bool { return c.hwnd == front }) {
			return front
		}
		return slices.MaxFunc(found, func(a, b candidate) int { return a.area - b.area }).hwnd
	}
	tr.log("host: none within %d ancestors", konst.OverlayAncestors)
	return 0
}

func locate(pix []byte, size image.Point, marks []Mark, grid image.Point) (cols, rows []int, err error) {
	n, m := grid.X, grid.Y
	if n < 2 || m < 3 || len(marks) != n+m-1 || len(pix) < size.X*size.Y*graphicskonst.GDIBytes {
		return nil, nil, errors.New("no screen or a grid too small to mark")
	}
	at := func(x, y int) color.RGBA {
		i := (y*size.X + x) * graphicskonst.GDIBytes
		return color.RGBA{R: pix[i+2], G: pix[i+1], B: pix[i], A: math.MaxUint8}
	}
	a, b, c, d := marks[0].Color, marks[1].Color, marks[n].Color, marks[n+1].Color
	pattern := func(length int, along func(int) color.RGBA, first, second color.RGBA, want int) []int {
		for i := 0; i < length; i++ {
			if along(i) != first {
				continue
			}
			edges, j := []int{i}, i+1
			for ; j < length && (along(j) == first || along(j) == second); j++ {
				if along(j) != along(j-1) {
					edges = append(edges, j)
				}
			}
			if len(edges) == want {
				return append(edges, j)
			}
			i = j - 1
		}
		return nil
	}
	across := func(y int) []int {
		return pattern(size.X, func(x int) color.RGBA { return at(x, y) }, a, b, n)
	}
	bottom := size.Y - 1
	for bottom >= 0 && across(bottom) == nil {
		bottom--
	}
	if bottom < 0 {
		return nil, nil, fmt.Errorf("no pixel line holds the %d alternating marks of the last row", n)
	}
	cols = across(bottom)
	tops := map[int]int{}
	for x := cols[0]; x < cols[n]; x++ {
		y := bottom
		for y > 0 && at(x, y-1) == at(x, bottom) {
			y--
		}
		tops[y]++
	}
	top := bottom
	for y, count := range tops {
		if count > tops[top] {
			top = y
		}
	}
	for x := cols[n-1]; x < cols[n]; x++ {
		found := pattern(top, func(y int) color.RGBA { return at(x, y) }, c, d, m-1)
		switch {
		case found == nil || found[m-1] != top:
		case rows == nil:
			rows = found
		case !slices.Equal(found, rows):
			return nil, nil, fmt.Errorf("the last column's marks disagree between x %d and the lines before it", x)
		}
	}
	if rows == nil {
		return nil, nil, fmt.Errorf("the last column's marks at x %d..%d do not run down to the last row's at y %d", cols[n-1], cols[n], top)
	}
	return cols, append(rows, bottom+1), nil
}

func remap(edges []int, cell int) []int32 {
	m := make([]int32, 0, edges[len(edges)-1]-edges[0])
	for c := range len(edges) - 1 {
		width := edges[c+1] - edges[c]
		for p := range width {
			m = append(m, int32(c*cell+min((2*p+1)*cell/(2*width), cell-1)))
		}
	}
	return m
}

func within(m []int32, lo, hi int) (from, to int) {
	from = len(m)
	for i, v := range m {
		if int(v) >= lo && int(v) < hi {
			from, to = min(from, i), i+1
		}
	}
	return from, to
}

type overlay struct {
	mu             sync.Mutex
	win            host
	cells          image.Point
	grid, client   image.Point
	able           bool
	marking        bool
	page           color.RGBA
	marked         []Mark
	painted, read  time.Time
	reads          int
	found          []int
	blurred        bool
	shown          bool
	stopped        bool
	fresh          bool
	refused        bool
	broken         bool
	trace          *trace
	origin         image.Point
	cell           image.Point
	base, size     image.Point
	columns, lines []int32
	src, dst       []byte
}

func (o *overlay) layout(page color.RGBA) []Mark {
	nudge := func(v uint8) uint8 {
		if v == math.MaxUint8 {
			return v - 1
		}
		return v + 1
	}
	a, b, c := page, page, page
	a.R, b.G, c.B = nudge(page.R), nudge(page.G), nudge(page.B)
	d := color.RGBA{R: nudge(page.R), G: nudge(page.G), B: nudge(page.B), A: page.A}
	n, m := o.grid.X, o.grid.Y
	marks := make([]Mark, 0, n+m-1)
	for x := range n {
		marks = append(marks, Mark{image.Pt(x, m-1), [2]color.RGBA{a, b}[x%2]})
	}
	for y := range m - 1 {
		marks = append(marks, Mark{image.Pt(n-1, y), [2]color.RGBA{c, d}[y%2]})
	}
	return marks
}

func (o *overlay) marks(page color.RGBA, now time.Time) []Mark {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.marking {
		return nil
	}
	if o.marked == nil || page != o.page {
		o.marked, o.page, o.painted, o.found = o.layout(page), page, now, nil
		o.trace.log("marks: %d cells on page %v", len(o.marked), page)
	}
	return o.marked
}

func (o *overlay) focus(focused bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.blurred = !focused
}

func (o *overlay) tick(grid image.Point, now time.Time) (input.ResizeEvent, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return input.ResizeEvent{}, false
	}
	at, err := o.win.place()
	was, asked := o.cell, o.refused
	o.refused = false
	if err != nil || at.client != o.client || grid != o.grid {
		o.drop(fmt.Sprintf("host %v, client %v to %v, grid %v to %v", err, o.client, at.client, o.grid, grid))
		o.client, o.grid, o.reads, o.marked = at.client, grid, 0, nil
		asked = asked || o.marking
		o.marking = false
	}
	able := err == nil && at.shown && at.front && !o.blurred
	if able && !o.able || o.reads >= konst.OverlayReads && now.Sub(o.read) >= konst.OverlayRestart {
		o.reads = 0
	}
	o.able = able
	if o.marking && able && !o.painted.IsZero() && now.Sub(o.painted) >= konst.OverlaySettle && now.Sub(o.read) >= konst.OverlayRetry {
		o.read = now
		o.reads++
		o.measure(at)
	}
	if want := able && o.cell == (image.Point{}) && !o.broken && o.reads < konst.OverlayReads; want != o.marking {
		o.marking, o.marked, o.painted, asked = want, nil, time.Time{}, true
		o.trace.log("marks: wanted %t after %d reads, able %t; the page must be one opaque colour and the profile truecolor for them to be painted", want, o.reads, able)
	}
	visible := able && o.cell != image.Point{}
	if visible && at.origin != o.origin {
		o.origin = at.origin
		o.win.move(o.origin.Add(o.base))
	}
	switch {
	case visible && !o.shown:
		o.trace.log("show at %v", o.origin.Add(o.base))
		o.win.show()
	case !visible && o.shown:
		o.trace.log("hide: host %v, shown %t, foreground %t, focus out %t", err, at.shown, at.front, o.blurred)
		o.win.hide()
	}
	o.shown = visible
	if o.cell == was && !asked {
		return input.ResizeEvent{}, false
	}
	cell := o.cell
	if cell == (image.Point{}) {
		cell = o.cells
	}
	return input.ResizeEvent{Width: grid.X, Height: grid.Y, Cell: cell}, true
}

func (o *overlay) measure(at hostPlace) {
	pix := o.win.capture(image.Rectangle{Min: at.origin, Max: at.origin.Add(at.client)})
	if pix == nil {
		o.trace.log("read %d: no screen", o.reads)
		return
	}
	cols, rows, err := locate(pix, at.client, o.marked, o.grid)
	if err != nil {
		o.trace.log("read %d: %v", o.reads, err)
		o.found = nil
		return
	}
	n, m := o.grid.X, o.grid.Y
	o.trace.log("read %d: row marks at %v, column marks at %v", o.reads, image.Rect(cols[0], rows[m-1], cols[n], rows[m]), image.Rect(cols[n-1], rows[0], cols[n], rows[m-1]))
	if edges := append(slices.Clone(cols), rows...); !slices.Equal(edges, o.found) {
		o.found = edges
		o.trace.log("read %d: waiting for a second read to verify", o.reads)
		return
	}
	o.trace.log("read %d: verified", o.reads)
	o.cell = image.Pt(max(1, int(math.Round(float64(cols[n]-cols[0])/float64(n)))), max(1, int(math.Round(float64(rows[m]-rows[0])/float64(m)))))
	o.base = image.Pt(cols[0], rows[0])
	o.size = image.Pt(cols[n], rows[m]).Sub(o.base)
	o.columns, o.lines = remap(cols, o.cell.X), remap(rows, o.cell.Y)
	o.src = make([]byte, n*o.cell.X*m*o.cell.Y*graphicskonst.GDIBytes)
	o.dst = make([]byte, o.size.X*o.size.Y*graphicskonst.GDIBytes)
	o.fresh, o.marking = true, false
	o.trace.log("calibrated: cell %dx%d from %.3fx%.3f, panel %v in Zed's client", o.cell.X, o.cell.Y, float64(o.size.X)/float64(n), float64(o.size.Y)/float64(m), image.Rectangle{Min: o.base, Max: o.base.Add(o.size)})
}

func (o *overlay) drop(why string) {
	o.trace.log("drop: %s, was calibrated %t", why, o.cell != image.Point{})
	if o.shown {
		o.win.hide()
		o.shown = false
	}
	o.cell, o.origin = image.Point{}, image.Point{}
}

func (o *overlay) surface() (Graphics, image.Point) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cell == (image.Point{}) {
		return GraphicsNone, o.cells
	}
	return GraphicsGDI, o.cell
}

func (o *overlay) paint(p Pixels) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return true
	}
	if o.cell == (image.Point{}) || p.Cell != o.cell || p.Grid != o.grid || o.fresh && !p.Clear {
		o.trace.log("paint refused: cell %v grid %v clear %t, overlay cell %v grid %v waiting for a clear %t", p.Cell, p.Grid, p.Clear, o.cell, o.grid, o.fresh)
		o.refused = o.cell != image.Point{}
		return false
	}
	o.fresh = false
	stride := o.grid.X * o.cell.X * graphicskonst.GDIBytes
	var dirty image.Rectangle
	if p.Clear {
		clear(o.src)
		dirty = image.Rectangle{Max: o.grid}
	}
	for _, tile := range p.Tiles {
		r := image.Rect(tile.Cells.Min.X*o.cell.X, tile.Cells.Min.Y*o.cell.Y, tile.Cells.Max.X*o.cell.X, tile.Cells.Max.Y*o.cell.Y)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			row := o.src[y*stride+r.Min.X*graphicskonst.GDIBytes : y*stride+r.Max.X*graphicskonst.GDIBytes]
			if tile.Pix == nil {
				clear(row)
				continue
			}
			copy(row, tile.Pix[(y-r.Min.Y)*r.Dx()*graphicskonst.GDIBytes:])
		}
		dirty = dirty.Union(tile.Cells)
	}
	var area image.Rectangle
	area.Min.X, area.Max.X = within(o.columns, dirty.Min.X*o.cell.X, dirty.Max.X*o.cell.X)
	area.Min.Y, area.Max.Y = within(o.lines, dirty.Min.Y*o.cell.Y, dirty.Max.Y*o.cell.Y)
	if area.Empty() {
		return true
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			at := (y*o.size.X + x) * graphicskonst.GDIBytes
			copy(o.dst[at:at+graphicskonst.GDIBytes], o.src[int(o.lines[y])*stride+int(o.columns[x])*graphicskonst.GDIBytes:])
		}
	}
	drawn := o.win.draw(o.origin.Add(o.base), o.size, o.dst, area)
	o.trace.log("draw at %v size %v dirty %v, ok %t", o.origin.Add(o.base), o.size, area, drawn)
	if !drawn {
		o.drop("the layered window refused to draw, cells path for good")
		o.broken, o.refused = true, true
	}
	return drawn
}

func (o *overlay) close() {
	o.mu.Lock()
	o.stopped = true
	o.drop("exit")
	o.mu.Unlock()
	o.win.release()
}
