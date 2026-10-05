package terminal

import (
	"errors"
	"image"
	"math"
	"strings"
	"sync"
	"time"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
)

type Tile struct {
	Cells  image.Rectangle
	Pix    []byte
	Glyphs []Glyph
}

type Glyph struct {
	Cell    image.Point
	Cluster string
	Fg      color.RGBA
	Bold    bool
	Wide    bool
}

type Pixels struct {
	Cell  image.Point
	Grid  image.Point
	Clear bool
	Tiles []Tile
}

type window interface {
	client() (image.Point, error)
	blit(pix []byte, size image.Point, r image.Rectangle)
	read(size image.Point, r image.Rectangle) []byte
	invalidate(r image.Rectangle)
	release()
}

var errNoWindow = errors.New("terminal: no console window to draw on")

type geometry struct{ client, grid image.Point }

func (g geometry) cell() image.Point {
	return image.Pt(g.client.X/g.grid.X, g.client.Y/g.grid.Y)
}

type canvas struct {
	mu      sync.Mutex
	win     window
	shape   geometry
	cell    image.Point
	pix     []byte
	drawn   image.Rectangle
	trace   *trace
	wrote   time.Time
	dropped bool
	seen    geometry
	since   time.Time
	asked   bool
	settle  time.Time
	repaint time.Time
	stopped bool
	wake    chan struct{}
	done    chan struct{}
	ended   chan struct{}
}

func openCanvas(t tty, tr *trace) (*canvas, error) {
	win, err := t.drawable()
	if err != nil {
		return nil, err
	}
	c := &canvas{win: win, trace: tr, wake: make(chan struct{}, 1), done: make(chan struct{}), ended: make(chan struct{})}
	cols, rows, err := t.size()
	g, ok := c.measure(cols, rows)
	if err != nil || !ok || win.read(g.client, image.Rect(0, 0, 1, 1)) == nil {
		win.release()
		return nil, errors.Join(err, errNoWindow)
	}
	c.adopt(g)
	go c.run()
	return c, nil
}

func (c *canvas) at(x, y int) int { return (y*c.shape.client.X + x) * graphicskonst.GDIBytes }

func (c *canvas) measure(cols, rows int) (geometry, bool) {
	client, err := c.win.client()
	g := geometry{client, image.Pt(cols, rows)}
	return g, err == nil && cols > 0 && rows > 0 && client.X >= cols && client.Y >= rows
}

func (c *canvas) adopt(g geometry) {
	c.shape, c.cell, c.drawn, c.dropped = g, g.cell(), image.Rectangle{}, false
	c.pix = make([]byte, g.client.X*g.client.Y*graphicskonst.GDIBytes)
}

func (c *canvas) drop(client image.Point) {
	if !c.dropped {
		c.dropped, c.drawn, c.asked = true, image.Rectangle{}, false
		c.win.invalidate(image.Rectangle{Max: client})
	}
}

func (c *canvas) refit(cols, rows int, now time.Time) (image.Point, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.measure(cols, rows)
	switch {
	case c.stopped, !c.dropped && ok && g == c.shape:
		return c.cell, false
	case !c.dropped:
		c.drop(g.client)
	}
	if g != c.seen {
		c.seen, c.since, c.asked = g, now, false
	}
	if !ok || c.asked || now.Sub(c.since) < konst.GDIResizeSettle {
		return c.cell, false
	}
	c.asked = true
	return g.cell(), true
}

func (c *canvas) paint(p Pixels, cols, rows int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, ok := c.measure(cols, rows)
	switch {
	case c.stopped:
		return true
	case ok && p.Clear && p.Grid == g.grid && p.Cell == g.cell():
		if !c.drawn.Empty() {
			c.win.invalidate(c.drawn)
		}
		c.adopt(g)
	case !ok || c.dropped || g != c.shape || p.Grid != g.grid || p.Cell != c.cell:
		c.drop(g.client)
		return false
	}
	var dirty image.Rectangle
	for _, tile := range p.Tiles {
		r := image.Rect(tile.Cells.Min.X*c.cell.X, tile.Cells.Min.Y*c.cell.Y, tile.Cells.Max.X*c.cell.X, tile.Cells.Max.Y*c.cell.Y)
		at := r.Intersect(image.Rectangle{Max: c.shape.client})
		for y := at.Min.Y; y < at.Max.Y; y++ {
			row := c.pix[c.at(at.Min.X, y):c.at(at.Max.X, y)]
			if tile.Pix == nil {
				clear(row)
				continue
			}
			copy(row, tile.Pix[((y-r.Min.Y)*r.Dx()+at.Min.X-r.Min.X)*graphicskonst.GDIBytes:])
		}
		if tile.Pix == nil {
			c.win.invalidate(at)
			continue
		}
		dirty, c.drawn = dirty.Union(at), c.drawn.Union(at)
	}
	if !dirty.Empty() {
		c.win.blit(c.pix, c.shape.client, dirty)
		c.trace.log("gdi: painted %v, %v after the write", dirty, time.Since(c.wrote))
	}
	c.settled()
	return true
}

