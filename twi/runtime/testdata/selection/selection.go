package selection

import "github.com/pehcastro/twind/twi"

//go:generate go run github.com/pehcastro/twind/internal/twirgen

const (
	Card    = "flex flex-col w-24 border border-zinc-500 bg-zinc-900 px-1"
	Wrapped = "alpha bravo charlie delta echo foxtrot"
)

func App(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row gap-2 p-1 h-full bg-black text-white"),
			twi.Element(twi.Class(Card),
				twi.Text(Wrapped),
				twi.Element(twi.Class("flex flex-row gap-1"),
					twi.Text("golf"),
					twi.Element(twi.Class("select-none text-zinc-400"), twi.Text("nosel")),
					twi.Text("hotel"),
				),
				twi.Element(twi.Class("select-all"), twi.Text("whole thing")),
			),
			twi.Element(twi.Class(Card), twi.Text("india juliet kilo"), twi.Text("lima mike"),
				twi.Element(twi.Class("truncate"), twi.Text("中文 wide and truncated beyond the edge")),
			),
		)
	}
}
