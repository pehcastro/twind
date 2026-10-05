package counter

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
)

func New(rt *twi.Runtime) func() twi.Node {
	count := twi.NewSignal(rt, 0)
	return func() twi.Node {
		return twi.Element(
			twi.OnKey(func(e *twi.Event) {
				switch e.Key.Rune {
				case '+':
					count.Set(count.Get() + 1)
				case '-':
					count.Set(count.Get() - 1)
				}
			}),
			twi.Text("count "+strconv.Itoa(count.Get())),
		)
	}
}
