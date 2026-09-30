package present

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"image"
	"io"
	"slices"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

type Screen struct {
	Out      io.Writer
	Profile  color.Profile
	Graphics terminal.Graphics
	Cell     image.Point
	Sync     bool
	Margins  bool

	cols, rows        int
	cell              image.Point
	painter           paint.Painter
	fresh             bool
	text, shown, want *buffer.Buffer
	out               bytes.Buffer
	writer            terminal.Writer

	turn         int
	scenes       [2]scene.Frame
	bounds       image.Rectangle
	columns      []column
	workers      []*worker
	pending      []pending
	drawn        []bool
	busy         []int
	sending      []int
	pieces       []piece
	spans        [][2]int
	tileRows     [][]byte
	spliced      []uint8
	splices      map[[2]int32]int32
	tiles        []image.Rectangle
	tileOf       []int
	hashes, sent []uint64
	dirty, send  []bool
	moved, plain []bool
	images       map[uint64]uint32
	uses         []int
	samples      []color.Color
	sampled      []bool
	cache        map[uint64]*cached
	groups       []group
	parts        []part
	seed         maphash.Seed
	frame        uint64
	rastered     int
	imageBytes   int
	kitty        graphics.Kitty
}

type cached struct {
	pix     []uint8
	stride  int
	row     []int32
	solid   [][2]int
	uniform [2]int
	frame   uint64
}

type group struct {
	layer   int
	into    int
	opacity float64
	own     bool
}

type step uint8

const (
	drawBox step = iota
	openGroup
	closeGroup
)

type part struct {
	step    step
	c       *cached
	r       image.Rectangle
	at      image.Point
	into    int
	opacity float64
}

func (s *Screen) Frame(root scene.Node, cols, rows int) error {
	s.out.Reset()
	if s.Sync {
		s.out.WriteString(termkonst.SyncBegin)
	}
	start := s.out.Len()
	if cols != s.cols || rows != s.rows || s.Cell != s.cell || s.text == nil {
		s.reset(cols, rows)
	}
	look := paint.Glyphs
	switch {
	case s.Graphics == terminal.GraphicsNone && s.Profile <= color.Attributes:
		look = paint.Plain
	case s.Graphics == terminal.GraphicsNone:
		look = paint.Composited
	}
	s.painter.Paint(s.text, &root, look)
	s.imageBytes = 0
	if s.Graphics != terminal.GraphicsNone {
		s.surfaces(&root)
	}
	s.compose()
	if s.Graphics == terminal.GraphicsNone {
		s.scrollbars(&root)
	}
	if err := s.writer.Diff(s.shown, s.want); err != nil {
		return err
	}
	s.shown, s.want = s.want, s.shown
	s.fresh = false
	if s.out.Len() == start {
		return nil
	}
	if s.Sync {
		s.out.WriteString(termkonst.SyncEnd)
	}
	_, err := s.Out.Write(s.out.Bytes())
	return err
}

func (s *Screen) reset(cols, rows int) {
	for slot, uses := range s.uses {
		if uses > 0 {
			s.out.Write(graphics.KittyDelete(s.out.AvailableBuffer(), konst.KittyFirstImage+uint32(slot)))
		}
	}
	if s.text != nil && s.underText() {
		s.out.WriteString(termkonst.Reset + termkonst.CSI + "2J")
	}
	if s.Cell != s.cell || s.cache == nil {
		s.cache, s.splices, s.seed = map[uint64]*cached{}, map[[2]int32]int32{}, maphash.MakeSeed()
	}
	s.cols, s.rows, s.cell, s.fresh = cols, rows, s.Cell, true
	s.text, s.shown, s.want = buffer.New(cols, rows), buffer.New(cols, rows), buffer.New(cols, rows)
	s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	unknown := buffer.Cell{Grapheme: "\x00"}
	if s.underText() {
		unknown = buffer.Cell{}
	}
	s.shown.Fill(buffer.Rect{W: cols, H: rows}, unknown)
	if s.Graphics == terminal.GraphicsNone {
		return
	}
	s.bounds = image.Rect(0, 0, cols*s.Cell.X, rows*s.Cell.Y)
	band := 1
	for s.Graphics == terminal.GraphicsSixel && band*s.Cell.Y%graphicskonst.SixelBand != 0 {
		band++
	}
	span := konst.TileColumns * s.Cell.X
	s.columns = slices.Grow(s.columns[:0], (s.bounds.Dx()+span-1)/span)[:(s.bounds.Dx()+span-1)/span]
	for i := range s.columns {
		c := &s.columns[i]
		if width := 4 * (min(i*span+span, s.bounds.Max.X) - i*span); width != c.width {
			*c = column{width: width}
		}
		c.free = c.free[:0]
		clear(c.holders)
		for k := range c.store {
			c.free = append(c.free, int32(k))
		}
		c.lineOf = slices.Grow(c.lineOf[:0], s.bounds.Dy())[:s.bounds.Dy()]
		for y := range c.lineOf {
			c.lineOf[y] = -1
		}
	}
	s.tiles, s.tileOf = s.tiles[:0], make([]int, cols*rows)
	for bottom := rows; bottom > 0; bottom -= band {
		for left := 0; left < cols; left += konst.TileColumns {
			t := image.Rect(left, max(bottom-band, 0), min(left+konst.TileColumns, cols), bottom)
			for y := t.Min.Y; y < t.Max.Y; y++ {
				for x := t.Min.X; x < t.Max.X; x++ {
					s.tileOf[y*cols+x] = len(s.tiles)
				}
			}
			s.tiles = append(s.tiles, t)
		}
	}
	n := len(s.tiles)
	s.hashes, s.sent, s.dirty, s.send, s.moved, s.plain = make([]uint64, n), make([]uint64, n), make([]bool, n), make([]bool, n), make([]bool, n), make([]bool, n)
	s.spans, s.pieces = make([][2]int, n), make([]piece, n)
	s.samples, s.sampled = make([]color.Color, cols*rows), make([]bool, cols*rows)
	if s.Graphics == terminal.GraphicsKitty {
		s.images, s.uses = map[uint64]uint32{}, make([]int, n)
	}
}

