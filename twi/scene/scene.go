package scene

import (
	"fmt"
	"strings"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/text"
)

type Border struct {
	Style                    style.BorderStyle
	Radius                   style.Radius
	Top, Right, Bottom, Left bool
	Color                    color.Color
}

type Node struct {
	Bounds                                 layout.Rect
	Padding                                layout.Rect
	Content                                layout.Rect
	Clip                                   layout.Rect
	Position                               layout.Position
	ZIndex                                 int
	Opacity                                float64
	Scroll                                 bool
	Background                             color.Color
	Gradient                               style.Gradient
	Border                                 Border
	Shadows, InsetShadows                  []style.Shadow
	Foreground                             color.Color
	Bold, Italic, Underline, Strikethrough bool
	TextAlign                              style.TextAlign
	Truncate                               bool
	Children                               []Node
	text                                   string
}

type Text struct{ clean string }

func Sanitize(raw string) Text { return Text{text.Sanitize(raw, text.RemoveBidi)} }

func (t Text) Size(availableWidth int) (width, height int) {
	lines := text.Wrap(t.clean, availableWidth)
	for _, line := range lines {
		width = max(width, text.Width(line))
	}
	return width, len(lines)
}

func New(box *layout.Box, s style.ComputedStyle, content Text) Node {
	n := Node{
		Bounds:   box.BorderBox,
		Padding:  box.PaddingBox,
		Content:  box.ContentBox,
		Clip:     box.Clip,
		Position: box.Style.Position,
		ZIndex:   box.Style.ZIndex,
		Opacity:  s.Opacity,
		Scroll:   scrolls(s.OverflowX) || scrolls(s.OverflowY),
	}
	if s.Visibility == style.Hidden {
		return n
	}
	own := func(c color.Color) color.Color {
		if c.Kind == color.Current {
			return s.Color
		}
		return c
	}
	edges := box.Style.Border
	n.Background = own(s.Background)
	n.Gradient = s.Gradient
	for _, stop := range []*style.GradientStop{&n.Gradient.From, &n.Gradient.Via, &n.Gradient.To} {
		stop.Color = own(stop.Color)
	}
	n.Border = Border{
		Style: s.BorderStyle, Radius: s.Radius, Color: own(s.BorderColor),
		Top: edges.Top > 0, Right: edges.Right > 0, Bottom: edges.Bottom > 0, Left: edges.Left > 0,
	}
	for _, sh := range s.Shadows {
		sh.Color = own(sh.Color)
		n.Shadows = append(n.Shadows, sh)
	}
	for _, sh := range s.InsetShadows {
		sh.Color = own(sh.Color)
		n.InsetShadows = append(n.InsetShadows, sh)
	}
	n.Foreground = s.Color
	n.Bold, n.Italic, n.Underline, n.Strikethrough = s.Bold, s.Italic, s.Underline, s.Strikethrough
	n.TextAlign = s.TextAlign
	n.text = content.clean
	return n
}

func (n Node) Lines() []string {
	if !n.Truncate {
		return text.Wrap(n.text, n.Content.W)
	}
	lines := strings.Split(n.text, "\n")
	for i, line := range lines {
		lines[i] = text.Truncate(line, n.Content.W)
	}
	return lines
}

func scrolls(o style.Overflow) bool {
	switch o {
	case style.OverflowVisible, style.OverflowHidden:
		return false
	case style.OverflowScroll, style.OverflowAuto:
		return true
	}
	panic(fmt.Sprintf("scene: unknown overflow %d", o))
}
