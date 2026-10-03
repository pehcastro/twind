package terminal

import (
	"image"
	"image/png"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
)

type fakeHost struct {
	mu       sync.Mutex
	at       hostPlace
	gone     bool
	screen   []byte
	covered  bool
	reads    int
	shown    bool
	moves    []image.Point
	draws    []image.Rectangle
	drawnAt  image.Point
	size     image.Point
	pix      []byte
	made     int
	bare     bool
	resized  bool
	released bool
	refused  bool
	term     *fakeTerminal
	leftLast bool
}

func (h *fakeHost) place() (hostPlace, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.gone {
		return hostPlace{}, errNoWindow
	}
	return h.at, nil
}

func (h *fakeHost) capture(r image.Rectangle) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reads++
	full := image.Rectangle{Min: h.at.origin, Max: h.at.origin.Add(h.at.client)}
	if h.covered || h.screen == nil || !r.In(full) {
		return nil
	}
	var seen []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		at := ((y-full.Min.Y)*h.at.client.X + r.Min.X - full.Min.X) * 4
		seen = append(seen, h.screen[at:at+r.Dx()*4]...)
	}
	return seen
}

func (h *fakeHost) pixels(size image.Point) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.bare {
		return nil
	}
	if size != h.size {
		h.size, h.pix = size, make([]byte, size.X*size.Y*4)
		h.made++
	}
	return h.pix
}

func (h *fakeHost) draw(at, size image.Point, dirty image.Rectangle) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if size != h.size {
		h.resized = true
	}
	h.drawnAt = at
	h.draws = append(h.draws, dirty)
	return !h.refused
}

func (h *fakeHost) move(at image.Point) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.moves = append(h.moves, at)
}

func (h *fakeHost) show() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shown = true
}

func (h *fakeHost) hide() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shown = false
}

func (h *fakeHost) release() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.released = true
	if h.term != nil {
		h.leftLast = !strings.Contains(h.term.out(), konst.LeaveScreen)
	}
}

func (h *fakeHost) set(change func(h *fakeHost)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	change(h)
}

func (h *fakeHost) state() (shown bool, moves, draws int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.shown, len(h.moves), len(h.draws)
}

var (
	zedGrid   = image.Pt(200, 34)
	zedClient = image.Pt(1950, 1000)
	zedOrigin = image.Pt(100, 50)
	zedPanel  = image.Pt(341, 276)
	zedWidth  = 7.8
	zedHeight = 16.913
	zedCell   = image.Pt(8, 17)
	zedCells  = image.Pt(7, 16)
	zedPage   = color.RGBA{R: 5, G: 4, B: 6, A: 255}
)

func column(c int) int {
	if c == zedGrid.X {
		return column(c-1) + int(math.Ceil(zedWidth))
	}
	return zedPanel.X + int(math.Floor(float64(c)*zedWidth))
}

func row(r int) int {
	return int(math.Ceil(float64(zedPanel.Y) + float64(r)*zedHeight - 0.5))
}

func zedScreen(marks []Mark) []byte {
	pix := make([]byte, zedClient.X*zedClient.Y*4)
	fill := func(r image.Rectangle, c color.RGBA) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				copy(pix[(y*zedClient.X+x)*4:], []byte{c.B, c.G, c.R, 255})
			}
		}
	}
	fill(image.Rectangle{Max: zedClient}, zedPage)
	for _, m := range marks {
		left := column(m.Cell.X)
		fill(image.Rect(left, row(m.Cell.Y), left+int(math.Ceil(zedWidth)), row(m.Cell.Y+1)), m.Color)
	}
	return pix
}

func zedHost() *fakeHost {
	return &fakeHost{at: hostPlace{origin: zedOrigin, client: zedClient, shown: true, front: true}}
}

func calibrate(t *testing.T, o *overlay, h *fakeHost, now time.Time) (time.Time, []input.ResizeEvent) {
	t.Helper()
	var resizes []input.ResizeEvent
	for range konst.OverlayReads {
		if ev, ok := o.tick(zedGrid, now); ok {
			resizes = append(resizes, ev)
		}
		if marks := o.marks(zedPage, now); marks != nil {
			h.set(func(h *fakeHost) { h.screen = zedScreen(marks) })
		}
		if g, _ := o.surface(); g == GraphicsGDI {
			return now, resizes
		}
		now = now.Add(konst.OverlayRetry)
	}
	return now, resizes
}

func TestOverlayMarksTheLastRowAndColumn(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now := time.Now()
	if o.marks(zedPage, now) != nil {
		t.Fatalf("marks before the first tick")
	}
	if ev, ok := o.tick(zedGrid, now); !ok || ev != (input.ResizeEvent{Width: zedGrid.X, Height: zedGrid.Y, Cell: zedCells}) {
		t.Errorf("marking started without asking for a frame: %+v %v", ev, ok)
	}
	marks := o.marks(zedPage, now)
	if len(marks) != zedGrid.X+zedGrid.Y-1 {
		t.Fatalf("%d marks, want %d", len(marks), zedGrid.X+zedGrid.Y-1)
	}
	a, b := color.RGBA{R: 7, G: 4, B: 6, A: 255}, color.RGBA{R: 5, G: 6, B: 6, A: 255}
	c, d := color.RGBA{R: 5, G: 4, B: 8, A: 255}, color.RGBA{R: 7, G: 6, B: 8, A: 255}
	for i, want := range []Mark{{image.Pt(0, 33), a}, {image.Pt(1, 33), b}, {image.Pt(199, 33), b}, {image.Pt(199, 0), c}, {image.Pt(199, 1), d}, {image.Pt(199, 32), c}} {
		if !slices.Contains(marks, want) {
			t.Errorf("mark %d %+v missing", i, want)
		}
	}
	white := o.marks(color.RGBA{R: 255, G: 255, B: 255, A: 255}, now)
	if !slices.Contains(white, Mark{image.Pt(0, 33), color.RGBA{R: 253, G: 255, B: 255, A: 255}}) {
		t.Errorf("a channel at 255 was not lowered: %+v", white[:2])
	}
	for _, step := range []func(h *fakeHost){
		func(h *fakeHost) { h.at.front = false },
		func(h *fakeHost) { h.at.shown = false },
	} {
		h := zedHost()
		o := &overlay{win: h, cells: zedCells}
		h.set(step)
		o.tick(zedGrid, now)
		if o.marks(zedPage, now) != nil {
			t.Errorf("marked while Zed was not in front")
		}
	}
	o.focus(false)
	if ev, ok := o.tick(zedGrid, now); !ok || o.marks(zedPage, now) != nil {
		t.Errorf("focus out: marks %v, frame asked %v %+v, want no marks and a frame that wipes them", o.marks(zedPage, now) != nil, ok, ev)
	}
}

