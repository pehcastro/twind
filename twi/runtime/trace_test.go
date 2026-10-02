package runtime_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/runtime"
)

type tracing struct {
	*backend
	mu    sync.Mutex
	lines []string
}

func (b *tracing) Trace(format string, args ...any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, regexp.MustCompile(`[0-9.]+(ms|µs|s)\b`).ReplaceAllString(fmt.Sprintf(format, args...), "T"))
}

func TestRuntimeTracesEachPhase(t *testing.T) {
	for _, dev := range []bool{true, false} {
		a := &devApp{b: newBackend(30, 7), page: "intro", done: make(chan error, 1)}
		traced := &tracing{backend: a.b}
		a.grid = blank(a.b)
		state := ""
		if dev {
			state = filepath.Join(t.TempDir(), "state.json")
		}
		a.rt = runtime.New(runtime.Config{Clock: &clock{}, Sheet: scrollSheet(t), Profile: color.None, DevState: state})
		go func() { a.done <- a.rt.Run(traced, a.tree) }()
		a.frame(t)
		a.b.events <- wheel(1, 1, input.MouseWheelDown)
		a.frame(t)
		a.b.events <- wheel(1, 1, input.MouseWheelDown)
		a.frame(t)
		a.quit(t)
		want := []string{"runtime: first frame drawn in T", "runtime: first input", "runtime: first frame after the first input, T after it"}
		if dev {
			want = append(want, "runtime: snapshot written in T")
		}
		if got := strings.Join(traced.lines, "\n"); got != strings.Join(want, "\n") {
			t.Errorf("dev %t, trace:\n%s\nwant:\n%s", dev, got, strings.Join(want, "\n"))
		}
	}
}
