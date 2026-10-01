package present

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"image"
	"io"
	"math"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	konst "github.com/twind-dev/twind/internal/konst/paint"
	presentkonst "github.com/twind-dev/twind/internal/konst/present"
	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/graphics"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

type Screen struct {
	Out      io.Writer
	Profile  color.Profile
	Graphics terminal.Graphics
	Cell     image.Point
	Sync     bool
	Margins  bool
	Widths   text.Widths
	Workers  int

	cols, rows        int
	cell              image.Point
	painter           paint.Painter
	fresh             bool
	text, shown, want *buffer.Buffer
	out               bytes.Buffer
	writer            terminal.Writer

	turn         int
	scenes       [2]*scene.Frame
	bounds       image.Rectangle
	columns      []column
	workers      []*worker
	pending      []pending
	jobs         []job
	order        []int
	jobKeys      []byte
	seen         []seen
	lookOps      []raster.Op
	turned       []raster.Op
	lookRows     []bool
	across       []bool
	lookPieces   [][2]int
	lookPix      []uint8
	lookAt       [][4]int
	sums         [][4]int
	bands        []band
	sampling     []run
	sending      []int
	pieces       []piece
	twins        []int
	bases        [][]part
	based        []bool
	claims       map[twin]int
	lock         sync.Mutex
	recipes      []recipe
	arena        []byte
	painted      atomic.Bool
	drawing      []int
	hidden       []bool
	tileRows     [][]byte
	lineRuns     [][]run
	pix          []uint8
	key          []byte
	spliced      []run
	blank        []run
	splices      map[[2]int32]int32
	tiles        []image.Rectangle
	band         int
	hashes, sent []uint64
	dirty, send  []bool
	moved, plain []bool
	needs        []need
	images       map[uint64]uint32
	uses         []int
	samples      []color.Color
	sampled      []bool
	cache        map[uint64]*cached
	shapes       map[string]*cached
	shape        []byte
	idle         []bool
	seed         maphash.Seed
	frame        uint64
	rastered     int
	imageBytes   int
	kitty        *graphics.Kitty
}

type cached struct {
	ready   sync.WaitGroup
	row     []int32
	starts  []int32
	lines   [][]run
	change  [][2]int32
	uniform [2]int
	frame   uint64
	id      int
}

type run = graphics.Run

type twin struct {
	hash uint64
	size image.Point
}

type group struct {
	layer   int
	into    int
	opacity float64
	own     bool
}

type need uint8

const (
	keep need = iota
	erase
	either
)

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
	s.painter.Widths, s.painter.Profile, s.imageBytes = s.Widths, s.Profile, 0
	s.painted.Store(false)
	paint := func() {
		if s.text == nil {
			s.text = buffer.New(s.cols, s.rows)
		}
		s.painter.Paint(s.text, &root, look)
		s.painted.Store(true)
		s.screens()
	}
	if s.Graphics == terminal.GraphicsNone {
		paint()
	} else {
		next, changed := s.damaged(&root)
		s.surfaces(next, changed, paint)
		s.transmit()
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
		s.cache, s.shapes, s.splices, s.seed = map[uint64]*cached{}, map[string]*cached{}, map[[2]int32]int32{}, maphash.MakeSeed()
	}
	s.cols, s.rows, s.cell, s.fresh = cols, rows, s.Cell, true
	s.text, s.shown, s.want = nil, nil, nil
	s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
	if s.Graphics == terminal.GraphicsNone {
		return
	}
	s.bounds = image.Rect(0, 0, cols*s.Cell.X, rows*s.Cell.Y)
	s.band = 1
	for s.Graphics == terminal.GraphicsSixel && s.band*s.Cell.Y%graphicskonst.SixelBand != 0 {
		s.band++
	}
	span := konst.TileColumns * s.Cell.X
	s.columns = slices.Grow(s.columns[:0], (s.bounds.Dx()+span-1)/span)[:(s.bounds.Dx()+span-1)/span]
	for i := range s.columns {
		c := &s.columns[i]
		if width := min(i*span+span, s.bounds.Max.X) - i*span; width != c.width {
			*c = column{width: width}
		}
		c.free = c.free[:0]
		clear(c.holders)
		for k := range c.store {
			c.free = append(c.free, int32(k))
		}
		c.lineOf, c.baseOf = slices.Grow(c.lineOf[:0], s.bounds.Dy())[:s.bounds.Dy()], c.baseOf[:0]
		for y := range c.lineOf {
			c.lineOf[y] = -1
		}
	}
	s.tiles = slices.Grow(s.tiles[:0], len(s.columns)*((rows+s.band-1)/s.band))
	for bottom := rows; bottom > 0; bottom -= s.band {
		for left := 0; left < cols; left += konst.TileColumns {
			s.tiles = append(s.tiles, image.Rect(left, max(bottom-s.band, 0), min(left+konst.TileColumns, cols), bottom))
		}
	}
	n := len(s.tiles)
	s.hashes, s.sent, s.dirty, s.send, s.moved, s.plain = make([]uint64, n), make([]uint64, n), make([]bool, n), make([]bool, n), make([]bool, n), make([]bool, n)
	s.pieces, s.twins, s.claims, s.bases, s.based = make([]piece, n), make([]int, n), map[twin]int{}, make([][]part, n), make([]bool, n)
	s.samples, s.sampled, s.needs = make([]color.Color, cols*rows), make([]bool, cols*rows), make([]need, cols)
	if s.Graphics == terminal.GraphicsKitty {
		s.images, s.uses = map[uint64]uint32{}, make([]int, n)
		if s.kitty == nil {
			s.kitty = &graphics.Kitty{}
		}
	}
}

