package highlight_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/highlight"
)

func BenchmarkGo2000Lines(b *testing.B) {
	file, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "src", "go", "parser", "parser.go"))
	if err != nil {
		b.Fatal(err)
	}
	lines := strings.SplitAfterN(string(file), "\n", 2001)
	src := strings.Join(lines[:2000], "")
	g := highlight.Go()
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for range highlight.Tokens(src, g) {
	}
	spans := 0
	for b.Loop() {
		for range highlight.Tokens(src, g) {
			spans++
		}
	}
	b.ReportMetric(float64(spans)/float64(b.N), "spans/op")
}
