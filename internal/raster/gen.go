//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"go/format"
	"log"
	"math"
	"os"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/raster"
)

func main() {
	var table []byte
	low := math.Erf(-konst.ShadowReach / math.Sqrt2)
	for i := range konst.PhiSteps + 1 {
		t := (2*float64(i)/konst.PhiSteps - 1) * konst.ShadowReach
		table = binary.LittleEndian.AppendUint64(table, math.Float64bits((math.Erf(t/math.Sqrt2)-low)/(-2*low)))
	}
	var escaped strings.Builder
	for _, b := range table {
		fmt.Fprintf(&escaped, "\\x%02x", b)
	}
	src, err := format.Source(fmt.Appendf(nil, "package raster\n\nconst phiTable = \"%s\"\n", escaped.String()))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("phi_gen.go", src, 0o644); err != nil {
		log.Fatal(err)
	}
}
