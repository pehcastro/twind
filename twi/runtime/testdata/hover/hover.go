package hover

import "github.com/twind-dev/twind/twi"

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const Pill = "px-1 bg-zinc-700 hover:bg-sky-500 active:bg-red-500 focus-visible:underline"

type Trace struct {
	Events []string
	Open   bool
}

func (t *Trace) record(event string) { t.Events = append(t.Events, event) }

func (t *Trace) watched(name string, options ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{
		twi.Key(name),
		twi.OnPointerEnter(func() { t.record("enter " + name) }),
		twi.OnPointerLeave(func() { t.record("leave " + name) }),
		twi.OnClick(func(*twi.Event) { t.record("click " + name) }),
	}, options...)...)
}

func (t *Trace) App(rt *twi.Runtime) func() twi.Node {
	return func() twi.Node {
		root := []twi.NodeOption{twi.Class("flex flex-col gap-1 p-1 h-full bg-black text-white")}
		if t.Open {
			root = append(root, t.watched("overlay", twi.Class("fixed top-0 left-0 w-8 h-3 z-50 bg-zinc-900"), twi.Text("overlay"),
				twi.OnPointerDownOutside(func() {
					t.record("outside")
					t.Open = false
					rt.Invalidate()
				})))
		}
		return twi.Element(append(root,
			t.watched("row", twi.Class("flex flex-row gap-2"),
				t.watched("one", twi.Focusable(), twi.Class(Pill), twi.Text("one")),
				t.watched("two", twi.Focusable(), twi.Class(Pill), twi.Text("two")),
			),
			twi.Element(twi.Class("text-zinc-400"), twi.Text("plain text")),
		)...)
	}
}
