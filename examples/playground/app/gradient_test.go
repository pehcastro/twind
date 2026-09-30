package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestExamplesUseThemeGradients(t *testing.T) {
	palette := regexp.MustCompile(`(^|[\s"':])(from|via|to)-(red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|slate|gray|zinc|neutral|stone|black|white)\b`)
	files := 0
	err := filepath.WalkDir(filepath.Join("..", ".."), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || d.Name() == "twir_gen.go" {
			return err
		}
		files++
		src, err := os.ReadFile(path)
		for i, line := range strings.Split(string(src), "\n") {
			if m := palette.FindString(line); m != "" {
				t.Errorf("%s:%d: %s is a palette gradient stop, use a theme colour such as from-primary-700 or to-chart-2", filepath.ToSlash(path), i+1, strings.TrimLeft(m, " \t\"':"))
			}
		}
		return err
	})
	if err != nil || files < 10 {
		t.Fatalf("%d Go files under examples: %v", files, err)
	}
}
