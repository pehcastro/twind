package hello

import "github.com/twind-dev/twind/twi"

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func App() twi.Node {
	return twi.Element(
		twi.Class(
			"flex flex-col gap-2 p-4 "+
				"bg-zinc-950 text-zinc-100",
		),

		twi.Text("Hello Twind"),

		twi.Element(
			twi.Class(
				"border rounded-lg p-2",
			),

			twi.Text("Terminal DOM"),
		),
	)
}

func Grid() twi.Node { return twi.Element(twi.Class("grid")) }

func Borderless() twi.Node { return twi.Element(twi.Class("border border-none p-2"), twi.Text("x")) }
