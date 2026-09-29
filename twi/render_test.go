package twi_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/testdata/hello"
	"github.com/twind-dev/twind/twi/text"
)

type cell struct {
	glyph  string
	fg, bg string
}

func decode(t *testing.T, out string) [][]cell {
	t.Helper()
	sgr := regexp.MustCompile(`^\x1b\[([0-9;]*)m`)
	var rows [][]cell
	var row []cell
	fg, bg := "", ""
	for out != "" {
		if m := sgr.FindStringSubmatch(out); m != nil {
			params := strings.Split(m[1], ";")
			for i := 0; i < len(params); i++ {
				switch params[i] {
				case "0", "":
					fg, bg = "", ""
				case "38", "48":
					rgb := strings.Join(params[i+2:i+5], ",")
					if params[i] == "38" {
						fg = rgb
					} else {
						bg = rgb
					}
					i += 4
				case "39":
					fg = ""
				case "49":
					bg = ""
				}
			}
			out = out[len(m[0]):]
			continue
		}
		if out[0] == 0x1b {
			t.Fatalf("escape that is not SGR: %q", out[:min(len(out), 12)])
		}
		if out[0] == '\n' {
			rows, row, out = append(rows, row), nil, out[1:]
			continue
		}
		cluster := ""
		for g := range text.Graphemes(out) {
			cluster = g
			break
		}
		for range text.Width(cluster) {
			row = append(row, cell{cluster, fg, bg})
		}
		out = out[len(cluster):]
	}
	if row != nil {
		t.Fatalf("output does not end in a newline; last row %v", row)
	}
	return rows
}

func sheet(t *testing.T) twi.RenderOption {
	t.Helper()
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	return twi.Styles(s)
}

func TestRenderHelloTruecolor(t *testing.T) {
	out := twi.RenderString(hello.App(), sheet(t), twi.ColorProfile(color.TrueColor))
	lines := strings.SplitAfter(out, "\n")
	for i, line := range lines[:len(lines)-1] {
		if !strings.HasSuffix(line, "\x1b[0m\n") {
			t.Errorf("line %d does not end with ESC[0m then newline: %q", i, line[max(len(line)-12, 0):])
		}
	}
	rows := decode(t, out)
	if len(rows) != 18 {
		t.Fatalf("%d rows, want 18", len(rows))
	}
	for y, row := range rows {
		if len(row) != 80 {
			t.Errorf("row %d is %d cells, want 80", y, len(row))
		}
		for x, c := range row {
			if c.bg != "9,9,11" {
				t.Errorf("cell %d,%d background %q, want zinc-950 9,9,11", x, y, c.bg)
			}
			if c.glyph != " " && c.fg != "244,244,245" {
				t.Errorf("cell %d,%d %q foreground %q, want zinc-100 244,244,245", x, y, c.glyph, c.fg)
			}
		}
	}
	at := func(x, y int, want string) {
		t.Helper()
		var got strings.Builder
		for _, c := range rows[y][x : x+len([]rune(want))] {
			got.WriteString(c.glyph)
		}
		if got.String() != want {
			t.Errorf("at %d,%d got %q, want %q", x, y, got.String(), want)
		}
	}
	at(4, 4, "Hello Twind")
	at(4, 7, "╭─")
	at(74, 7, "─╮")
	at(4, 8, "│")
	at(75, 12, "│")
	at(7, 10, "Terminal DOM")
	at(4, 13, "╰─")
	at(74, 13, "─╯")
}

func TestRenderDefaultsOnAPipe(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("FORCE_COLOR", "")
	out := twi.RenderString(hello.App(), sheet(t))
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("a writer that is not a terminal got escapes: %q", out[:min(len(out), 40)])
	}
	rows := decode(t, out)
	if len(rows) != 18 || len(rows[7]) != 80 || rows[7][75].glyph != "╮" {
		t.Errorf("default width is not 80: %d rows, row 7 is %d cells", len(rows), len(rows[7]))
	}
	t.Setenv("FORCE_COLOR", "3")
	if out := twi.RenderString(hello.App(), sheet(t)); !strings.Contains(out, "\x1b[") {
		t.Error("FORCE_COLOR=3 on a pipe gave no colour")
	}
}

func TestRenderNarrowWrapsLikeLayout(t *testing.T) {
	rows := decode(t, twi.RenderString(hello.App(), sheet(t), twi.Width(20)))
	if len(rows) != 19 {
		t.Fatalf("%d rows, want 19: the card grows one row for the wrapped text", len(rows))
	}
	for y, want := range map[int]string{10: "│  Termin  │", 11: "│  al DOM  │", 14: "╰──────────╯"} {
		var got strings.Builder
		for _, c := range rows[y][4:16] {
			got.WriteString(c.glyph)
		}
		if got.String() != want {
			t.Errorf("row %d got %q, want %q", y, got.String(), want)
		}
	}
}

func TestRenderBorderNoneTakesNoSpace(t *testing.T) {
	rows := decode(t, twi.RenderString(hello.Borderless(), sheet(t)))
	if len(rows) != 5 || rows[2][2].glyph != "x" {
		t.Errorf("border-none reserved space or drew a border: %d rows, %v", len(rows), rows)
	}
}

func TestRenderUnsupportedDisplayFails(t *testing.T) {
	err := twi.Render(&strings.Builder{}, hello.Grid(), sheet(t))
	if err == nil || !strings.Contains(err.Error(), "display") {
		t.Errorf("display grid rendered without an error: %v", err)
	}
}

func TestRenderHostileText(t *testing.T) {
	corpus, err := os.ReadFile("text/testdata/hostile.txt")
	if err != nil {
		t.Fatal(err)
	}
	sgr, cases := regexp.MustCompile(`\x1b\[[0-9;]*m`), 0
	for line := range strings.Lines(string(corpus)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		var input strings.Builder
		for _, field := range fields[1 : len(fields)-1] {
			s, err := strconv.Unquote(field)
			if err != nil {
				t.Fatalf("%s: %v", fields[0], err)
			}
			input.WriteString(s)
		}
		for _, p := range []color.Profile{color.None, color.TrueColor} {
			out := twi.RenderString(twi.Text(input.String()), twi.ColorProfile(p))
			rest := sgr.ReplaceAllString(out, "")
			if i := strings.IndexFunc(rest, func(r rune) bool { return r == 0x1b || r == 0x9b || r == 0x9d || r == 0x7 }); i >= 0 {
				t.Errorf("%s, profile %d: control %q reached the output: %q", fields[0], p, rest[i], out)
			}
			if p == color.None && strings.ContainsRune(out, 0x1b) {
				t.Errorf("%s: escape with no colour profile: %q", fields[0], out)
			}
		}
		cases++
	}
	if cases == 0 {
		t.Fatal("empty hostile corpus")
	}
	t.Logf("%d hostile cases", cases)
}

func TestRenderStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale("testdata/hello", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twi/testdata/hello IR is stale: run go generate there")
	}
}
