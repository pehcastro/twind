package relate

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/style"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const (
	Lit     = "bg-sky-500"
	Pressed = "bg-red-500"
)

func cell(name, classes string, options ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class("px-1 bg-zinc-700 " + classes), twi.Text(name)}, options...)...)
}

func row(classes string, cells ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-row gap-1 " + classes)}, cells...)...)
}

func App(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-full bg-black text-white"),
			row("group", cell("name", ""), cell("badge", "group-hover:bg-sky-500 group-active:bg-red-500"), cell("named", "group-hover/item:bg-sky-500")),
			row("group/item", cell("item", ""), cell("unnamed", "group-hover:bg-sky-500"), cell("itemed", "group-hover/item:bg-sky-500")),
			row("", cell("before", "peer-hover:bg-sky-500"), cell("peer", "peer"), cell("after", "peer-hover:bg-sky-500")),
			row("px-1 has-[:hover]:bg-sky-500", cell("inner", ""), cell("other", "")),
			row("hover:[&>svg]:bg-sky-500", cell("icon", "", twi.Tag(style.ElementSVG)), cell("label", "")),
			row("*:hover:bg-sky-500", twi.Element(twi.Class("px-1"), twi.Text("kid"))),
			row("", cell("plain", ""), cell("still", "")),
		)
	}
}

func Icons() twi.Node {
	return twi.Element(twi.Class("flex flex-row gap-1 [&>svg]:size-4"),
		twi.Element(twi.Tag(style.ElementSVG), twi.Class(Lit), twi.Text("★")),
		twi.Element(twi.Class("w-1"), twi.Element(twi.Tag(style.ElementSVG), twi.Class(Lit), twi.Text("☆"))),
		twi.Element(twi.Text("s")),
		twi.Text("t"),
	)
}