func TestOverlayLocatesZedsCellEdges(t *testing.T) {
	marks := (&overlay{grid: zedGrid}).layout(zedPage)
	cols, rows, err := locate(zedScreen(marks), zedClient, marks, zedGrid)
	if err != nil {
		t.Fatal(err)
	}
	for c := range zedGrid.X + 1 {
		if cols[c] != column(c) {
			t.Errorf("column edge %d at %d, want %d", c, cols[c], column(c))
		}
	}
	for r := range zedGrid.Y + 1 {
		if rows[r] != row(r) {
			t.Errorf("row edge %d at %d, want %d", r, rows[r], row(r))
		}
	}
	stray := zedScreen(marks)
	for y := 100; y < 120; y++ {
		for x := 400; x < 1000; x++ {
			copy(stray[(y*zedClient.X+x)*4:], [][]byte{{6, 4, 6, 255}, {6, 5, 5, 255}, {7, 4, 5, 255}, {7, 5, 6, 255}}[(x/8+y/17)%4])
		}
	}
	if got, gotRows, err := locate(stray, zedClient, marks, zedGrid); err != nil || !slices.Equal(got, cols) || !slices.Equal(gotRows, rows) {
		t.Errorf("mark colours elsewhere on the screen changed the edges: %v", err)
	}
	for name, spoil := range map[string]func(pix []byte){
		"absent": func(pix []byte) { copy(pix, zedScreen(nil)) },
		"covered": func(pix []byte) {
			for y := row(zedGrid.Y - 1); y < row(zedGrid.Y); y++ {
				clear(pix[(y*zedClient.X+column(50))*4 : (y*zedClient.X+column(60))*4])
			}
		},
		"column covered": func(pix []byte) {
			for y := row(3); y < row(5); y++ {
				clear(pix[(y*zedClient.X+column(zedGrid.X-1))*4 : (y*zedClient.X+column(zedGrid.X))*4])
			}
		},
	} {
		pix := zedScreen(marks)
		spoil(pix)
		if _, _, err := locate(pix, zedClient, marks, zedGrid); err == nil {
			t.Errorf("%s: located", name)
		}
	}
}

func TestOverlayLocatesTheMarksZedPainted(t *testing.T) {
	f, err := os.Open("testdata/zed-marks.png")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	size := img.Bounds().Size()
	pix := make([]byte, 0, size.X*size.Y*4)
	for y := range size.Y {
		for x := range size.X {
			r, g, b, a := img.At(x, y).RGBA()
			pix = append(pix, byte(b>>8), byte(g>>8), byte(r>>8), byte(a>>8))
		}
	}
	at := func(x, y int) color.RGBA {
		i := (y*size.X + x) * 4
		return color.RGBA{R: pix[i+2], G: pix[i+1], B: pix[i], A: 255}
	}
	grid := image.Pt(151, 40)
	marks := (&overlay{grid: grid}).layout(color.RGBA{R: 5, G: 4, B: 6, A: 255})
	cols, rows, err := locate(pix, size, marks, grid)
	if err != nil {
		t.Fatal(err)
	}
	if cols[0] != 341 || cols[grid.X] != 1519 || rows[grid.Y-1] != 935 || rows[grid.Y] != 952 {
		t.Errorf("columns %d..%d rows %d..%d, want the marks the shot shows at x 341..1519 and y 935..952", cols[0], cols[grid.X], rows[grid.Y-1], rows[grid.Y])
	}
	for c := range grid.X {
		if want := marks[c].Color; !near(at(cols[c], 943), want) || !near(at(cols[c+1]-1, 943), want) || cols[c+1]-cols[c] < 7 || cols[c+1]-cols[c] > 8 {
			t.Errorf("column %d at %d..%d: pixels %v and %v, want %v, 7 or 8 px wide", c, cols[c], cols[c+1], at(cols[c], 943), at(cols[c+1]-1, 943), want)
		}
	}
	for r := range grid.Y - 1 {
		if want := marks[grid.X+r].Color; !near(at(1515, rows[r]), want) || !near(at(1515, rows[r+1]-1), want) {
			t.Errorf("row %d at %d..%d: pixels %v and %v, want %v", r, rows[r], rows[r+1], at(1515, rows[r]), at(1515, rows[r+1]-1), want)
		}
	}
}

