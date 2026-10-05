package highlight_test

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/highlight"
)

func benchSpans(b *testing.B, src string, g *highlight.Grammar) {
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

func benchFile(b *testing.B, name string, g *highlight.Grammar) {
	src, err := os.ReadFile(filepath.Join("testdata", "bench", name))
	if err != nil {
		b.Fatal(err)
	}
	benchSpans(b, string(src), g)
}

func BenchmarkGo2000Lines(b *testing.B) {
	file, err := os.ReadFile(filepath.Join(build.Default.GOROOT, "src", "go", "parser", "parser.go"))
	if err != nil {
		b.Fatal(err)
	}
	lines := strings.SplitAfterN(string(file), "\n", 2001)
	benchSpans(b, strings.Join(lines[:2000], ""), highlight.Go())
}

func BenchmarkTypeScript2000Lines(b *testing.B) {
	benchFile(b, "typescript.ts", highlight.TypeScript())
}

func BenchmarkTSX2000Lines(b *testing.B) { benchFile(b, "tsx.tsx", highlight.TSX()) }

func BenchmarkCSS2000Lines(b *testing.B) { benchFile(b, "css.css", highlight.CSS()) }

func BenchmarkYAML2000Lines(b *testing.B) { benchFile(b, "yaml.yaml", highlight.YAML()) }

func BenchmarkMarkdown2000Lines(b *testing.B) { benchFile(b, "markdown.md", highlight.Markdown()) }

func BenchmarkConsole2000Lines(b *testing.B) { benchFile(b, "console.txt", highlight.Console()) }
