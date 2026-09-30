package raster

import (
	"math"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/raster"
)

func TestPhiTableIsTheNormalCDF(t *testing.T) {
	if len(phiTable) != 8*(konst.PhiSteps+1) {
		t.Fatalf("phiTable holds %d bytes, want %d: run go generate ./twi/raster", len(phiTable), 8*(konst.PhiSteps+1))
	}
	low := math.Erf(-konst.ShadowReach / math.Sqrt2)
	for i := range konst.PhiSteps + 1 {
		x := (2*float64(i)/konst.PhiSteps - 1) * konst.ShadowReach
		if got, want := phi(i), (math.Erf(x/math.Sqrt2)-low)/(-2*low); got != want {
			t.Fatalf("phi(%d) = %v, want %v: run go generate ./twi/raster", i, got, want)
		}
	}
}
