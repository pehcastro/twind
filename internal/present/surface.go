package present

import (
	"bytes"
	"hash/maphash"
	"image"
	imagecolor "image/color"
	"image/draw"
	"math"

	colorkonst "github.com/twind-dev/twind/internal/konst/color"
	graphicskonst "github.com/twind-dev/twind/internal/konst/graphics"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/terminal"
)

func (s *Screen) composite(f *scene.Frame, t image.Rectangle) {
	draw.Draw(s.surface, t, image.Transparent, image.Point{}, draw.Src)
	s.groups = s.groups[:0]
	for i := range f.Layers {
		l := &f.Layers[i]
		for len(s.groups) > 0 && s.groups[len(s.groups)-1].layer != l.Parent {
			s.pop(t)
		}
		target := s.surface
		if n := len(s.groups); n > 0 {
			target = s.groups[n-1].img
		}
		own := target != nil && l.Opacity > 0 && l.Opacity < 1
		switch {
		case own:
			target = s.group(len(s.groups), t)
		case l.Opacity <= 0:
			target = nil
		}
		s.groups = append(s.groups, group{layer: i, img: target, opacity: l.Opacity, own: own})
		if target == nil || !l.Visual.Overlaps(t) {
			continue
		}
		for j := range l.Boxes {
			b := &l.Boxes[j]
			at := b.Visual.Add(l.Origin)
			if r := at.Intersect(t); !r.Empty() {
				over(target, r, s.boxRaster(b), r.Min.Sub(at.Min))
			}
		}
	}
	for len(s.groups) > 0 {
		s.pop(t)
	}
}

func over(dst *image.RGBA, r image.Rectangle, src *image.RGBA, at image.Point) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		d := dst.Pix[dst.PixOffset(r.Min.X, y):dst.PixOffset(r.Max.X, y)]
		p := src.Pix[src.PixOffset(at.X, at.Y+y-r.Min.Y):][:len(d)]
		for i := 0; i < len(d); i += 4 {
			switch a := uint32(p[i+3]); a {
			case 0:
			case math.MaxUint8:
				end := i + 4
				for end < len(d) && p[end+3] == math.MaxUint8 {
					end += 4
				}
				copy(d[i:end], p[i:end])
				i = end - 4
			default:
				keep := math.MaxUint8 - a
				for c := range 4 {
					d[i+c] = p[i+c] + uint8((uint32(d[i+c])*keep+math.MaxUint8/2)/math.MaxUint8)
				}
			}
		}
	}
}

func (s *Screen) pop(t image.Rectangle) {
	g := s.groups[len(s.groups)-1]
	s.groups = s.groups[:len(s.groups)-1]
	if !g.own {
		return
	}
	dst := s.surface
	if n := len(s.groups); n > 0 {
		dst = s.groups[n-1].img
	}
	alpha := image.NewUniform(imagecolor.Alpha{A: uint8(math.Round(g.opacity * math.MaxUint8))})
	draw.DrawMask(dst, t, g.img, t.Min, alpha, image.Point{}, draw.Over)
}

func (s *Screen) group(depth int, t image.Rectangle) *image.RGBA {
	for len(s.scratch) <= depth {
		s.scratch = append(s.scratch, &image.RGBA{})
	}
	g := s.scratch[depth]
	n := 4 * t.Dx() * t.Dy()
	if cap(g.Pix) < n {
		g.Pix = make([]uint8, n)
	}
	g.Pix, g.Stride, g.Rect = g.Pix[:n], 4*t.Dx(), t
	clear(g.Pix)
	return g
}

func (s *Screen) boxRaster(b *scene.Box) *image.RGBA {
	if c, ok := s.cache[b.Look]; ok {
		return c.img
	}
	img := image.NewRGBA(image.Rectangle{Max: b.Visual.Size()})
	s.ops = append(s.ops[:0], b.Ops...)
	for i := range s.ops {
		s.ops[i].Box.X -= float64(b.Visual.Min.X)
		s.ops[i].Box.Y -= float64(b.Visual.Min.Y)
	}
	s.raster.Draw(img, s.ops, img.Rect)
	s.rastered++
	s.cache[b.Look] = &cached{img: img, frame: s.frame}
	return img
}