func (s *Screen) tileAt(x, y int) int {
	return (s.rows-1-y)/s.band*len(s.columns) + x/konst.TileColumns
}

func (s *Screen) screens() {
	if s.shown != nil {
		return
	}
	unknown := buffer.Cell{Grapheme: "\x00"}
	if s.underText() {
		unknown = buffer.Cell{}
	}
	s.shown, s.want = buffer.Filled(s.cols, s.rows, unknown), buffer.Filled(s.cols, s.rows, buffer.Cell{})
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
	for y := range s.rows {
		text, shown, want := s.text.Row(y), s.shown.Row(y), s.want.Row(y)
		copy(want, text)
		for x := range want {
			c := &want[x]
			switch {
			case s.Graphics == terminal.GraphicsNone:
			case s.Graphics == terminal.GraphicsKitty && s.plain[s.tileAt(x, y)]:
				c.Bg = s.sample(x, y)
			case s.Graphics == terminal.GraphicsKitty:
				c.Bg = color.Color{}
			case blank(*c) && shown[x].Grapheme == "":
				c.Grapheme, c.Fg, c.Bg, c.Attr, c.Width = "", shown[x].Fg, shown[x].Bg, shown[x].Attr, shown[x].Width
			case blank(*c):
				c.Fg, c.Bg, c.Attr, c.Width = color.Color{}, s.sample(x, y), 0, buffer.Narrow
			default:
				c.Bg = s.sample(x, y)
			}
		}
	}
}

func blank(c buffer.Cell) bool {
	return c.Grapheme == " " && c.Attr&(buffer.Underline|buffer.Strikethrough|buffer.Inverse) == 0
}

func (s *Screen) damaged(root *scene.Node) (*scene.Frame, bool) {
	if s.scenes[1-s.turn] == nil {
		s.scenes[1-s.turn] = &scene.Frame{}
	}
	prev, next := s.scenes[s.turn], s.scenes[1-s.turn]
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
	s.drawing = s.drawing[:0]
	for t, dirty := range s.dirty {
		if dirty {
			s.drawing = append(s.drawing, t)
		}
	}
	s.gather(next)
	return next, changed
}

func (s *Screen) surfaces(next *scene.Frame, changed bool, paint func()) {
	clear(s.pieces)
	clear(s.claims)
	for t := range s.twins {
		s.twins[t] = t
	}
	for _, w := range s.workers {
		w.out = w.out[:0]
	}
	for _, t := range s.drawing {
		if s.moved[t] {
			s.sent[t], s.moved[t] = s.hash(t), false
		}
	}
	s.recipes, s.arena = s.recipes[:0], slices.Grow(s.arena[:0], presentkonst.RecipeBytes*len(s.drawing))
	s.rasterise(len(s.drawing), paint, func(w *worker, i int) {
		t := s.drawing[i]
		twin, memo := w.fill(s, next, t)
		height := s.tiles[t].Dy() * s.Cell.Y
		if twin >= 0 {
			s.plain[t] = s.plain[twin]
		} else {
			s.plain[t] = s.Graphics != terminal.GraphicsSixel && s.Profile == color.TrueColor && s.plainTile(w.rowLines(height))
		}
		if s.Graphics != terminal.GraphicsKitty && !s.plain[t] && s.hashes[t] != s.sent[t] {
			s.claim(t)
		}
		w.measure(s, t, twin)
		if memo {
			w.remember(s, t)
		}
	})
	if changed {
		s.evict(next)
	}
}

func (s *Screen) claim(t int) {
	key := twin{s.hashes[t], s.tiles[t].Size()}
	s.lock.Lock()
	defer s.lock.Unlock()
	first, taken := s.claims[key]
	if !taken {
		s.claims[key], first = t, t
	}
	s.twins[t] = first
}

