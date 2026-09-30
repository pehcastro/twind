package surfaces

import (
	"testing"

	"github.com/twind-dev/twind/twi/tailwind"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", "twir_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twir_gen.go is stale against the classes in this package: run go generate")
	}
}
