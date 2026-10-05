package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SkeletonDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row items-center gap-2"),
			ui.Skeleton(twi.Class("h-3 w-6 rounded-full")),
			twi.Element(twi.Class("flex flex-col gap-1"),
				ui.Skeleton(twi.Class("h-1 w-40")),
				ui.Skeleton(twi.Class("h-1 w-30")),
			),
		)
	}
}
