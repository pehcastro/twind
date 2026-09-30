package scene

import (
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
	ScrollContent                          layout.Rect
	Background                             color.Color
	Gradient                               style.Gradient
	Border                                 Border
	Shadows, InsetShadows                  []style.Shadow
	Foreground                             color.Color
	Bold, Italic, Underline, Strikethrough bool
	TextAlign                              style.TextAlign
	Truncate                               bool
	Children                               []Node
	text                                   Text
}

type Text struct {
	clean   string
	wrapped *map[int][]string
}

func Sanitize(raw string) Text {
	clean := text.Sanitize(raw, text.RemoveBidi)
	if clean == "" {
		return Text{}
	}
	return Text{clean, &map[int][]string{}}
}

func (t Text) wrap(width int) []string {
	if t.wrapped == nil {
		return text.Wrap(t.clean, width)
	}
	lines, ok := (*t.wrapped)[width]
	if !ok {
		lines = text.Wrap(t.clean, width)
		(*t.wrapped)[width] = lines
	}
	return lines
}

func (t Text) Size(availableWidth int) (width, height int) {
	lines := t.wrap(availableWidth)
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
		Scroll:   box.Style.Overflow == layout.OverflowScroll,
	}
	if n.Scroll {
		p := box.PaddingBox
		n.ScrollContent = layout.Rect{X: p.X - box.ScrollX, Y: p.Y - box.ScrollY, W: box.ScrollWidth, H: box.ScrollHeight}
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
	n.text = content
	return n
}

func (n Node) Lines() []string {
	if n.text.clean == "" {
		return nil
	}
	if !n.Truncate {
		return n.text.wrap(n.Content.W)
	}
	lines := strings.Split(n.text.clean, "\n")
	for i, line := range lines {
		lines[i] = text.Truncate(line, n.Content.W)
	}
	return lines
}

func (n *Node) Thumb(unit int) (from, to int, ok bool) {
	view, content := n.Padding.H, n.ScrollContent.H
	if !n.Scroll || content <= view || view <= 0 {
		return 0, 0, false
	}
	track := view * unit
	length := max(track*view/content, unit)
	from = (track - length) * (n.Padding.Y - n.ScrollContent.Y) / (content - view)
	return from, from + length, true
}