func (s *Screen) unplace(t int) {
	hash, slot := s.sent[t], s.images[s.sent[t]]
	s.sent[t] = 0
	if s.uses[slot]--; s.uses[slot] > 0 {
		s.out.Write(graphics.KittyUnplace(s.out.AvailableBuffer(), konst.KittyFirstImage+slot, uint32(t)+1))
		return
	}
	delete(s.images, hash)
	s.out.Write(graphics.KittyDelete(s.out.AvailableBuffer(), konst.KittyFirstImage+slot))
}

func (s *Screen) underText() bool {
	return s.Graphics == terminal.GraphicsSixel || s.Graphics == terminal.GraphicsITerm2
}

func (s *Screen) compose() {
	underImage := buffer.Cell{}
	for y := range s.rows {
		text, shown, want := s.text.Row(y), s.shown.Row(y), s.want.Row(y)
		for x, c := range text {
			switch {
			case s.Graphics == terminal.GraphicsNone:
			case s.Graphics == terminal.GraphicsKitty && s.plain[s.tileOf[y*s.cols+x]]:
				c.Bg = s.sample(x, y)
			case s.Graphics == terminal.GraphicsKitty:
				c.Bg = color.Color{}
			case blank(c) && shown[x] == underImage:
				c = underImage
			case blank(c):
				c = buffer.Cell{Grapheme: " ", Bg: s.sample(x, y)}
			default:
				c.Bg = s.sample(x, y)
			}
			want[x] = c
		}
	}
}

func blank(c buffer.Cell) bool {
	return c.Grapheme == " " && c.Attr&(buffer.Underline|buffer.Strikethrough|buffer.Inverse) == 0
}

