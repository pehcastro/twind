package twi_test

import (
	"strconv"
	"testing"

	"github.com/pehcastro/twind/twi"
)

func BenchmarkBuildTree1000(b *testing.B) {
	levels := []string{"flex flex-col border", "flex flex-row gap-1", "flex flex-col grow"}
	b.ReportAllocs()
	for b.Loop() {
		left := 997
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
		twi.Element(twi.Text("keys "+strconv.Itoa(b.N)), twi.Element(opts...))
	}
}
