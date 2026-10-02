package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type Carousel struct {
	control
	Orientation  Orientation
	Index        int
	OnChange     func(int)
	slides, made int
	turned       int
}

func NewCarousel(rt *twi.Runtime) *Carousel { return &Carousel{control: control{rt: rt}} }

func (c *Carousel) Node(children ...twi.NodeOption) twi.Node {
	c.slides, c.made = c.made, 0
	if c.slides > 0 && c.Index >= c.slides {
		c.Index = c.slides - 1
		c.rt.Invalidate()
	}
	keys := c.behave(func(k input.KeyEvent) bool {
		previous, next := input.KeyArrowLeft, input.KeyArrowRight
		if c.Orientation == Vertical {
			previous, next = input.KeyArrowUp, input.KeyArrowDown
		}
		switch k.Key {
		case previous:
			c.step(-1)
		case next:
			c.step(1)
		default:
			return false
		}
		return true
	})
	return part("relative "+focusRing, slices.Concat(keys, []twi.NodeOption{twi.Data("slot", "carousel")}, children))
}

func (c *Carousel) Content(children ...twi.NodeOption) twi.Node {
	return part("flex h-full w-full overflow-hidden", append([]twi.NodeOption{twi.Data("slot", "carousel-content")}, children...))
}

func (c *Carousel) Item(children ...twi.NodeOption) twi.Node {
	at := c.made
	c.made++
	if at != c.Index {
		return closed()
	}
	slide := ""
	if c.turned != 0 {
		from := map[Orientation]string{Horizontal: "slide-in-from-right", Vertical: "slide-in-from-bottom"}
		if c.turned < 0 {
			from = map[Orientation]string{Horizontal: "slide-in-from-left", Vertical: "slide-in-from-top"}
		}
		slide = " animate-in duration-300 ease-out " + pick("carousel item", c.Orientation, from)
	}
	return part("flex flex-col h-full w-full min-w-0 min-h-0 shrink-0"+slide, append([]twi.NodeOption{twi.Data("slot", "carousel-item")}, children...))
}

func (c *Carousel) Previous(children ...twi.NodeOption) twi.Node {
	return c.arrow(-1, "carousel-previous", pick("carousel previous", c.Orientation, map[Orientation]string{
		Horizontal: "inset-y-0 -left-4",
		Vertical:   "inset-x-0 -top-2",
	}), pick("carousel previous", c.Orientation, map[Orientation]string{Horizontal: "←", Vertical: "↑"}), children)
}

func (c *Carousel) Next(children ...twi.NodeOption) twi.Node {
	return c.arrow(1, "carousel-next", pick("carousel next", c.Orientation, map[Orientation]string{
		Horizontal: "inset-y-0 -right-4",
		Vertical:   "inset-x-0 -bottom-2",
	}), pick("carousel next", c.Orientation, map[Orientation]string{Horizontal: "→", Vertical: "↓"}), children)
}

func (c *Carousel) arrow(by int, slot, at, glyph string, children []twi.NodeOption) twi.Node {
	if len(children) == 0 {
		children = []twi.NodeOption{icon(glyph, "")}
	}
	return part("absolute flex items-center justify-center "+at, []twi.NodeOption{part(button(Outline, SizeIcon, idleRing(Outline)+" "+focusRing)+" rounded-full", append([]twi.NodeOption{
		twi.Data("slot", slot), twi.Focusable(), c.click(func() { c.step(by) }),
	}, children...))})
}

func (c *Carousel) step(by int) {
	if c.slides == 0 {
		return
	}
	c.Index, c.turned = (c.Index+by+c.slides)%c.slides, by
	notify(c.OnChange, c.Index)
}
