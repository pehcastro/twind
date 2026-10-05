package ui

import (
	"testing"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

func TestKeyedChildKeepsItsBoxWhenASiblingBeforeItGoes(t *testing.T) {
	first := true
	d := staged(t, 20, 5, func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			rows := []twi.NodeOption{twi.OnKey(func(e *twi.Event) {
				if k := e.Key; !k.Release && k.Key == input.KeyRune && k.Rune == 'x' {
					first = false
					rt.Invalidate()
				}
			})}
			if first {
				rows = append(rows, twi.Element(twi.Key("a"), twi.Class("w-6 bg-red-500 transition-colors duration-200"), twi.Text("A")))
			}
			rows = append(rows, twi.Element(twi.Key("b"), twi.Class("w-6 bg-blue-500 transition-colors duration-200"), twi.Text("B")))
			return twi.Element(rows...)
		}
	})
	x, y, _ := at(d.Frame(), "B")
	settled := d.Frame().At(x, y).Bg
	d.Press("x")
	d.Advance(50 * time.Millisecond)
	bx, by, ok := at(d.Frame(), "B")
	if _, _, gone := at(d.Frame(), "A"); gone || !ok {
		t.Fatalf("A still drawn or B missing:\n%s", d.Frame().Text())
	}
	if got := d.Frame().At(bx, by).Bg; got != settled {
		t.Errorf("B 50 ms after A went: bg %+v, want its own %+v at once; a box matched by index transitions from A's colour", got.RGBA, settled.RGBA)
	}
}