func (s *Screen) evict(f *scene.Frame) {
	for _, l := range f.Layers {
		for _, b := range l.Boxes {
			if c, ok := s.cache[b.Look]; ok {
				c.frame = s.frame
			}
		}
	}
	for look, c := range s.cache {
		if c.frame != s.frame {
			delete(s.cache, look)
		}
	}
}

func (s *Screen) damage(r image.Rectangle) {
	r = r.Intersect(s.surface.Rect)
	cells := image.Rect(r.Min.X/s.Cell.X, r.Min.Y/s.Cell.Y, (r.Max.X+s.Cell.X-1)/s.Cell.X, (r.Max.Y+s.Cell.Y-1)/s.Cell.Y)
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		for x := cells.Min.X; x < cells.Max.X; x++ {
			s.dirty[s.tileOf[y*s.cols+x]] = true
		}
	}
}

func (s *Screen) pixels(cells image.Rectangle) image.Rectangle {
	return image.Rect(cells.Min.X*s.Cell.X, cells.Min.Y*s.Cell.Y, cells.Max.X*s.Cell.X, cells.Max.Y*s.Cell.Y)
}

func (s *Screen) hash(r image.Rectangle) uint64 {
	var h maphash.Hash
	h.SetSeed(s.seed)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		_, _ = h.Write(s.surface.Pix[s.surface.PixOffset(r.Min.X, y):s.surface.PixOffset(r.Max.X, y)])
	}
	return h.Sum64()
}

func (s *Screen) sample(x, y int) color.Color {
	i := y*s.cols + x
	if s.sampled[i] {
		return s.samples[i]
	}
	s.sampled[i], s.samples[i] = true, color.Color{}
	if m := raster.Mean(s.surface, s.pixels(image.Rect(x, y, x+1, y+1))); m.A > 0 {
		if s.Graphics == terminal.GraphicsSixel {
			m.R, m.G, m.B = register(m.R), register(m.G), register(m.B)
		}
		m.A = math.MaxUint8
		s.samples[i] = color.Color{Kind: color.Literal, RGBA: m}
	}
	return s.samples[i]
}

func (s *Screen) quantise(r image.Rectangle) {
	var from, to [3]uint8
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := s.surface.Pix[s.surface.PixOffset(r.Min.X, y):s.surface.PixOffset(r.Max.X, y)]
		for i := 0; i < len(row); i += 4 {
			if row[i+3] != math.MaxUint8 {
				continue
			}
			if rgb := [3]uint8(row[i : i+3]); rgb != from {
				from, to = rgb, palette(color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2]}.ANSI256())
			}
			copy(row[i:i+3], to[:])
		}
	}
}

func palette(index uint8) [3]uint8 {
	if index >= colorkonst.GreyBase {
		v := uint8(colorkonst.GreyFirstLevel + int(index-colorkonst.GreyBase)*colorkonst.GreyLevelStep)
		return [3]uint8{v, v, v}
	}
	cube := int(index - colorkonst.CubeBase)
	var rgb [3]uint8
	for c := 2; c >= 0; c-- {
		if n := cube % colorkonst.CubeSide; n > 0 {
			rgb[c] = uint8(colorkonst.CubeFirstLevel + (n-1)*colorkonst.CubeLevelStep)
		}
		cube /= colorkonst.CubeSide
	}
	return rgb
}

func register(v uint8) uint8 {
	percent := (int(v)*graphicskonst.SixelPercent + math.MaxUint8/2) / math.MaxUint8
	return uint8((percent*math.MaxUint8 + graphicskonst.SixelPercent/2) / graphicskonst.SixelPercent)
}

func (s *Screen) flat(x, y int) bool {
	r := s.pixels(image.Rect(x, y, x+1, y+1))
	first := s.surface.Pix[s.surface.PixOffset(r.Min.X, r.Min.Y):][:4]
	for py := r.Min.Y; py < r.Max.Y; py++ {
		row := s.surface.Pix[s.surface.PixOffset(r.Min.X, py):s.surface.PixOffset(r.Max.X, py)]
		for i := 0; i < len(row); i += 4 {
			if !bytes.Equal(row[i:i+4], first) {
				return false
			}
		}
	}
	return true
}
