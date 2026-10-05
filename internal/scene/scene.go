package scene

import (
	"math"
	"strings"

	lkonst "github.com/pehcastro/twind/internal/konst/layout"

	"github.com/pehcastro/twind/internal/layout"
	"github.com/pehcastro/twind/internal/raster"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/text"
)

type Border struct {
	Style                    style.BorderStyle
	Radius                   style.Radius
	Top, Right, Bottom, Left bool
	Color                    color.Color
}

type Half uint8

const (
	HalfTop Half = 1 << iota
	HalfBottom
)

type Halves struct{ Bounds, Padding, Clip Half }

type Node struct {
	Bounds                                 layout.Rect
	Clip                                   layout.Rect
	Children                               []Node
	Shadows                                []style.Shadow
	ZIndex                                 int
	TopLayer                               int
	Opacity                                float64
	Position                               layout.Position
	Scroll                                 bool
	HidesOverflow                          bool
	Background                             color.Color
	Turn                                   float64
	Shrink                                 float64
	Border                                 Border
	InsetShadows                           []style.Shadow
	Gradient                               style.Gradient
	Pixels                                 *raster.Pixels
	Padding                                layout.Rect
	Content                                layout.Rect
	ScrollContent                          layout.Rect
	Visibility                             style.Visibility
	PointerEvents                          style.PointerEvents
	Foreground                             color.Color
	Bold, Italic, Underline, Strikethrough bool
	TextAlign                              style.TextAlign
	Truncate, NoWrap                       bool
	Halves                                 Halves
	enclosed                               bool
	text                                   Text
	wrapping                               text.Wrapping
	reach                                  layout.Rect
}

func (n *Node) Enclose() {
	n.reach, n.enclosed = n.Bounds, true
	for i := range n.Children {
		c := &n.Children[i]
		if !c.enclosed {
			n.reach = layout.Rect{X: -lkonst.Unbounded / 2, Y: -lkonst.Unbounded / 2, W: lkonst.Unbounded, H: lkonst.Unbounded}
			return
		}
		switch r := c.reach; {
		case r.W <= 0 || r.H <= 0:
		case n.reach.W <= 0 || n.reach.H <= 0:
			n.reach = r
		default:
			x, y := min(n.reach.X, r.X), min(n.reach.Y, r.Y)
			n.reach = layout.Rect{X: x, Y: y, W: max(n.reach.X+n.reach.W, r.X+r.W) - x, H: max(n.reach.Y+n.reach.H, r.Y+r.H) - y}
		}
	}
}

