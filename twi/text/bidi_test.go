package text

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

func TestBidiCharacterTest(t *testing.T) {
	data, err := os.ReadFile("testdata/BidiCharacterTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# BidiCharacterTest-"+UnicodeVersion+".txt") {
		t.Fatalf("BidiCharacterTest.txt is not Unicode %s", UnicodeVersion)
	}
	run, levelsFailed, orderFailed := 0, 0, 0
	for line := range strings.Lines(string(data)) {
		fields := strings.Split(strings.TrimSpace(line), ";")
		if len(fields) != 5 || strings.HasPrefix(line, "#") {
			continue
		}
		var s strings.Builder
		for code := range strings.FieldsSeq(fields[0]) {
			r, err := strconv.ParseUint(code, 16, 32)
			if err != nil {
				t.Fatal(err)
			}
			s.WriteRune(rune(r))
		}
		var p paragraph
		p.resolve(s.String(), Direction(fields[1][0]-'0'))
		ks := make([]int, len(p.runes))
		for i := range ks {
			ks[i] = i
		}
		levels := make([]uint8, len(ks))
		p.line(ks, levels)
		var got []string
		var units []unit
		for i, level := range levels {
			if p.classes[i] == bBN {
				got = append(got, "x")
				continue
			}
			got = append(got, strconv.Itoa(int(level)))
			units = append(units, unit{from: i, level: level})
		}
		run++
		if want := strconv.Itoa(int(p.level)) + ";" + strings.Join(got, " "); want != fields[2]+";"+fields[3] {
			levelsFailed++
			if levelsFailed <= 20 {
				t.Errorf("%s: levels %s, want %s;%s", strings.TrimSpace(line), want, fields[2], fields[3])
			}
			continue
		}
		reorder(units)
		order := make([]string, len(units))
		for i, u := range units {
			order[i] = strconv.Itoa(u.from)
		}
		if !slices.Equal(order, strings.Fields(fields[4])) {
			orderFailed++
			if orderFailed <= 20 {
				t.Errorf("%s: order %s", strings.TrimSpace(line), strings.Join(order, " "))
			}
		}
	}
	t.Logf("BidiCharacterTest-%s: %d lines run, %d failed levels, %d failed order", UnicodeVersion, run, levelsFailed, orderFailed)
	if run == 0 {
		t.Fatal("no lines run")
	}
}

func TestBidiLeads(t *testing.T) {
	detects := func(s string) bool {
		out := wrapper{s: s, width: konst.LeapMin * konst.LeapMin}
		return Wrapping{}.segments(s, 0, len(s), out.place)
	}
	found := 0
	for r := range utf8.MaxRune + 1 {
		switch bidiClass(bidiOf(r) & konst.BidiClassMask) {
		case bR, bAL, bAN, bRLE, bRLO, bRLI:
			found++
			for _, s := range []string{string(r), "a " + string(r), "a" + string(r), "é" + string(r), "the quick brown fox jumps over 中文" + string(r) + " the lazy dog", "ൎ" + string(r)} {
				if !detects(s) {
					t.Errorf("right-to-left %U missed in %+q", r, s)
				}
			}
		}
	}
	ltr := "a well-known pangram 中文排版。 ∑ € ր ﬀ \U00010000 é"
	if found == 0 || detects(ltr) {
		t.Errorf("%d right-to-left runes; %+q taken as right-to-left", found, ltr)
	}
}

func TestBidiWrap(t *testing.T) {
	cases := []struct {
		name  string
		b     Wrapping
		s     string
		width int
		want  []string
	}{
		{"english in rtl", Wrapping{Dir: DirRTL}, "hello world", 80, []string{"hello world"}},
		{"hebrew in ltr", Wrapping{}, "a שלום b", 80, []string{"a םולש b"}},
		{"logical wrap", Wrapping{Dir: DirAuto}, "שלום עולם טוב", 9, []string{"םלוע םולש", "בוט"}},
		{"mixed wrap", Wrapping{}, "abc שלום עולם def", 8, []string{"abc םולש", "םלוע def"}},
		{"mirrored", Wrapping{Dir: DirRTL}, "(שלום)", 80, []string{"(םולש)"}},
		{"not mirrored in ltr", Wrapping{}, "a (שלום) b", 80, []string{"a (םולש) b"}},
		{"digits", Wrapping{Dir: DirAuto}, "עמוד 12", 80, []string{"12 דומע"}},
		{"marks", Wrapping{}, "שָׁלוֹם", 80, []string{"םוֹלשָׁ"}},
		{"zwj", Wrapping{Dir: DirRTL}, "שלום " + family, 80, []string{family + " םולש"}},
		{"collapsed", Wrapping{}, "שלום   עולם", 80, []string{"םלוע םולש"}},
		{"preserved", Wrapping{Space: SpacePreserve}, "שלום  עולם", 80, []string{"םלוע  םולש"}},
		{"trailing spaces stay right", Wrapping{Space: SpacePreserve}, "שלום  ", 80, []string{"םולש  "}},
		{"arabic", Wrapping{}, "سلام", 80, []string{"مالس"}},
		{"paragraphs", Wrapping{Dir: DirAuto}, "שלום\nhello", 80, []string{"םולש", "hello"}},
	}
	for _, c := range cases {
		if got := c.b.Wrap(c.s, c.width); !slices.Equal(got, c.want) {
			t.Errorf("%s: Wrap(%q, %d) = %q, want %q", c.name, c.s, c.width, got, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("unknown direction did not panic")
		}
	}()
	Wrapping{Dir: DirAuto + 1}.Wrap("a", 1)
}
