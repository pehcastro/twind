package style_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/theme"
)

func BenchmarkCompute(b *testing.B) {
	sheet := appSheet(b).WithTheme(builtin(b, "zinc", theme.Dark))
	parent := sheet.Compute(style.ComputedStyle{}, strings.Fields("text-foreground bg-background"))
	for name, classes := range map[string][]string{
		"plain":    strings.Fields("flex flex-col gap-2 p-2 border rounded-lg bg-card text-card-foreground shadow-md"),
		"variants": strings.Fields("flex flex-col gap-2 p-2 border rounded-lg bg-card text-card-foreground shadow-md hover:bg-muted focus-visible:ring-2 disabled:opacity-50 data-[state=open]:bg-accent"),
	} {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				sheet.Compute(parent, classes)
			}
		})
	}
}