func (n *Node) Holds(x, y int) bool {
	r := n.reach
	return !n.enclosed || x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

type Text struct {
	clean   string
	wrapped *wrapped
}

type wrapped struct {
	wrapping  text.Wrapping
	unbounded []string
	natural   int
	narrower  map[int][]string
	oneLine   bool
	line      [1]string
}

func Sanitize(raw string) Text {
	clean, ascii := raw, printable(raw)
	if !ascii {
		clean = text.Sanitize(raw, text.RemoveBidi)
	}
	if clean == "" {
		return Text{}
	}
	return Text{clean, &wrapped{oneLine: ascii && clean[0] != ' ' && clean[len(clean)-1] != ' ' && !strings.Contains(clean, "  ")}}
}

func printable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

func Wrapping(s *style.ComputedStyle) text.Wrapping {
	return text.Wrapping{
		Word: [...]text.WordBreak{
			style.WordBreakNormal:  text.WordBreakNormal,
			style.WordBreakAll:     text.WordBreakAll,
			style.WordBreakKeepAll: text.WordBreakKeepAll,
		}[s.WordBreak],
		Overflow: [...]text.OverflowWrap{
			style.OverflowWrapNormal:    text.OverflowWrapNormal,
			style.OverflowWrapBreakWord: text.OverflowWrapBreakWord,
			style.OverflowWrapAnywhere:  text.OverflowWrapAnywhere,
		}[s.OverflowWrap],
		Space: [...]text.Space{
			style.WhiteSpaceNormal:  text.SpaceCollapse,
			style.WhiteSpaceNowrap:  text.SpaceCollapse,
			style.WhiteSpacePre:     text.SpacePreserve,
			style.WhiteSpacePreWrap: text.SpacePreserve,
			style.WhiteSpacePreLine: text.SpaceCollapse,
		}[s.WhiteSpace],
	}
}

func (t Text) wrap(b text.Wrapping, width int) []string {
	if t.wrapped == nil {
		return b.Wrap(t.clean, width)
	}
	w := t.wrapped
	if w.unbounded == nil || w.wrapping != b {
		*w = wrapped{wrapping: b, oneLine: w.oneLine, line: [1]string{t.clean}}
		if w.oneLine && b.Dir != text.DirRTL {
			w.unbounded, w.natural = w.line[:], len(t.clean)
		} else {
			w.unbounded = b.Wrap(t.clean, math.MaxInt)
			for _, line := range w.unbounded {
				w.natural = max(w.natural, b.Widths.Width(line))
			}
		}
	}
	if width >= w.natural {
		return w.unbounded
	}
	lines, ok := w.narrower[width]
	if !ok {
		if w.narrower == nil {
			w.narrower = map[int][]string{}
		}
		lines = b.Wrap(t.clean, width)
		w.narrower[width] = lines
	}
	return lines
}

func (t Text) Size(b text.Wrapping, availableWidth int) (width, height int) {
	lines := t.wrap(b, availableWidth)
	if t.wrapped != nil && availableWidth >= t.wrapped.natural {
		return t.wrapped.natural, len(lines)
	}
	for _, line := range lines {
		width = max(width, b.Widths.Width(line))
	}
	return width, len(lines)
}

func (t Text) MinContent(b text.Wrapping) int {
	return b.MinContent(t.clean)
}

func New(box *layout.Box, s style.ComputedStyle, content Text) Node {
	var n Node
	n.Fill(box, &s, content)
	return n
}

func (n *Node) Fill(box *layout.Box, s *style.ComputedStyle, content Text) {
	*n = Node{
		Bounds:        box.BorderBox,
		Padding:       box.PaddingBox,
		Content:       box.ContentBox,
		Clip:          box.Clip,
		Position:      box.Style.Position,
		ZIndex:        box.Style.ZIndex,
		Opacity:       s.Opacity,
		Scroll:        box.Style.Overflow == layout.OverflowScroll,
		HidesOverflow: box.Style.Overflow != layout.OverflowVisible,
		Visibility:    s.Visibility,
		PointerEvents: s.PointerEvents,
	}
	if n.Scroll {
		p := box.PaddingBox
		n.ScrollContent = layout.Rect{X: p.X - box.ScrollX, Y: p.Y - box.ScrollY, W: box.ScrollWidth, H: box.ScrollHeight}
	}
	if s.Visibility == style.Hidden {
		return
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
	n.text, n.wrapping = content, Wrapping(s)
}

func (n *Node) Direct(d text.Direction) {
	n.wrapping.Dir = d
	if d == text.DirRTL && (n.TextAlign == style.TextLeft || n.TextAlign == style.TextJustify) {
		n.TextAlign = style.TextRight
	}
}

func (n *Node) Lines(w text.Widths) []string {
	if n.text.clean == "" {
		return nil
	}
	b := n.wrapping
	b.Widths = w
	if !n.NoWrap && !n.Truncate {
		return n.text.wrap(b, n.Content.W)
	}
	lines := n.text.wrap(b, math.MaxInt)
	if !n.Truncate {
		return lines
	}
	truncated := make([]string, len(lines))
	for i, line := range lines {
		truncated[i] = w.Truncate(line, n.Content.W)
	}
	return truncated
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