func (s *Screen) surfaces(root *scene.Node) {
	prev, next := &s.scenes[s.turn], &s.scenes[1-s.turn]
	s.turn = 1 - s.turn
	next.Record(root, s.Cell)
	clear(s.dirty)
	clear(s.send)
	changed := s.fresh
	if s.fresh {
		for t := range s.dirty {
			s.dirty[t] = true
		}
	} else {
		d := scene.Diff(prev, next)
		for _, r := range d.Rects {
			s.damage(r)
		}
		for _, m := range d.Moves {
			l := &next.Layers[m.Layer]
			s.damage(l.Visual)
			s.damage(l.Visual.Sub(m.To).Add(m.From).Intersect(l.Clip))
		}
		for _, sc := range d.Scrolls {
			s.scroll(next, sc)
		}
		changed = len(d.Rects)+len(d.Moves)+len(d.Scrolls) > 0
	}
	if changed {
		s.frame++
	}
	s.parts, s.busy = s.parts[:0], s.busy[:0]
	for t, dirty := range s.dirty {
		if !dirty {
			continue
		}
		s.collect(next, t)
		if column := t % len(s.columns); !slices.Contains(s.busy, column) {
			s.busy = append(s.busy, column)
		}
	}
	clear(s.pieces)
	for _, w := range s.workers {
		w.out = w.out[:0]
	}
	s.rasterise()
	s.parallel(len(s.busy), func(w *worker, i int) {
		for t := s.busy[i]; t < len(s.tiles); t += len(s.columns) {
			if !s.dirty[t] {
				continue
			}
			if s.moved[t] {
				s.sent[t], s.moved[t] = s.hash(t), false
			}
			w.fill(s, t)
			s.hashes[t] = s.hash(t)
			s.plain[t] = s.Graphics != terminal.GraphicsSixel && s.Profile == color.TrueColor && s.plainTile(t)
			if s.Graphics != terminal.GraphicsKitty && !s.plain[t] && s.hashes[t] != s.sent[t] {
				w.encode(s, t)
			}
		}
	})
	for t, dirty := range s.dirty {
		if !dirty {
			continue
		}
		cells := s.tiles[t]
		switch {
		case !s.plain[t]:
			s.send[t] = s.hashes[t] != s.sent[t]
		case s.Graphics == terminal.GraphicsKitty && s.sent[t] != 0:
			s.unplace(t)
		case s.Graphics == terminal.GraphicsITerm2 && (s.sent[t] != 0 || s.fresh):
			s.shown.Fill(buffer.Rect{X: cells.Min.X, Y: cells.Min.Y, W: cells.Dx(), H: cells.Dy()}, buffer.Cell{Grapheme: "\x00"})
			s.sent[t] = 0
		}
		for y := cells.Min.Y; y < cells.Max.Y; y++ {
			clear(s.sampled[y*s.cols+cells.Min.X : y*s.cols+cells.Max.X])
		}
	}
	if changed {
		s.evict(next)
	}
	if s.underText() {
		underImage := buffer.Cell{}
		for y := range s.rows {
			shown, text := s.shown.Row(y), s.text.Row(y)
			for x := range shown {
				if t := s.tileOf[y*s.cols+x]; shown[x] != underImage && blank(text[x]) && !s.send[t] && !s.plain[t] && !s.flat(x, y) {
					s.send[t] = true
				}
			}
		}
	}
	s.sending = s.sending[:0]
	for t, send := range s.send {
		if send && s.Graphics != terminal.GraphicsKitty && s.pieces[t].w == nil {
			s.sending = append(s.sending, t)
		}
	}
	s.parallel(len(s.sending), func(w *worker, i int) { w.encode(s, s.sending[i]) })
	sent := false
	for t, send := range s.send {
		if send {
			s.put(t)
			sent = true
		}
	}
	if sent {
		s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	}
}

func (s *Screen) put(t int) {
	cells := s.tiles[t]
	start := s.out.Len()
	if s.underText() && s.sent[t] != 0 && !s.opaque(t) {
		s.out.WriteString(termkonst.Reset)
		for y := cells.Min.Y; y < cells.Max.Y; y++ {
			fmt.Fprintf(&s.out, "%s%d;%dH%s%d%s", termkonst.CSI, y+1, cells.Min.X+1, termkonst.CSI, cells.Dx(), konst.EraseCells)
		}
	}
	if s.Graphics == terminal.GraphicsKitty && s.sent[t] != 0 {
		s.unplace(t)
	}
	switch s.Graphics {
	case terminal.GraphicsSixel, terminal.GraphicsITerm2:
		p := s.pieces[t]
		s.out.Write(p.w.out[p.lo:p.hi])
	case terminal.GraphicsKitty:
		dst := s.out.AvailableBuffer()
		slot, placed := s.images[s.hashes[t]]
		if placed {
			dst = graphics.KittyPlace(dst, s.placement(t), konst.KittyFirstImage+slot, uint32(t)+1)
		} else {
			slot = uint32(slices.Index(s.uses, 0))
			s.images[s.hashes[t]] = slot
			s.tileRows = s.tileLines(t, s.tileRows[:0])
			dst = s.kitty.Encode(dst, s.tileRows, s.placement(t), konst.KittyFirstImage+slot, uint32(t)+1)
		}
		s.uses[slot]++
		s.out.Write(dst)
	case terminal.GraphicsNone:
		panic("present: a tile put without graphics")
	default:
		panic(fmt.Sprintf("present: unknown graphics %d", s.Graphics))
	}
	s.imageBytes += s.out.Len() - start
	s.sent[t] = s.hashes[t]
	if s.underText() {
		for y := cells.Min.Y; y < cells.Max.Y; y++ {
			clear(s.shown.Row(y)[cells.Min.X:cells.Max.X])
		}
	}
}
