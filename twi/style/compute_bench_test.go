package style_test

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
)

func BenchmarkCompute(b *testing.B) {
	sheet := appSheet(b).WithTheme(builtin(b, "twind", theme.Dark))
	parent := sheet.Compute(style.ComputedStyle{}, strings.Fields("text-foreground bg-background"))
	for _, tc := range []struct {
		name    string
		classes []string
		node    style.NodeState
	}{
		{"plain", strings.Fields("flex flex-col gap-2 p-2 border rounded-lg bg-card text-card-foreground shadow-md"), style.NodeState{}},
		{"variants", strings.Fields("flex flex-col gap-2 p-2 border rounded-lg bg-card text-card-foreground shadow-md hover:bg-muted focus-visible:ring-2 disabled:opacity-50 data-[state=open]:bg-accent"), style.NodeState{}},
		{"ring", strings.Fields("border shadow-xs focus-visible:ring-ring/50 focus-visible:ring-2"), style.NodeState{States: style.StateFocusVisible}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sheet.ComputeState(parent, tc.classes, tc.node)
			}
		})
	}
}