func TestOverlayCalibratesFromTwoMatchingReads(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now, resizes := calibrate(t, o, h, time.Now())
	if g, cell := o.surface(); g != GraphicsGDI || cell != zedCell {
		t.Fatalf("graphics %d cell %v after %d reads, want GDI and %v", g, cell, h.reads, zedCell)
	}
	if h.reads != 2 || len(resizes) != 2 || resizes[1] != (input.ResizeEvent{Width: zedGrid.X, Height: zedGrid.Y, Cell: zedCell}) {
		t.Errorf("%d reads and resizes %+v, want 2 reads, a frame for the marks, then the calibrated cell", h.reads, resizes)
	}
	if o.marks(zedPage, now) != nil {
		t.Errorf("still marking once calibrated")
	}
	h = zedHost()
	o = &overlay{win: h, cells: zedCells}
	now = time.Now()
	o.tick(zedGrid, now)
	h.set(func(h *fakeHost) { h.screen = zedScreen(o.marks(zedPage, now)) })
	o.tick(zedGrid, now.Add(konst.OverlayRetry))
	panel := zedPanel
	zedPanel = zedPanel.Add(image.Pt(0, 1))
	h.set(func(h *fakeHost) { h.screen = zedScreen(o.marks(zedPage, now)) })
	zedPanel = panel
	o.tick(zedGrid, now.Add(2*konst.OverlayRetry))
	if g, _ := o.surface(); g != GraphicsNone {
		t.Errorf("calibrated from two reads that disagree")
	}
}

func TestOverlayGivesUpAndRestoresTheCells(t *testing.T) {
	h := zedHost()
	h.covered = true
	o := &overlay{win: h, cells: zedCells}
	now, _ := calibrate(t, o, h, time.Now())
	ev, wiped := o.tick(zedGrid, now)
	if o.marks(zedPage, now) != nil || h.reads != konst.OverlayReads || !wiped || ev.Cell != zedCells {
		t.Errorf("after %d failed reads: marking %v, frame asked %v %+v, want no marks after %d reads and a frame to wipe them", h.reads, o.marks(zedPage, now) != nil, wiped, ev, konst.OverlayReads)
	}
	o.tick(zedGrid, now.Add(konst.OverlayRestart/2))
	if o.marks(zedPage, now) != nil {
		t.Errorf("marking again before %v", konst.OverlayRestart)
	}
	if ev, ok := o.tick(zedGrid, now.Add(konst.OverlayRestart)); !ok || o.marks(zedPage, now) == nil {
		t.Errorf("no new marks %v after giving up, Zed still in front: %+v %v", konst.OverlayRestart, ev, ok)
	}
	h.set(func(h *fakeHost) { h.covered, h.at.front = false, false })
	o.tick(zedGrid, now)
	h.set(func(h *fakeHost) { h.at.front = true })
	if _, resizes := calibrate(t, o, h, now); len(resizes) == 0 {
		t.Errorf("did not mark again when Zed came back to the front")
	}
	if g, _ := o.surface(); g != GraphicsGDI {
		t.Errorf("not calibrated after Zed came back to the front")
	}
}

func TestOverlayFollowsZed(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	ring := image.Rect(10, 2, 18, 3)
	if o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}}) {
		t.Errorf("a paint before calibration was accepted")
	}
	now, _ := calibrate(t, o, h, time.Now())
	base := zedOrigin.Add(zedPanel)
	if shown, moves, _ := h.state(); shown || moves == 0 || h.moves[moves-1] != base {
		t.Fatalf("after calibration shown %v moved to %v, want hidden at %v until the first full paint", shown, h.moves, base)
	}
	if o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}}) {
		t.Errorf("a paint without Clear after calibration was accepted")
	}
	if _, ok := o.tick(zedGrid, now); !ok {
		t.Errorf("a refused paint did not ask for a frame")
	}
	if o.paint(Pixels{Cell: zedCells, Grid: zedGrid, Clear: true}) {
		t.Errorf("a paint for the old cell was accepted")
	}
	pix := opaque(ring, zedCell)
	for i := 0; i < len(pix); i += 4 {
		if (i/4)%(ring.Dx()*zedCell.X)/zedCell.X == 12-ring.Min.X {
			pix[i+3] = 0
		}
	}
	if !o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: pix}}}) {
		t.Fatalf("the first full paint was refused")
	}
	placed(t, h, ring, 12)
	if shown, _, _ := h.state(); !shown {
		t.Errorf("not shown after the first full paint")
	}

	_, moves, draws := h.state()
	h.set(func(h *fakeHost) { h.at.origin = h.at.origin.Add(image.Pt(30, -20)) })
	reads := h.reads
	o.tick(zedGrid, now)
	if _, m, d := h.state(); m != moves+1 || h.moves[m-1] != base.Add(image.Pt(30, -20)) || d != draws || h.reads != reads {
		t.Errorf("Zed moved: %d moves to %v, %d draws, %d reads, want one move to %v and nothing else", m-moves, h.moves[m-1], d-draws, h.reads-reads, base.Add(image.Pt(30, -20)))
	}
	for _, step := range []struct {
		name   string
		grid   image.Point
		change func(h *fakeHost)
	}{
		{"panel resized", image.Pt(201, 34), func(*fakeHost) {}},
		{"Zed resized", zedGrid, func(h *fakeHost) { h.at.client = image.Pt(1951, 1000) }},
		{"Zed closed", zedGrid, func(h *fakeHost) { h.gone = true }},
	} {
		h := zedHost()
		o := &overlay{win: h, cells: zedCells}
		now, _ := calibrate(t, o, h, time.Now())
		h.set(step.change)
		ev, ok := o.tick(step.grid, now)
		if !ok || ev != (input.ResizeEvent{Width: step.grid.X, Height: step.grid.Y, Cell: zedCells}) {
			t.Errorf("%s: event %+v %v, want a resize back to %v", step.name, ev, ok, zedCells)
		}
		if shown, _, _ := h.state(); shown {
			t.Errorf("%s: still shown", step.name)
		}
		if g, cell := o.surface(); g != GraphicsNone || cell != zedCells {
			t.Errorf("%s: graphics %d cell %v, want none and %v", step.name, g, cell, zedCells)
		}
		if o.paint(Pixels{Cell: zedCell, Grid: step.grid, Clear: true}) {
			t.Errorf("%s: a paint was accepted after the calibration was dropped", step.name)
		}
		if marking := o.marks(zedPage, now) != nil; marking == (step.name == "Zed closed") {
			t.Errorf("%s: marking %v", step.name, marking)
		}
	}
}

