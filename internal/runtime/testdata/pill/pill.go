package pill

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func pill(name string, options ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{
		twi.Key(name),
		twi.Focusable(),
		twi.Class("rounded-full px-1 bg-zinc-700 text-white disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-sky-500 data-[state=on]:underline"),
		twi.Text(name),
	}, options...)...)
}

func Many(count int) func(*twi.Runtime) func() twi.Node {
	return func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			pills := []twi.NodeOption{twi.Class("flex flex-row flex-wrap gap-2 p-1 h-full bg-black focus-within:bg-zinc-900")}
			for i := range count {
				pills = append(pills, pill("pill "+strconv.Itoa(i)))
			}
			return twi.Element(pills...)
		}
	}
}

func App(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-black"),
			twi.Element(twi.Class("flex flex-row gap-4 px-2 py-1 focus-within:bg-zinc-900"), pill("one"), pill("two", twi.Data("state", "on"))),
			twi.Element(twi.Class("flex flex-row gap-4 px-2 py-1"), pill("off", twi.Disabled())),
		)
	}
}
