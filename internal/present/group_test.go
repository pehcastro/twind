package present

import (
	"fmt"
	"image"
	imagecolor "image/color"
	"image/draw"
	"math"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

func masked(s *Screen) *image.RGBA {
	f := &s.scenes[s.turn]
	targets := []*image.RGBA{image.NewRGBA(s.bounds)}
	type open struct {
		layer       int
		own, hidden bool
		opacity     float64
	}
	var stack []open
	pop := func() {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if top.own {
			src := targets[len(targets)-1]
			targets = targets[:len(targets)-1]
			alpha := image.NewUniform(imagecolor.Alpha{A: uint8(math.Round(top.opacity * math.MaxUint8))})
			draw.DrawMask(targets[len(targets)-1], s.bounds, src, image.Point{}, alpha, image.Point{}, draw.Over)
		}
	}
	for i := range f.Layers {
		l := &f.Layers[i]
		for len(stack) > 0 && stack[len(stack)-1].layer != l.Parent {
			pop()
		}
		hidden := l.Opacity <= 0 || len(stack) > 0 && stack[len(stack)-1].hidden
		own := !hidden && l.Opacity < 1
		if own {
			targets = append(targets, image.NewRGBA(s.bounds))
		}
		stack = append(stack, open{i, own, hidden, l.Opacity})
		if hidden {
			continue
		}
		for _, b := range l.Boxes {
			at := b.Visual.Add(l.Origin)
			if v := at.Intersect(l.Clip).Intersect(s.bounds); !v.Empty() {
				over(targets[len(targets)-1], v, s.cache[b.Look], v.Min.Sub(at.Min), 0)
			}
		}
	}
	for len(stack) > 0 {
		pop()
	}
	if s.Profile == color.ANSI256 {
		quantise(targets[0].Pix)
	}
	return targets[0]
}

func ink(r, g, b, a uint8) color.RGBA { return color.RGBA{R: r, G: g, B: b, A: a} }

var opaque, glass = ink(30, 200, 120, 255), ink(200, 180, 20, 128)

func box(x, y, w, h int, bg color.RGBA) scene.Node {
	n := flatPage(bg, "")
	n.Bounds, n.Padding, n.Content = layout.Rect{X: x, Y: y, W: w, H: h}, layout.Rect{X: x, Y: y, W: w, H: h}, layout.Rect{X: x, Y: y, W: w, H: h}
	return n
}

func card(x, y int, bg color.RGBA) scene.Node {
	n := box(x, y, 12, 5, bg)
	n.Border.Radius = style.RadiusLg
	n.Shadows = []style.Shadow{{Y: 4, Blur: 12, Spread: -2, Color: color.Color{Kind: color.Literal, RGBA: ink(0, 0, 0, 90)}}}
	return n
}

func faded(opacity float64, children ...scene.Node) scene.Node {
	g := box(0, 0, cols, rows, color.RGBA{})
	g.Opacity, g.Children = opacity, children
	return g
}

func page(children ...scene.Node) scene.Node {
	p := flatPage(ink(255, 255, 255, 255), "")
	p.Gradient = style.Gradient{
		GradientLine: style.GradientLine{Kind: style.GradientLinear, Direction: style.ToRight},
		From:         style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: ink(250, 20, 60, 255)}},
		To:           style.GradientStop{Color: color.Color{Kind: color.Literal, RGBA: ink(10, 90, 240, 255)}, Position: 1},
	}
	p.Children = children
	return p
}

func sameAsMasked(t *testing.T, s *Screen, name string) {
	t.Helper()
	got, want := s.image(), masked(s)
	for i := 0; i < len(want.Pix); i += 4 {
		if g, w := got.Pix[i:i+4], want.Pix[i:i+4]; string(g) != string(w) {
			t.Errorf("%s: pixel %d,%d is %v, want %v", name, i/4%want.Rect.Dx(), i/4/want.Rect.Dx(), g, w)
			return
		}
	}
}

func TestGroupsCompositeAsDrawMaskDoes(t *testing.T) {
	scenes := map[string]scene.Node{}
	for _, opacity := range []float64{0.02, 0.3, 0.5, 0.97} {
		scenes[fmt.Sprintf("one card at %.2f", opacity)] = page(faded(opacity, card(4, 3, opaque)))
		scenes[fmt.Sprintf("one translucent box at %.2f", opacity)] = page(faded(opacity, box(30, 2, 20, 6, glass)))
		scenes[fmt.Sprintf("two overlapping at %.2f", opacity)] = page(faded(opacity, card(4, 3, opaque), box(10, 5, 20, 6, glass)))
		scenes[fmt.Sprintf("nested at %.2f", opacity)] = page(faded(opacity, box(40, 10, 30, 8, opaque), faded(0.5, card(44, 12, glass))))
		scenes[fmt.Sprintf("a group holding one group at %.2f", opacity)] = page(faded(opacity, faded(0.5, card(60, 3, opaque))))
	}
	scenes["siblings of one look at two opacities"] = page(faded(0.4, box(2, 16, 10, 3, opaque)), faded(0.8, box(20, 16, 10, 3, opaque)), faded(0.8, box(38, 16, 10, 3, glass)))
	scenes["a hidden group"] = page(faded(0, card(4, 3, opaque)), faded(0.5, box(50, 3, 10, 3, opaque)))
	for name, root := range scenes {
		s, _ := screen(terminal.GraphicsSixel)
		frame(t, s, root)
		sameAsMasked(t, s, name)
	}
}

func TestFadingOverAKeptPageMatchesAFullComposite(t *testing.T) {
	steps := []struct {
		name string
		root scene.Node
	}{
		{"fade at 0.1", page(card(4, 3, opaque), faded(0.1, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"fade at 0.3, the page kept", page(card(4, 3, opaque), faded(0.3, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"fade at 0.4 over the kept page", page(card(4, 3, opaque), faded(0.4, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"the card under it moved", page(card(5, 3, opaque), faded(0.6, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"fade at 0.7", page(card(5, 3, opaque), faded(0.7, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"the layer moves down over the kept page", page(card(5, 3, opaque), faded(0.8, box(10, 9, 20, 6, opaque), card(12, 11, glass)))},
		{"fade at 0.8", page(card(5, 3, opaque), faded(0.8, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"a second card joins the page", page(card(5, 3, opaque), card(40, 2, glass), faded(0.9, box(10, 5, 20, 6, opaque), card(12, 6, glass)))},
		{"the layer gone", page(card(5, 3, opaque), card(40, 2, glass))},
		{"the layer back at 0.3", page(card(5, 3, opaque), card(40, 2, glass), faded(0.3, box(10, 5, 20, 6, opaque)))},
	}
	for _, p := range []color.Profile{color.TrueColor, color.ANSI256} {
		s, _ := screen(terminal.GraphicsSixel)
		s.Profile = p
		for _, step := range steps {
			frame(t, s, step.root)
			sameAsMasked(t, s, fmt.Sprintf("profile %d, %s", p, step.name))
		}
	}
}
