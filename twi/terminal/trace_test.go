package terminal

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
)

func TestTraceEachTerminalPhase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.log")
	o, err := offered(func(k string) string { return map[string]string{konst.TraceEnv: path}[k] })
	if err != nil {
		t.Fatal(err)
	}
	term := newFake("\x1b[?2026;1$y", "\x1b[?1u", "\x1b[?62;22c")
	b, err := enter(term, term.tty, Options{}, o)
	if err != nil {
		t.Fatal(err)
	}
	b.Trace("app: %s", "a line from above")
	for range 2 {
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var phases []string
	for line := range strings.Lines(string(text)) {
		_, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
		phases = append(phases, regexp.MustCompile(`[0-9.]+(ms|µs|s)\b`).ReplaceAllString(rest, "T"))
	}
	want := []string{
		"terminal: detecting",
		"terminal: detected identity 0 in T",
		"overlay: none, terminal: no console window to draw on",
		"app: a line from above",
		"terminal: first frame written, 5 bytes",
		"terminal: exit",
	}
	if strings.Join(phases, "\n") != strings.Join(want, "\n") {
		t.Errorf("trace:\n%s\nwant:\n%s", strings.Join(phases, "\n"), strings.Join(want, "\n"))
	}
}

func TestNoTraceWithoutTheEnv(t *testing.T) {
	o, err := offered(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	term := newFake("\x1b[?2026;1$y", "\x1b[?1u", "\x1b[?62;22c")
	b, err := enter(term, term.tty, Options{}, o)
	if err != nil {
		t.Fatal(err)
	}
	b.Trace("nothing %d", 1)
	if b.traced != nil {
		t.Error("a trace file is open without TWIND_TRACE")
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}
}