func TestOverlayRecalibratesAfterAResize(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now, _ := calibrate(t, o, h, time.Now())
	h.set(func(h *fakeHost) { h.at.client = image.Pt(1951, 1000) })
	zedClient = image.Pt(1951, 1000)
	defer func() { zedClient = image.Pt(1950, 1000) }()
	if _, resizes := calibrate(t, o, h, now); len(resizes) < 2 || resizes[len(resizes)-1].Cell != zedCell {
		t.Errorf("after a resize: %+v, want a frame for the marks and then the calibrated cell", resizes)
	}
	if g, _ := o.surface(); g != GraphicsGDI {
		t.Errorf("not calibrated again after the resize")
	}
}

func placed(t *testing.T, h *fakeHost, ring image.Rectangle, text int) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	base := h.drawnAt.Sub(zedOrigin)
	if base != zedPanel || h.size != image.Pt(column(zedGrid.X), row(zedGrid.Y)).Sub(zedPanel) {
		t.Fatalf("overlay at %v size %v, want %v size %v", base, h.size, zedPanel, image.Pt(column(zedGrid.X), row(zedGrid.Y)).Sub(zedPanel))
	}
	alpha := func(x, y int) byte { return h.pix[((y-base.Y)*h.size.X+x-base.X)*4+3] }
	middle := (row(ring.Min.Y) + row(ring.Max.Y)) / 2
	for x := base.X; x < base.X+h.size.X; x++ {
		inside := x >= column(ring.Min.X) && x < column(ring.Max.X) && (x < column(text) || x >= column(text+1))
		if got := alpha(x, middle) == 255; got != inside {
			t.Errorf("column %d drawn %v, want %v by Zed's cell edges", x, got, inside)
		}
	}
	for y := base.Y; y < base.Y+h.size.Y; y++ {
		inside := y >= row(ring.Min.Y) && y < row(ring.Max.Y)
		if got := alpha(column(14), y) == 255; got != inside {
			t.Errorf("row %d drawn %v, want %v by Zed's cell edges", y, got, inside)
		}
	}
}

func TestOverlayPaintsTilesInPlace(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	calibrate(t, o, h, time.Now())
	width := zedGrid.X * zedCell.X
	surface := make([]byte, width*zedGrid.Y*zedCell.Y*4)
	tile := func(cells image.Rectangle, seed byte) Tile {
		pix := make([]byte, cells.Dx()*zedCell.X*cells.Dy()*zedCell.Y*4)
		for i := range pix {
			pix[i] = seed + byte(i*7+i/1031)
		}
		return Tile{Cells: cells, Pix: pix}
	}
	hole := func(cells image.Rectangle) Tile { return Tile{Cells: cells} }
	area := func(cells image.Rectangle) image.Rectangle {
		var r image.Rectangle
		r.Min.X, r.Max.X = within(o.columns, cells.Min.X*zedCell.X, cells.Max.X*zedCell.X)
		r.Min.Y, r.Max.Y = within(o.lines, cells.Min.Y*zedCell.Y, cells.Max.Y*zedCell.Y)
		return r
	}
	for _, step := range []struct {
		name  string
		clear bool
		tiles []Tile
	}{
		{"clear with two tiles", true, []Tile{tile(image.Rect(10, 2, 18, 8), 1), tile(image.Rect(150, 30, 158, 34), 2)}},
		{"a tile and a hole over the first", false, []Tile{tile(image.Rect(40, 10, 48, 16), 3), hole(image.Rect(12, 4, 16, 6))}},
		{"two far tiles around the first", false, []Tile{tile(image.Rect(0, 0, 8, 6), 4), tile(image.Rect(192, 28, 200, 34), 5)}},
		{"a narrow tile on uneven columns", false, []Tile{tile(image.Rect(97, 17, 99, 18), 6)}},
		{"clear again with one tile", true, []Tile{tile(image.Rect(60, 20, 68, 26), 7)}},
	} {
		if step.clear {
			clear(surface)
		}
		want := image.Rectangle{}
		if step.clear {
			want = image.Rectangle{Max: o.size}
		}
		for _, tl := range step.tiles {
			r := image.Rect(tl.Cells.Min.X*zedCell.X, tl.Cells.Min.Y*zedCell.Y, tl.Cells.Max.X*zedCell.X, tl.Cells.Max.Y*zedCell.Y)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				row := surface[(y*width+r.Min.X)*4 : (y*width+r.Max.X)*4]
				if tl.Pix == nil {
					clear(row)
					continue
				}
				copy(row, tl.Pix[(y-r.Min.Y)*r.Dx()*4:])
			}
			want = want.Union(area(tl.Cells))
		}
		if !o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: step.clear, Tiles: step.tiles}) {
			t.Fatalf("%s: refused", step.name)
		}
		h.mu.Lock()
		if got := h.draws[len(h.draws)-1]; got != want {
			t.Errorf("%s: dirty %v, want the tiles' overlay area %v", step.name, got, want)
		}
		wrong := 0
		for y := range o.size.Y {
			for x := range o.size.X {
				at, from := (y*o.size.X+x)*4, (int(o.lines[y])*width+int(o.columns[x]))*4
				if !slices.Equal(h.pix[at:at+4], surface[from:from+4]) {
					if wrong++; wrong == 1 {
						t.Errorf("%s: overlay pixel (%d,%d) is %v, want Twind pixel (%d,%d) %v", step.name, x, y, h.pix[at:at+4], o.columns[x], o.lines[y], surface[from:from+4])
					}
				}
			}
		}
		if wrong > 0 {
			t.Errorf("%s: %d overlay pixels wrong", step.name, wrong)
		}
		if h.made != 1 || h.resized {
			t.Errorf("%s: the overlay surface was made %d times, a draw for another size %v", step.name, h.made, h.resized)
		}
		h.mu.Unlock()
	}
}

