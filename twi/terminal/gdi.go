package terminal

import (
	"errors"
	"image"
	"sync"
	"time"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

type Tile struct {
	Cells image.Rectangle
	Pix   []byte
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
	dropped bool
	seen    geometry
	since   time.Time
	asked   bool
	settle  time.Time
	stopped bool
	wake    chan struct{}
	done    chan struct{}
	ended   chan struct{}
}

func openCanvas(t tty) (*canvas, error) {
	win, err := t.drawable()
	if err != nil {
		return nil, err
	}
	c := &canvas{win: win, wake: make(chan struct{}, 1), done: make(chan struct{}), ended: make(chan struct{})}
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
	}
	c.settled()
	return true
}

func (c *canvas) written() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.settled()
}

func (c *canvas) settled() {
	c.settle = time.Now().Add(konst.GDISettle)
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
		if time.Now().Before(c.settle) {
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

func (b *Backend) Paint(p Pixels) bool {
	c := b.canvas.Load()
	if c == nil {
		return true
	}
	cols, rows, err := b.tty.size()
	return err == nil && c.paint(p, cols, rows)
}
