package terminal

import (
	"image"
	"slices"
	"testing"
)

func TestCoverageCachedPerCharacterUntilTheFontChanges(t *testing.T) {
	consolas := Font{Face: "Consolas", Size: image.Pt(8, 16)}
	term := newFake(conPTY...)
	term.tty.window, term.tty.face, term.tty.lacking = true, consolas, "☾"
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Exit() //nolint:errcheck
	if b.Capabilities.Font != consolas {
		t.Fatalf("font %+v, want %+v", b.Capabilities.Font, consolas)
	}
	frame := func() {
		if _, err := b.Write([]byte("frame")); err != nil {
			t.Fatal(err)
		}
	}
	covers := func(step, cluster string, want bool, asked ...string) {
		t.Helper()
		if got := b.Covers(cluster); got != want {
			t.Errorf("%s: covers %q %v, want %v", step, cluster, got, want)
		}
		if !slices.Equal(term.tty.asked, asked) {
			t.Errorf("%s: asked %q, want %q", step, term.tty.asked, asked)
		}
	}
	covers("first ask", "☾", false, "Consolas ☾")
	covers("another character", "✓", true, "Consolas ☾", "Consolas ✓")
	covers("cached", "☾", false, "Consolas ☾", "Consolas ✓")
	frame()
	covers("same font next frame", "☾", false, "Consolas ☾", "Consolas ✓")
	term.tty.face.Size = image.Pt(10, 20)
	frame()
	if b.Capabilities.Font != term.tty.face {
		t.Errorf("font after resize %+v, want %+v", b.Capabilities.Font, term.tty.face)
	}
	covers("font resized", "☾", false, "Consolas ☾", "Consolas ✓", "Consolas ☾")
	term.tty.face.Face, term.tty.lacking = "Cascadia Mono", ""
	frame()
	covers("face changed", "☾", true, "Consolas ☾", "Consolas ✓", "Consolas ☾", "Cascadia Mono ☾")
}

func TestCoverageOnlyOnConhost(t *testing.T) {
	consolas := Font{Face: "Consolas", Size: image.Pt(8, 16)}
	term := newFake(windowsTerminal...)
	term.tty.face, term.tty.lacking = consolas, "☾"
	b, err := enter(term, term.tty, Options{}, offer{})
	if err != nil {
		t.Fatal(err)
	}
	if b.Capabilities.Font != (Font{}) || !b.Covers("☾") || len(term.tty.asked) > 0 {
		t.Errorf("windows terminal: font %+v, asked %q, want no font and no asks", b.Capabilities.Font, term.tty.asked)
	}
	if err := b.Exit(); err != nil {
		t.Fatal(err)
	}

	term = newFake(append([]string{"\x1b[3;1R"}, conPTY...)...)
	term.tty.window, term.tty.face = true, consolas
	caps, _, err := query(term, term.tty, offer{})
	if err != nil {
		t.Fatal(err)
	}
	if caps.Font != consolas {
		t.Errorf("query on conhost: font %+v, want %+v", caps.Font, consolas)
	}
}
