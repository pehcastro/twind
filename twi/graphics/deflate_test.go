package graphics

import (
	"testing"

	"github.com/twind-dev/twind/internal/konst/graphics"
)

func TestHuffmanLengthsAreLimitedAndComplete(t *testing.T) {
	cases := map[string][]int32{
		"fibonacci":   make([]int32, graphics.DeflateLiterals),
		"one symbol":  make([]int32, graphics.DeflateDistances),
		"two symbols": make([]int32, graphics.DeflateCodeLengths),
		"uniform":     make([]int32, graphics.DeflateLiterals),
		"steep":       make([]int32, graphics.DeflateCodeLengths),
	}
	a, b := int32(1), int32(1)
	for i := range 40 {
		cases["fibonacci"][i*7] = a
		a, b = b, min(a+b, 1<<30)
	}
	cases["one symbol"][5] = 9
	cases["two symbols"][0], cases["two symbols"][18] = 1, 1000
	for i := range cases["uniform"] {
		cases["uniform"][i] = 3
	}
	for i := range cases["steep"] {
		cases["steep"][i] = 1 << i
	}
	var d deflater
	for name, freq := range cases {
		for _, limit := range []int{graphics.DeflateMaxBits, graphics.DeflateCodeLengthBits} {
			if name != "steep" && name != "two symbols" && limit == graphics.DeflateCodeLengthBits {
				continue
			}
			var h huffman
			d.build(&h, freq, limit)
			kraft, used := 0, 0
			for s, l := range h.lengths[:len(freq)] {
				if int(l) > limit {
					t.Fatalf("%s limit %d: symbol %d has length %d", name, limit, s, l)
				}
				if freq[s] > 0 && l == 0 {
					t.Fatalf("%s limit %d: used symbol %d has no code", name, limit, s)
				}
				if l > 0 {
					kraft += 1 << (limit - int(l))
					used++
				}
			}
			if kraft != 1<<limit || used < 2 {
				t.Fatalf("%s limit %d: kraft %d of %d over %d codes", name, limit, kraft, 1<<limit, used)
			}
		}
	}
}
