package ui

import (
	"fmt"
	"image"

	"github.com/pehcastro/twind/twi"
)

type MessageScroller struct {
	rt            *twi.Runtime
	following     bool
	view, content *twi.Ref
	end           string
	seen          image.Point
}

func NewMessageScroller(rt *twi.Runtime) *MessageScroller {
	s := &MessageScroller{rt: rt, following: true, view: twi.NewRef(rt), content: twi.NewRef(rt)}
	s.end = fmt.Sprintf("\x00%p", s)
	return s
}

func (s *MessageScroller) Node(children ...twi.NodeOption) twi.Node {
	return part("group/message-scroller relative flex size-full min-h-0 flex-col overflow-hidden", append([]twi.NodeOption{twi.Data("slot", "message-scroller")}, children...))
}

func (s *MessageScroller) Viewport(children ...twi.NodeOption) twi.Node {
	sizes := image.Pt(s.content.Bounds().Dy(), s.view.Bounds().Dy())
	if s.following && sizes != s.seen {
		s.seen = sizes
		s.rt.Dispatch(s.latest)
	}
	return part("flex flex-1 flex-col min-h-0 min-w-0 overflow-y-auto "+focusRing, []twi.NodeOption{
		twi.Data("slot", "message-scroller-viewport"), twi.Focusable(), twi.Measure(s.view), twi.OnScroll(s.scrolled),
		part("flex shrink-0 min-h-full flex-col gap-1", append([]twi.NodeOption{twi.Data("slot", "message-scroller-content"), twi.Measure(s.content)}, children...)),
		part("shrink-0", []twi.NodeOption{twi.Key(s.end)}),
	})
}

func (s *MessageScroller) Item(children ...twi.NodeOption) twi.Node {
	return part("min-w-0 shrink-0", append([]twi.NodeOption{twi.Data("slot", "message-scroller-item")}, children...))
}

func (s *MessageScroller) Button(children ...twi.NodeOption) twi.Node {
	if s.following {
		return closed()
	}
	if len(children) == 0 {
		children = []twi.NodeOption{icon("↓", "")}
	}
	return part(button(Secondary, SizeIcon, idleRing(Outline)+" "+focusRing)+" absolute bottom-1 left-1/2 -translate-x-1/2 rounded-full animate-in fade-in-0 zoom-in-95 duration-200", append([]twi.NodeOption{
		twi.Data("slot", "message-scroller-button"), twi.Focusable(), twi.OnClick(func(*twi.Event) { s.latest() }),
	}, children...))
}

func (s *MessageScroller) latest() {
	s.following = true
	s.rt.ScrollIntoView(s.end)
}

func (s *MessageScroller) scrolled(offset image.Point) {
	if following := offset.Y+s.view.Bounds().Dy() >= s.content.Bounds().Dy(); following != s.following {
		s.following = following
		s.rt.Invalidate()
	}
}