func TestOverlayRefusedDrawFallsBackToCells(t *testing.T) {
	for _, why := range []string{"refused draw", "no surface"} {
		h := zedHost()
		o := &overlay{win: h, cells: zedCells}
		now, _ := calibrate(t, o, h, time.Now())
		h.set(func(h *fakeHost) { h.refused, h.bare = why == "refused draw", why == "no surface" })
		ring := image.Rect(10, 2, 18, 3)
		if o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}}) {
			t.Errorf("%s: the paint was reported drawn", why)
		}
		if ev, ok := o.tick(zedGrid, now); !ok || ev.Cell != zedCells {
			t.Errorf("%s: event %+v %v, want a resize back to %v", why, ev, ok, zedCells)
		}
		calibrate(t, o, h, now)
		if g, cell := o.surface(); g != GraphicsNone || cell != zedCells || o.marks(zedPage, now) != nil {
			t.Errorf("%s: graphics %d cell %v marking %v, want none, %v and no marks for good", why, g, cell, o.marks(zedPage, now) != nil, zedCells)
		}
		if shown, _, _ := h.state(); shown {
			t.Errorf("%s: the window is shown", why)
		}
	}
}

func TestOverlayChosenForZedAlacrittyAndRio(t *testing.T) {
	newerConPTY := []string{"\x1b[6;16;8t", "\x1b[4;576;960t\x1b[?61;6;7;14;21;22;23;24;28;32;42c"}
	newerConPTYSixel := []string{"\x1b[6;16;8t", "\x1b[4;576;960t\x1b[?61;4;6;7;14;21;22;23;24;28;32;42c"}
	cases := []struct {
		name       string
		answers    []string
		conhost    bool
		host       bool
		exe, class string
		offer      offer
		chosen     bool
	}{
		{"zed with a window", zed, false, true, "Zed.exe", "Zed::Window", offer{zed: true}, true},
		{"zed without a window", zed, false, false, "", "", offer{zed: true}, false},
		{"zed forced none", zed, false, true, "Zed.exe", "Zed::Window", offer{zed: true, forced: true}, false},
		{"windows terminal", windowsTerminal, false, true, "WindowsTerminal.exe", "CASCADIA_HOSTING_WINDOW_CLASS", offer{}, false},
		{"windows terminal under an alacritty window", windowsTerminal, false, true, "alacritty.exe", konst.WinitClass, offer{}, false},
		{"conhost", conPTY, true, true, "alacritty.exe", konst.WinitClass, offer{}, false},
		{"inbox conpty in mintty", conPTY, false, true, "mintty.exe", "mintty", offer{}, false},
		{"inbox conpty in another winit program", conPTY, false, true, "foo.exe", konst.WinitClass, offer{}, false},
		{"inbox conpty in alacritty with another class", conPTY, false, true, "alacritty.exe", "Alacritty", offer{}, false},
		{"kitty", kitty, false, true, "kitty.exe", konst.WinitClass, offer{}, false},
		{"stock alacritty", conPTY, false, true, "alacritty.exe", konst.WinitClass, offer{}, true},
		{"stock alacritty without a window", conPTY, false, false, "alacritty.exe", konst.WinitClass, offer{}, false},
		{"stock alacritty forced none", conPTY, false, true, "alacritty.exe", konst.WinitClass, offer{forced: true}, false},
		{"stock rio", conPTY, false, true, "Rio.exe", konst.WinitClass, offer{}, true},
		{"alacritty with a newer conpty", newerConPTY, false, true, "Alacritty.exe", konst.WinitClass, offer{}, true},
		{"rio with a newer conpty draws sixel", newerConPTYSixel, false, true, "rio.exe", konst.WinitClass, offer{}, false},
	}
	for _, tc := range cases {
		term := newFake(tc.answers...)
		h := zedHost()
		term.tty.window, term.tty.exe, term.tty.class = tc.conhost, tc.exe, tc.class
		if tc.host {
			term.tty.host = h
		}
		b, err := enter(term, term.tty, Options{}, tc.offer)
		if err != nil {
			t.Fatal(err)
		}
		if got := b.overlay.Load() != nil; got != tc.chosen {
			t.Errorf("%s: overlay %v, want %v", tc.name, got, tc.chosen)
		}
		if caps := b.Current(); caps.Graphics == GraphicsGDI {
			t.Errorf("%s: GDI before any calibration", tc.name)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		if tc.host && h.released != tc.chosen {
			t.Errorf("%s: host released %v, want %v", tc.name, h.released, tc.chosen)
		}
		term = newFake(append([]string{"\x1b[3;1R"}, tc.answers...)...)
		term.tty.window, term.tty.exe, term.tty.class = tc.conhost, tc.exe, tc.class
		term.tty.host = zedHost()
		if caps, _, err := query(term, term.tty, tc.offer); err != nil || caps.Graphics == GraphicsGDI || term.tty.host.released {
			t.Errorf("%s: query err %v graphics %d touched the host %v", tc.name, err, caps.Graphics, term.tty.host.released)
		}
	}
}

func TestOverlayCalibratesOnAnIntegerGrid(t *testing.T) {
	exact := func(v uint8) uint8 { return v }
	rio := func(v uint8) uint8 {
		if v >= 46 {
			return v - 1
		}
		return v
	}
	for _, tc := range []struct {
		name string
		page color.RGBA
		read func(uint8) uint8
	}{
		{"alacritty", color.RGBA{R: 30, G: 30, B: 46, A: 255}, exact},
		{"rio reads 46 as 45", color.RGBA{R: 30, G: 30, B: 46, A: 255}, rio},
		{"rio merges 45 and 46", color.RGBA{R: 30, G: 30, B: 45, A: 255}, rio},
		{"alacritty near white nudges down", color.RGBA{R: 250, G: 252, B: 255, A: 255}, exact},
		{"alacritty next to a rewritten grey nudges down", color.RGBA{R: 10, G: 10, B: 10, A: 255}, exact},
	} {
		grid, client, panel, cell := image.Pt(120, 36), image.Pt(980, 600), image.Pt(5, 5), image.Pt(8, 16)
		h := &fakeHost{at: hostPlace{origin: image.Pt(40, 60), client: client, shown: true, front: true}}
		o := &overlay{win: h}
		now := time.Now()
		for range konst.OverlayReads {
			o.tick(grid, now)
			if marks := o.marks(tc.page, now); marks != nil {
				pix := make([]byte, client.X*client.Y*4)
				for i := 0; i < len(pix); i += 4 {
					copy(pix[i:], []byte{tc.read(tc.page.B), tc.read(tc.page.G), tc.read(tc.page.R), 255})
				}
				for _, m := range marks {
					for y := range cell.Y {
						for x := range cell.X {
							copy(pix[((panel.Y+m.Cell.Y*cell.Y+y)*client.X+panel.X+m.Cell.X*cell.X+x)*4:], []byte{tc.read(m.Color.B), tc.read(m.Color.G), tc.read(m.Color.R), 255})
						}
					}
				}
				h.set(func(h *fakeHost) { h.screen = pix })
			}
			now = now.Add(konst.OverlayRetry)
		}
		size := image.Pt(grid.X*cell.X, grid.Y*cell.Y)
		if g, got := o.surface(); g != GraphicsGDI || got != cell || o.base != panel || o.size != size {
			t.Errorf("%s: graphics %d cell %v base %v size %v after %d reads, want GDI, %v, %v and %v", tc.name, g, got, o.base, o.size, h.reads, cell, panel, size)
		}
	}
}

func TestOverlayMarksAvoidTheGreysTheInboxConPTYRewrites(t *testing.T) {
	rewritten := []uint8{12, 118, 204, 242}
	var pages []color.RGBA
	for _, g := range append([]uint8{0, 1, 253, 254, 255}, rewritten...) {
		for d := -3; d <= 3; d++ {
			v := uint8(min(max(int(g)+d, 0), 255))
			pages = append(pages, color.RGBA{R: v, G: v, B: v, A: 255}, color.RGBA{R: v, G: g, B: g, A: 255}, color.RGBA{R: g, G: v, B: g, A: 255}, color.RGBA{R: g, G: g, B: v, A: 255})
		}
	}
	grid := image.Pt(3, 3)
	for _, page := range pages {
		marks := (&overlay{grid: grid}).layout(page)
		four := []color.RGBA{marks[0].Color, marks[1].Color, marks[grid.X].Color, marks[grid.X+1].Color}
		for i, m := range four {
			if m.R == m.G && m.G == m.B && slices.Contains(rewritten, m.R) {
				t.Errorf("page %v: mark %d is %v, a grey the inbox ConPTY rewrites", page, i, m)
			}
			if near(m, page) {
				t.Errorf("page %v: mark %d is %v, within the slack of the page", page, i, m)
			}
			for j, other := range four[:i] {
				if near(m, other) {
					t.Errorf("page %v: marks %d and %d are %v and %v, within the slack of each other", page, j, i, other, m)
				}
			}
		}
	}
}

func TestOverlayDrawsOverZedAlacrittyAndRioOnly(t *testing.T) {
	for _, tc := range []struct {
		id         Identity
		exe, class string
		want       bool
	}{
		{IdentityZed, "Zed.exe", "Zed::Window", true},
		{IdentityZed, "", "", true},
		{IdentityInboxConPTY, "alacritty.exe", konst.WinitClass, true},
		{IdentityInboxConPTY, "RIO.EXE", konst.WinitClass, true},
		{IdentityOther, "alacritty.exe", konst.WinitClass, true},
		{IdentityOther, "rio.exe", konst.WinitClass, true},
		{IdentityInboxConPTY, "alacritty.exe", "Alacritty", false},
		{IdentityInboxConPTY, "foo.exe", konst.WinitClass, false},
		{IdentityInboxConPTY, "mintty.exe", "mintty", false},
		{IdentityOther, "WindowsTerminal.exe", "CASCADIA_HOSTING_WINDOW_CLASS", false},
		{IdentityConhost, "alacritty.exe", konst.WinitClass, false},
		{IdentityVSCode, "alacritty.exe", konst.WinitClass, false},
	} {
		if got := drawsOver(tc.id, tc.exe, tc.class); got != tc.want {
			t.Errorf("identity %d in %s class %q: overlay %v, want %v", tc.id, tc.exe, tc.class, got, tc.want)
		}
	}
}

func TestOverlayTraceNamesEachStep(t *testing.T) {
	var lines strings.Builder
	h := zedHost()
	o := &overlay{win: h, cells: zedCells, trace: &trace{w: &lines}}
	h.covered = true
	now := time.Now()
	o.tick(zedGrid, now)
	o.marks(zedPage, now)
	o.tick(zedGrid, now.Add(konst.OverlayRetry))
	h.covered = false
	now, _ = calibrate(t, o, h, now.Add(2*konst.OverlayRetry))
	ring := image.Rect(10, 2, 18, 3)
	o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}})
	h.set(func(h *fakeHost) { h.at.shown = false })
	o.tick(zedGrid, now)
	o.tick(image.Pt(201, 34), now)
	for _, want := range []string{"marks: 233 cells", "read 1: no screen", "read 2: no screen", "row marks at (341,834)-(1901,851)", "column marks at (1893,276)-(1901,834)", "read 4: verified", "calibrated: cell 8x17", "draw at", "ok true", "show at", "hide: host <nil>, shown false", "grid (200,34) to (201,34), was calibrated true"} {
		if !strings.Contains(lines.String(), want) {
			t.Errorf("trace has no %q:\n%s", want, lines.String())
		}
	}
}

