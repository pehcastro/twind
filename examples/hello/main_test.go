package main

import (
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi/tailwind"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run go generate", konst.GeneratedFile)
	}
}
