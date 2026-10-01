package runtime_test

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
)

type coveringBackend struct{ *backend }

func (coveringBackend) Covers(cluster string) bool { return cluster != "☾" }

func TestCoverageStandInFromTheBackend(t *testing.T) {
	b := newBackend(20, 3)
	b.caps.Font = terminal.Font{Face: "Consolas"}
	rt := twi.New(twi.Backend(coveringBackend{b}, &clock{}), twi.ColorProfile(color.TrueColor))
	done := make(chan error, 1)
	go func() { done <- rt.Run(func() twi.Node { return twi.Text("moon ☾") }) }()
	r := run{rt: rt, b: b, done: done}
	if f := r.next(t); strings.Contains(f, "☾") || !strings.Contains(f, "moon ●") {
		t.Errorf("frame %q, want the moon's stand-in", f)
	}
	if err := r.stop(t); err != nil {
		t.Fatal(err)
	}
}
