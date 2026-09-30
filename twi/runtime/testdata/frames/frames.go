package frames

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

type Layers struct {
	A, B   bool
	Clicks []string
}

func (l *Layers) App(rt *twi.Runtime) func() twi.Node {
	click := func(name string) twi.NodeOption {
		return twi.OnClick(func(*twi.Event) { l.Clicks = append(l.Clicks, name) })
	}
	menu := func(open bool, key, classes string, items ...string) []twi.NodeOption {
		if !open {
			return nil
		}
		options := []twi.NodeOption{twi.Key(key), twi.Class(classes), twi.TopLayer()}
		for _, item := range items {
			options = append(options, twi.Element(twi.Text(item)))
		}
		return []twi.NodeOption{twi.Element(options...)}
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row gap-2 h-full bg-black text-white"),
			twi.OnKey(func(k input.KeyEvent) {
				l.A = l.A != (k.Rune == 'a')
				l.B = l.B != (k.Rune == 'b')
				rt.Invalidate()
			}),
			twi.Element(slices.Concat(
				[]twi.NodeOption{twi.Class("relative w-16 h-5 overflow-hidden border border-zinc-500"), twi.Text("card")},
				menu(l.A, "a", "absolute top-1 left-1 w-6 flex flex-col bg-zinc-100 text-black", "a1", "a2", "a3", "a4"),
				menu(l.B, "b", "absolute top-2 left-3 w-6 flex flex-col bg-sky-500 text-black", "b1", "b2", "b3", "b4"),
			)...),
			twi.Element(twi.Class("relative flex flex-col w-12"),
				twi.Element(twi.Class("bg-zinc-700"), click("press"), twi.Text("press")),
				twi.Element(twi.Class("bg-zinc-700"), click("ghost"), twi.Text("ghost")),
				twi.Element(twi.Class("absolute top-0 left-0 w-12 h-1 pointer-events-none bg-sky-500/50"), click("glass")),
				twi.Element(twi.Class("invisible absolute top-1 left-0 w-12 h-1"), click("hidden"), twi.Text("hidden")),
			),
		)
	}
}

func Flag(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-full bg-black text-white"),
			twi.Element(twi.Class("w-fit border border-zinc-500"), twi.Text("🇧🇷ok")),
			twi.Element(twi.Class("w-20 aspect-video bg-sky-500")),
		)
	}
}

func Closing(rt *twi.Runtime) func() twi.Node {
	focused := twi.NewSignal(rt, 0)
	open, shown := false, false
	return func() twi.Node {
		field := twi.Element(twi.Key("field"), twi.Focusable(), twi.AutoFocus(), twi.OnFocus(func() { focused.Set(focused.Get() + 1) }), twi.Text([]string{"field", "once", "twice"}[min(focused.Get(), 2)]))
		keys := twi.OnKey(func(k input.KeyEvent) {
			open = k.Rune == 'o' || open && k.Key != input.KeyEscape
			rt.Invalidate()
		})
		switch {
		case open:
			shown = true
			return twi.Element(keys, field, twi.Element(twi.Key("dialog"), twi.FocusScope(), twi.Element(twi.Focusable(), twi.Text("dialog open"))))
		case shown:
			rt.Dispatch(func() {
				shown = false
				rt.Invalidate()
			})
			return twi.Element(keys, field, twi.Element(twi.Key("closing"), twi.Text("dialog closing")))
		}
		return twi.Element(keys, field)
	}
}

func Entering() twi.Node {
	return twi.Element(twi.Class("animate-in fade-in-0 duration-200 bg-white text-black"), twi.Text("entered"))
}

func Pulse(rt *twi.Runtime) func() twi.Node {
	on := true
	return func() twi.Node {
		pulse := twi.Element(twi.Class("w-10 h-1 bg-white animate-pulse"))
		if !on {
			pulse = twi.Text("still")
		}
		return twi.Element(twi.Class("h-full bg-black text-white"),
			twi.OnKey(func(k input.KeyEvent) {
				on = on && k.Rune != 's'
				rt.Invalidate()
			}),
			pulse,
		)
	}
}
