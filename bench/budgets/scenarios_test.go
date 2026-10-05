package budgets_test

import (
	"strconv"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/tailwind"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen -o twir_gen_test.go

const generated = "twir_gen_test.go"

func hello() twi.Node {
	return twi.Element(
		twi.Class("flex flex-col gap-2 p-4 bg-zinc-950 text-zinc-100"),
		twi.Text("Hello Twind"),
		twi.Element(twi.Class("border rounded-lg p-2"), twi.Text("Terminal DOM")),
	)
}

func tree(nodes int) twi.Node {
	left := nodes - 1
	levels := []string{"flex flex-col border", "flex flex-row gap-1", "flex flex-col grow"}
	var grow func(depth int) twi.Node
	grow = func(depth int) twi.Node {
		left--
		if depth == len(levels) {
			return twi.Text("item " + strconv.Itoa(left))
		}
		opts := []twi.NodeOption{twi.Class(levels[depth])}
		for range 4 {
			if left == 0 {
				break
			}
			opts = append(opts, grow(depth+1))
		}
		return twi.Element(opts...)
	}
	opts := []twi.NodeOption{twi.Class("flex flex-col bg-zinc-950 text-zinc-100")}
	for left > 0 {
		opts = append(opts, grow(0))
	}
	return twi.Element(opts...)
}

func keys(rt *twi.Runtime) func() twi.Node {
	pressed := twi.NewSignal(rt, 0)
	return func() twi.Node {
		return twi.Element(
			twi.OnKey(func(input.KeyEvent) { pressed.Set(pressed.Get() + 1) }),
			twi.Text("keys "+strconv.Itoa(pressed.Get())),
			tree(keyNodes-2),
		)
	}
}

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", generated)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run go generate", generated)
	}
}