func TestOverlayTraceOnlyWithTheVariable(t *testing.T) {
	path := t.TempDir() + "/trace.txt"
	for _, env := range []map[string]string{{}, {"TERM_PROGRAM": "zed", konst.TraceEnv: path}} {
		o, err := offered(func(k string) string { return env[k] })
		if err != nil || o.trace != env[konst.TraceEnv] {
			t.Fatalf("offer %+v %v, want the trace path %q", o, err, env[konst.TraceEnv])
		}
		term := newFake(zed...)
		term.tty.host = zedHost()
		b, err := enter(term, term.tty, Options{}, o)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(got), "overlay: opened") {
		t.Errorf("trace file %q %v, want the overlay opening", got, err)
	}
}

func TestOverlayHostIsTheForegroundOrLargestWindow(t *testing.T) {
	parents := map[uint32]uint32{10: 20, 20: 30, 30: 40, 40: 0}
	shown := map[uint32][]candidate{30: {{1, 100}, {2, 900}, {3, 400}}, 40: {{9, 5000}}}
	for _, tc := range []struct {
		name  string
		front uintptr
		want  uintptr
	}{
		{"foreground", 3, 3},
		{"largest", 9, 2},
		{"none", 0, 2},
	} {
		if got := hostOf(parents, shown, 10, tc.front, nil); got != tc.want {
			t.Errorf("%s: host %d, want %d", tc.name, got, tc.want)
		}
	}
	long := map[uint32]uint32{}
	for pid := uint32(1); pid <= konst.OverlayAncestors+1; pid++ {
		long[pid] = pid + 1
	}
	if got := hostOf(long, map[uint32][]candidate{konst.OverlayAncestors + 2: {{7, 1}}}, 1, 0, nil); got != 0 {
		t.Errorf("a host past %d ancestors was taken: %d", konst.OverlayAncestors, got)
	}
}

