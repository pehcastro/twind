package scene

import (
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
	Content                                layout.Rect
	Background                             color.Color
	Border                                 Border
	Foreground                             color.Color
	Bold, Italic, Underline, Strikethrough bool
	lines                                  []string
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
	if s.Visibility == style.Hidden {
		return Node{Bounds: box.BorderBox, Content: box.ContentBox}
	}
	own := func(c color.Color) color.Color {
		if c.Kind == color.Current {
			return s.Color
		}
		return c
	}
	edges := box.Style.Border
	return Node{
		Bounds:     box.BorderBox,
		Content:    box.ContentBox,
		Background: own(s.Background),
		Border: Border{
			Style: s.BorderStyle, Radius: s.Radius, Color: own(s.BorderColor),
			Top: edges.Top > 0, Right: edges.Right > 0, Bottom: edges.Bottom > 0, Left: edges.Left > 0,
		},
		Foreground:    s.Color,
		Bold:          s.Bold,
		Italic:        s.Italic,
		Underline:     s.Underline,
		Strikethrough: s.Strikethrough,
		lines:         text.Wrap(content.clean, box.ContentBox.W),
	}
}

func (n Node) Lines() []string { return n.lines }
