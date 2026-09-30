package hello

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

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

func Borderless() twi.Node { return twi.Element(twi.Class("border border-none p-2"), twi.Text("x")) }

func Card() twi.Node { return twi.Element(twi.Class("bg-card text-card-foreground"), twi.Text("card")) }

func Surfaces(rt *twi.Runtime) func() twi.Node {
	typed, lit := twi.NewSignal(rt, 0), twi.NewSignal(rt, false)
	return func() twi.Node {
		card := "w-12 border rounded-lg p-1 bg-zinc-800"
		if lit.Get() {
			card = "w-12 border rounded-lg p-1 bg-sky-800"
		}
		return twi.Element(
			twi.Class("flex flex-col gap-1 p-2 bg-zinc-950 text-zinc-100"),
			twi.OnKey(func(k input.KeyEvent) {
				switch k.Rune {
				case 't':
					typed.Set(typed.Get() + 1)
				case 'b':
					lit.Set(!lit.Get())
				}
			}),
			twi.Text("typed "+strconv.Itoa(typed.Get())),
			twi.Element(twi.Class(card), twi.Text("card")),
		)
	}
}