func TestOverlayKeepsZedBytes(t *testing.T) {
	out := func(host *fakeHost) string {
		term := newFake(zed...)
		term.tty.host = host
		b, err := enter(term, term.tty, Options{}, offer{zed: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
		if err := b.Exit(); err != nil {
			t.Fatal(err)
		}
		return term.out()
	}
	if with, without := out(zedHost()), out(nil); with != without {
		t.Errorf("Zed bytes with an overlay differ:\n got %q\nwant %q", with, without)
	}
}

func TestOverlayCalibratesFromScreenMarks(t *testing.T) {
	h := zedHost()
	term := newFake(zed...)
	term.tty.host, h.term = h, term
	b, err := enter(term, term.tty, Options{}, offer{zed: true})
	if err != nil {
		t.Fatal(err)
	}
	var resizes []input.ResizeEvent
	deadline := time.After(2 * time.Second)
	for len(resizes) == 0 || resizes[len(resizes)-1].Cell != zedCell {
		if marks := b.Marks(zedPage); marks != nil {
			h.set(func(h *fakeHost) { h.screen = zedScreen(marks) })
		}
		select {
		case ev := <-b.Events:
			if r, ok := ev.(input.ResizeEvent); ok {
				resizes = append(resizes, r)
			}
		case <-time.After(konst.OverlayRetry):
		case <-deadline:
			t.Fatalf("not calibrated within 2 s: resizes %+v, %d reads", resizes, h.reads)
		}
	}
	if caps := b.Current(); caps.Graphics != GraphicsGDI || caps.CellPixels != zedCell {
		t.Errorf("after calibration: graphics %d cell %v, want GDI and %v", caps.Graphics, caps.CellPixels, zedCell)
	}
	ring := image.Rect(10, 2, 18, 3)
	if !b.Paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}}) {
		t.Errorf("the backend refused a paint for the calibrated cell")
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	if shown, _, _ := h.state(); shown || !h.released || !h.leftLast {
		t.Errorf("after Exit: shown %v released %v before leaving %v, want hidden and released before the leave sequence", shown, h.released, h.leftLast)
	}
	_, _, draws := h.state()
	b.Paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: ring, Pix: opaque(ring, zedCell)}}})
	if _, _, now := h.state(); now != draws {
		t.Errorf("a paint after Exit drew")
	}
}

func TestOverlayStaysShownWithoutFocus(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now, _ := calibrate(t, o, h, time.Now())
	o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true})
	reads := h.reads
	for _, step := range []struct {
		name  string
		front bool
		focus bool
	}{
		{"another window in front", false, true},
		{"focus out with Zed in front", true, false},
		{"another window in front and focus out", false, false},
	} {
		h.set(func(h *fakeHost) { h.at.front = step.front })
		o.focus(step.focus)
		if ev, ok := o.tick(zedGrid, now); ok {
			t.Errorf("%s: asked for a frame %+v", step.name, ev)
		}
		if shown, _, _ := h.state(); !shown {
			t.Errorf("%s: hidden", step.name)
		}
		if g, cell := o.surface(); g != GraphicsGDI || cell != zedCell {
			t.Errorf("%s: graphics %d cell %v, want GDI and %v", step.name, g, cell, zedCell)
		}
	}
	if h.reads != reads+1 || o.marks(zedPage, now) != nil {
		t.Errorf("read the screen %d times or marked without focus, want one check with Zed in front and focus out", h.reads-reads)
	}
}