func (s *Screen) transmit() {
	for t, dirty := range s.dirty {
		cells := s.tiles[t]
		switch {
		case !dirty:
		case !s.plain[t]:
			s.send[t] = s.hashes[t] != s.sent[t]
		case s.Graphics == terminal.GraphicsKitty && s.sent[t] != 0:
			s.unplace(t)
		case s.Graphics == terminal.GraphicsITerm2 && (s.sent[t] != 0 || s.fresh):
			s.shown.Fill(buffer.Rect{X: cells.Min.X, Y: cells.Min.Y, W: cells.Dx(), H: cells.Dy()}, buffer.Cell{Grapheme: "\x00"})
			s.sent[t] = 0
		}
	}
	if s.underText() && !s.fresh {
		for y := range s.rows {
			shown, text := s.shown.Row(y), s.text.Row(y)
			for x := range shown {
				if t := s.tileAt(x, y); shown[x].Grapheme != "" && blank(text[x]) && !s.send[t] && !s.plain[t] && !s.flat(x, y) {
					s.send[t] = true
				}
			}
		}
	}
	s.sending = s.sending[:0]
	for t, send := range s.send {
		if send && s.Graphics != terminal.GraphicsKitty && s.twins[t] == t && s.pieces[t].w == nil {
			s.sending = append(s.sending, t)
		}
	}
	s.parallel(len(s.sending), 0, len(s.sending), nil, func(w *worker, i int) {
		w.lines = s.tileLines(s.sending[i], w.lines[:0])
		w.encode(s, s.sending[i])
	})
	size := 0
	for t, send := range s.send {
		if send {
			size += s.pieces[s.twins[t]].hi - s.pieces[s.twins[t]].lo
		}
	}
	s.out.Grow(size + s.cols*s.rows*graphicskonst.CellBytes)
	if !slices.Contains(s.send, true) {
		return
	}
	if s.underText() {
		s.ground()
	}
	for t, send := range s.send {
		if send {
			s.put(t)
		}
	}
	s.writer = terminal.Writer{Out: &s.out, Profile: s.Profile}
}

func (s *Screen) put(t int) {
	cells := s.tiles[t]
	start := s.out.Len()
	if s.Graphics == terminal.GraphicsKitty && s.sent[t] != 0 {
		s.unplace(t)
	}
	switch s.Graphics {
	case terminal.GraphicsSixel, terminal.GraphicsITerm2:
		p := s.pieces[s.twins[t]]
		encoded := p.w.out[p.lo:p.hi]
		if s.twins[t] != t && len(encoded) > 0 {
			at := strconv.AppendInt(append(s.out.AvailableBuffer(), termkonst.CSI...), int64(cells.Min.Y+1), 10)
			s.out.Write(append(strconv.AppendInt(append(at, ';'), int64(cells.Min.X+1), 10), 'H'))
			encoded = encoded[bytes.IndexByte(encoded, 'H')+1:]
		}
		s.out.Write(encoded)
	case terminal.GraphicsKitty:
		dst := s.out.AvailableBuffer()
		slot, placed := s.images[s.hashes[t]]
		if placed {
			dst = graphics.KittyPlace(dst, s.placement(t), konst.KittyFirstImage+slot, uint32(t)+1)
		} else {
			slot = uint32(slices.Index(s.uses, 0))
			s.images[s.hashes[t]] = slot
			s.pix, s.tileRows = expand(s.pix, s.tileRows, s.tileLines(t, s.lineRuns[:0]))
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
}

func (s *Screen) ground() {
	eraser, dst := terminal.Writer{Profile: s.Profile}, s.out.AvailableBuffer()
	for y := range s.rows {
		shown, text := s.shown.Row(y), s.text.Row(y)
		band := s.tileAt(0, y)
		for x := range shown {
			t := band + x/konst.TileColumns
			if s.needs[x] = keep; !s.send[t] {
				continue
			}
			c, ground := &shown[x], s.samples[y*s.cols+x]
			if !s.sampled[y*s.cols+x] {
				ground = s.sample(x, y)
			}
			if ground.RGBA.A != math.MaxUint8 {
				ground = color.Color{}
			}
			switch {
			case !blank(text[x]):
				s.needs[x] = either
			case *c != buffer.Cell{Bg: ground} || ground.RGBA.A == 0 && s.sent[t] != 0:
				s.needs[x] = erase
			}
			if c.Grapheme != "" {
				c.Grapheme = ""
			}
			c.Fg, c.Bg, c.Attr, c.Width = color.Color{}, ground, 0, 0
			if x > 0 && x == s.tiles[t].Min.X && text[x].Width == buffer.Continuation {
				shown[x-1].Grapheme, shown[x-1].Fg, shown[x-1].Bg, shown[x-1].Attr, shown[x-1].Width = "\x00", color.Color{}, color.Color{}, 0, 0
			}
		}
		for x := 0; x < len(shown); x++ {
			if s.needs[x] != erase {
				continue
			}
			n := 1
			for i := x + 1; i < len(shown) && s.needs[i] != keep && shown[i].Bg == shown[x].Bg && shown[i].Grapheme == shown[x].Grapheme; i++ {
				if s.needs[i] == erase {
					n = i - x + 1
				}
			}
			dst = eraser.Erase(dst, x, y, n, shown[x].Bg)
			x += n - 1
		}
	}
	s.imageBytes += len(dst)
	s.out.Write(dst)
}
