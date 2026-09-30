package text

import (
	"slices"
	"strings"
	"testing"
)

const (
	brazil  = "\U0001F1E7\U0001F1F7"
	family  = "\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467"
	rainbow = "\U0001F3F3\U0000FE0F\U0000200D\U0001F308"
	heart   = "\U00002764\U0000FE0F"
	thumb   = "\U0001F44D\U0001F3FD"
	keycap  = "1\U0000FE0F\U000020E3"
)

func TestOverrideWidth(t *testing.T) {
	cases := []struct {
		name  string
		w     Widths
		s     string
		width int
	}{
		{"flag override", Widths{Flag: 1}, brazil, 1},
		{"flag default", Widths{}, brazil, 2},
		{"zero is no override", Widths{Flag: 0, VS16: 1}, brazil, 2},
		{"lone regional indicator is not a flag", Widths{Flag: 4}, "\U0001F1E7", 2},
		{"two flags", Widths{Flag: 1}, brazil + brazil, 2},
		{"zwj family", Widths{ZWJ: 6, VS16: 1}, family, 6},
		{"zwj holding fe0f is zwj", Widths{ZWJ: 4, VS16: 1}, rainbow, 4},
		{"vs16", Widths{VS16: 1}, heart, 1},
		{"text presentation untouched", Widths{VS16: 2}, "\U00002764\U0000FE0E", 1},
		{"modifier", Widths{Modifier: 4, VS16: 1}, thumb, 4},
		{"keycap holding fe0f is keycap", Widths{Keycap: 3, VS16: 1}, keycap, 3},
		{"plain text untouched", Widths{Flag: 1, ZWJ: 1, VS16: 1, Modifier: 1, Keycap: 1}, "a中e\U00000301", 4},
		{"mixed", Widths{Flag: 1}, "a" + brazil + "b", 3},
	}
	for _, c := range cases {
		if got := c.w.Width(c.s); got != c.width {
			t.Errorf("%s: Width(%+q) = %d, want %d", c.name, c.s, got, c.width)
		}
	}
}

func TestOverrideWrapKeepsBorders(t *testing.T) {
	const inner = 6
	w := Widths{Flag: 1}
	lines := w.Wrap("go "+brazil+brazil+brazil+" ok "+brazil+"abcdefg", inner)
	var box []string
	for _, line := range lines {
		pad := inner - w.Width(line)
		if pad < 0 {
			t.Fatalf("line %+q is %d wide, over %d", line, w.Width(line), inner)
		}
		box = append(box, "|"+line+strings.Repeat(" ", pad)+"|")
	}
	want := []string{"|go " + brazil + brazil + brazil + "|", "|ok " + brazil + "  |", "|abcdef|", "|g     |"}
	if !slices.Equal(box, want) {
		t.Errorf("box %+q, want %+q", box, want)
	}
	for _, row := range box {
		if col := w.Width(row); col != inner+2 {
			t.Errorf("row %+q ends at column %d, want %d", row, col, inner+2)
		}
	}
	if got := w.Truncate(brazil+brazil+brazil+"abc", 4); got != brazil+brazil+brazil+"…" {
		t.Errorf("Truncate = %+q, want three flags and the ellipsis", got)
	}
}
