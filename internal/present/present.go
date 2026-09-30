package present

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"image"
	"io"
	"math"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/raster"
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
	surface      *image.RGBA
	tiles        []image.Rectangle
	tileOf       []int
	hashes, sent []uint64
	dirty, send  []bool
	moved        []bool
	samples      []color.Color
	sampled      []bool
	cache        map[uint64]*cached
	groups       []group
	scratch      []*image.RGBA
	ops          []raster.Op
	raster       raster.Raster
	seed         maphash.Seed
	frame        uint64
	rastered     int
	imageBytes   int
	sixel        graphics.Sixel
	kitty        graphics.Kitty
	iterm        graphics.ITerm
}

type cached struct {
	img     *image.RGBA
	frame   uint64
	solid   [][2]int
	uniform [2]int
}

type group struct {
	layer   int
	img     *image.RGBA
	opacity float64
	own     bool
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
		if err := s.surfaces(&root); err != nil {
			return err
		}
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
	if s.Graphics == terminal.GraphicsKitty {
		for t, h := range s.sent {
			if h != 0 {
				s.out.Write(graphics.KittyDelete(s.out.AvailableBuffer(), konst.KittyFirstImage+uint32(t)))
			}
		}
	}
	if s.text != nil && s.underText() {
		s.out.WriteString(termkonst.Reset + termkonst.CSI + "2J")
	}
	if s.Cell != s.cell || s.cache == nil {
		s.cache, s.seed = map[uint64]*cached{}, maphash.MakeSeed()
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
	s.surface = image.NewRGBA(image.Rect(0, 0, cols*s.Cell.X, rows*s.Cell.Y))
	band := 1
	for s.Graphics == terminal.GraphicsSixel && band*s.Cell.Y%graphicskonst.SixelBand != 0 {
		band++
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
	s.hashes, s.sent, s.dirty, s.send, s.moved = make([]uint64, n), make([]uint64, n), make([]bool, n), make([]bool, n), make([]bool, n)
	s.samples, s.sampled = make([]color.Color, cols*rows), make([]bool, cols*rows)
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

func (s *Screen) surfaces(root *scene.Node) error {
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
	for t, dirty := range s.dirty {
		if !dirty {
			continue
		}
		cells := s.tiles[t]
		if s.moved[t] {
			s.sent[t], s.moved[t] = s.hash(s.pixels(cells)), false
		}
		s.composite(next, s.pixels(cells))
		if s.Profile == color.ANSI256 {
			s.quantise(s.pixels(cells))
		}
		s.hashes[t] = s.hash(s.pixels(cells))
		if s.send[t] = s.hashes[t] != s.sent[t]; !s.send[t] {
			continue
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
				if t := s.tileOf[y*s.cols+x]; shown[x] != underImage && blank(text[x]) && !s.send[t] && !s.flat(x, y) {
					s.send[t] = true
				}
			}
		}
	}
	sent := false
	for t, send := range s.send {
		if !send {
			continue
		}
		if err := s.put(t); err != nil {
			return err
		}
		sent = true
	}
	if sent {
		s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	}
	return nil
}

func (s *Screen) put(t int) error {
	cells := s.tiles[t]
	img := s.surface.SubImage(s.pixels(cells)).(*image.RGBA)
	at := graphics.Placement{Col: cells.Min.X, Row: cells.Min.Y, Cols: cells.Dx(), Rows: cells.Dy()}
	start := s.out.Len()
	if s.underText() && s.sent[t] != 0 && !opaque(img) {
		s.out.WriteString(termkonst.Reset)
		for y := cells.Min.Y; y < cells.Max.Y; y++ {
			fmt.Fprintf(&s.out, "%s%d;%dH%s%d%s", termkonst.CSI, y+1, cells.Min.X+1, termkonst.CSI, cells.Dx(), konst.EraseCells)
		}
	}
	dst := s.out.AvailableBuffer()
	switch s.Graphics {
	case terminal.GraphicsSixel:
		dst = s.sixel.Encode(dst, img, at)
	case terminal.GraphicsKitty:
		dst = s.kitty.Encode(dst, img, at, konst.KittyFirstImage+uint32(t), 1)
	case terminal.GraphicsITerm2:
		var err error
		if dst, err = s.iterm.Encode(dst, img, at); err != nil {
			return err
		}
	case terminal.GraphicsNone:
		panic("present: a tile put without graphics")
	default:
		panic(fmt.Sprintf("present: unknown graphics %d", s.Graphics))
	}
	s.out.Write(dst)
	s.imageBytes += s.out.Len() - start
	s.sent[t] = s.hashes[t]
	if s.underText() {
		for y := cells.Min.Y; y < cells.Max.Y; y++ {
			clear(s.shown.Row(y)[cells.Min.X:cells.Max.X])
		}
	}
	return nil
}

func opaque(img *image.RGBA) bool {
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		row := img.Pix[img.PixOffset(img.Rect.Min.X, y):img.PixOffset(img.Rect.Max.X, y)]
		for i := 3; i < len(row); i += 4 {
			if row[i] != math.MaxUint8 {
				return false
			}
		}
	}
	return true
}