func (c *canvas) written() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wrote = time.Now()
	c.settled()
}

func (c *canvas) settled() {
	c.settle, c.repaint = time.Now().Add(konst.GDISettle), time.Now().Add(konst.GDIRepaintWindow)
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *canvas) check() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.dropped || c.drawn.Empty() {
		return
	}
	if client, err := c.win.client(); err != nil || client != c.shape.client {
		c.drop(client)
		return
	}
	seen := c.win.read(c.shape.client, c.drawn)
	for y := c.drawn.Min.Y; y < c.drawn.Max.Y && seen != nil; y++ {
		for i := c.at(c.drawn.Min.X, y); i < c.at(c.drawn.Max.X, y); i += graphicskonst.GDIBytes {
			if c.pix[i+3] != 0 && (c.pix[i] != seen[i] || c.pix[i+1] != seen[i+1] || c.pix[i+2] != seen[i+2]) {
				c.win.blit(c.pix, c.shape.client, c.drawn)
				c.trace.log("gdi: the console painted over pixel %d,%d, redrawn %v after the write", i/graphicskonst.GDIBytes%c.shape.client.X, i/graphicskonst.GDIBytes/c.shape.client.X, time.Since(c.wrote))
				return
			}
		}
	}
}

func (c *canvas) run() {
	defer close(c.ended)
	timer := time.NewTimer(konst.GDIIdlePoll)
	defer timer.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-c.wake:
		case <-timer.C:
			c.check()
		}
		c.mu.Lock()
		poll := konst.GDIIdlePoll
		switch now := time.Now(); {
		case now.Before(c.repaint):
			poll = konst.GDIRepaintPoll
		case now.Before(c.settle):
			poll = konst.GDISettlePoll
		}
		c.mu.Unlock()
		timer.Reset(poll)
	}
}

func (c *canvas) close() {
	c.mu.Lock()
	c.stopped = true
	if !c.drawn.Empty() {
		c.win.invalidate(c.drawn)
	}
	c.mu.Unlock()
	close(c.done)
	<-c.ended
	c.win.release()
}

type glyphKey struct {
	cluster string
	size    image.Point
	bold    bool
}

func (b *Backend) coverage(g Glyph, size image.Point) []uint8 {
	key := glyphKey{g.Cluster, size, g.Bold}
	if mask, drawn := b.glyphs[key]; drawn {
		return mask
	}
	if b.glyphs == nil || len(b.glyphs) >= konst.GlyphCache {
		b.glyphs = map[glyphKey][]uint8{}
	}
	for face := range strings.SplitSeq(konst.GlyphFaces, "|") {
		if !b.tty.lacks(face, g.Cluster) {
			b.glyphs[key] = b.tty.glyph(face, g.Cluster, size, g.Bold)
			break
		}
	}
	return b.glyphs[key]
}

func (b *Backend) ink(p Pixels) {
	for _, tile := range p.Tiles {
		width := tile.Cells.Dx() * p.Cell.X
		for _, g := range tile.Glyphs {
			size := p.Cell
			if g.Wide {
				size.X *= 2
			}
			mask, at := b.coverage(g, size), g.Cell.Sub(tile.Cells.Min)
			for y := range len(mask) / size.X {
				for x := range min(size.X, width-at.X*p.Cell.X) {
					a := uint32(mask[y*size.X+x])
					if a == 0 {
						continue
					}
					px := tile.Pix[((at.Y*p.Cell.Y+y)*width+at.X*p.Cell.X+x)*graphicskonst.GDIBytes:][:graphicskonst.GDIBytes]
					for i, v := range [4]uint32{uint32(g.Fg.B), uint32(g.Fg.G), uint32(g.Fg.R), math.MaxUint8} {
						px[i] = uint8((v*a + uint32(px[i])*(math.MaxUint8-a) + math.MaxUint8/2) / math.MaxUint8)
					}
				}
			}
		}
	}
}

func (b *Backend) Paint(p Pixels) bool {
	if o := b.overlay.Load(); o != nil {
		b.ink(p)
		return o.paint(p)
	}
	c := b.canvas.Load()
	if c == nil {
		return true
	}
	b.ink(p)
	cols, rows, err := b.tty.size()
	return err == nil && c.paint(p, cols, rows)
}
