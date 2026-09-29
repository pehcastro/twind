package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/tailwind"
)

func TestGeneratedHello(t *testing.T) {
	dir := filepath.Join("testdata", "hello")
	stale, err := tailwind.Stale(dir, konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale: run go generate there", filepath.Join(dir, konst.GeneratedFile))
	}
	src, err := os.ReadFile(filepath.Join(dir, konst.GeneratedFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(src), "// "+konst.IRMagic+" version=1 ") {
		t.Errorf("generated file does not start with the magic and version:\n%.120s", src)
	}
	for _, runtimeParse := range []string{"twi/css", "twi/tailwind", "Parse(", "strings."} {
		if strings.Contains(string(src), runtimeParse) {
			t.Errorf("generated file contains %q", runtimeParse)
		}
	}
}