func TestOverlayHidesWhenItsCellsLeaveTheScreen(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now, _ := calibrate(t, o, h, time.Now())
	top := image.Rect(0, 0, zedGrid.X, 1)
	full := Pixels{Cell: zedCell, Grid: zedGrid, Clear: true, Tiles: []Tile{{Cells: top, Pix: opaque(top, zedCell)}}}
	o.paint(full)
	other := color.RGBA{R: zedPage.R, G: zedPage.G, B: 200, A: 255}
	ours, foreign := zedScreen(nil), zedScreen(nil)
	for i := 0; i < len(foreign); i += 4 {
		foreign[i] = other.B
	}
	copy(ours[row(0)*zedClient.X*4:], foreign[row(0)*zedClient.X*4:row(1)*zedClient.X*4])
	step := func(name string, screen []byte, change func(h *fakeHost), wantReads int, want Graphics, wantEvent bool) {
		t.Helper()
		h.set(func(h *fakeHost) {
			h.screen = screen
			change(h)
		})
		reads := h.reads
		now = now.Add(konst.OverlayCoverPoll)
		ev, ok := o.tick(zedGrid, now)
		g, cell := o.surface()
		if h.reads-reads != wantReads || g != want || ok != wantEvent || ok && ev.Cell != cell {
			t.Errorf("%s: %d reads, graphics %d cell %v, event %+v %v; want %d reads, graphics %d, event %v", name, h.reads-reads, g, cell, ev, ok, wantReads, want, wantEvent)
		}
		if shown, _, _ := h.state(); want == GraphicsNone && shown || want == GraphicsGDI && !ok && !shown {
			t.Errorf("%s: shown %v with graphics %d", name, shown, g)
		}
	}
	same := func(*fakeHost) {}
	for range 5 {
		step("focused, another tab", foreign, same, 0, GraphicsGDI, false)
	}
	o.focus(false)
	step("focus out, Twind's page on the line", ours, same, 1, GraphicsGDI, false)
	step("focus out, another tab", foreign, same, 1, GraphicsNone, true)
	if o.paint(full) {
		t.Errorf("a pixel paint was accepted over another tab")
	}
	h.set(func(h *fakeHost) { h.screen = ours })
	reads := h.reads
	if o.tick(zedGrid, now.Add(konst.OverlayPoll)); h.reads != reads {
		t.Errorf("checked again before %v", konst.OverlayCoverPoll)
	}
	step("Twind's cells back", ours, same, 1, GraphicsGDI, true)
	if shown, _, _ := h.state(); shown {
		t.Errorf("shown with the bitmap from before the other tab")
	}
	if !o.paint(full) {
		t.Errorf("the full paint after coming back was refused")
	}
	step("another tab, Zed not in front", foreign, func(h *fakeHost) { h.at.front = false }, 0, GraphicsGDI, false)
	step("no screen", nil, func(h *fakeHost) { h.at.front = true }, 1, GraphicsGDI, false)
	step("another tab again", foreign, same, 1, GraphicsNone, true)
	o.marks(other, now)
	step("the page changed while hidden", foreign, same, 1, GraphicsGDI, true)
	o.paint(full)
	step("another tab with the new page", ours, same, 1, GraphicsNone, true)
	o.focus(true)
	step("focus in", ours, same, 0, GraphicsGDI, true)
	o.paint(full)
	for range 5 {
		step("focused again", ours, same, 0, GraphicsGDI, false)
	}
}

func TestOverlayHiddenGivesTheCellsLook(t *testing.T) {
	h := zedHost()
	o := &overlay{win: h, cells: zedCells}
	now, _ := calibrate(t, o, h, time.Now())
	o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true})
	reads := h.reads
	for _, name := range []string{"minimised", "minimised again"} {
		h.set(func(h *fakeHost) { h.at.shown = false })
		if ev, ok := o.tick(zedGrid, now); !ok || ev != (input.ResizeEvent{Width: zedGrid.X, Height: zedGrid.Y, Cell: zedCells}) {
			t.Errorf("%s: event %+v %v, want one frame in the cells look at %v", name, ev, ok, zedCells)
		}
		if _, ok := o.tick(zedGrid, now); ok {
			t.Errorf("%s: a second frame asked", name)
		}
		if shown, _, _ := h.state(); shown {
			t.Errorf("%s: still shown", name)
		}
		if g, cell := o.surface(); g != GraphicsNone || cell != zedCells {
			t.Errorf("%s: graphics %d cell %v, want none and %v like the cells path", name, g, cell, zedCells)
		}
		if o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true}) {
			t.Errorf("%s: a pixel paint was accepted while hidden", name)
		}
		h.set(func(h *fakeHost) { h.at.shown = true })
		if ev, ok := o.tick(zedGrid, now); !ok || ev.Cell != zedCell {
			t.Errorf("%s and back: event %+v %v, want one frame at %v", name, ev, ok, zedCell)
		}
		if shown, _, _ := h.state(); shown {
			t.Errorf("%s and back: shown with the bitmap from before", name)
		}
		if o.paint(Pixels{Cell: zedCell, Grid: zedGrid}) {
			t.Errorf("%s and back: a paint without Clear was accepted", name)
		}
		if _, ok := o.tick(zedGrid, now); !ok {
			t.Errorf("%s and back: the refused paint asked for no frame", name)
		}
		if !o.paint(Pixels{Cell: zedCell, Grid: zedGrid, Clear: true}) {
			t.Errorf("%s and back: the full paint was refused", name)
		}
		if shown, _, _ := h.state(); !shown {
			t.Errorf("%s and back: not shown after the full paint", name)
		}
	}
	if h.reads != reads || o.marks(zedPage, now) != nil {
		t.Errorf("recalibrated after a minimise: %d reads", h.reads-reads)
	}
}

func TestOverlayHiddenReportsWhatZedWithoutOverlayReports(t *testing.T) {
	caps := func(host *fakeHost) Capabilities {
		term := newFake(zed...)
		if host != nil {
			term.tty.host = host
		}
		b, err := enter(term, term.tty, Options{}, offer{zed: true})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = b.Exit() }()
		if o := b.overlay.Load(); o != nil {
			calibrate(t, o, host, time.Now())
			host.set(func(h *fakeHost) { h.at.shown = false })
			o.tick(zedGrid, time.Now())
		}
		return b.Current()
	}
	with, without := caps(zedHost()), caps(nil)
	if with.Graphics != without.Graphics || with.CellPixels != without.CellPixels {
		t.Errorf("a hidden overlay reports graphics %d cell %v, Zed without one %d %v", with.Graphics, with.CellPixels, without.Graphics, without.CellPixels)
	}
}
