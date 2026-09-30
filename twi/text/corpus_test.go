package text

import (
	"fmt"
	"hash/crc64"
	"os"
	"strconv"
	"strings"
	"testing"
)

const corpusFixture = "testdata/wrap-3a4e3b0.txt"

func wrapCorpus(t *testing.T) []string {
	data, err := os.ReadFile("testdata/GraphemeBreakTest.txt")
	if err != nil {
		t.Fatal(err)
	}
	corpus := []string{
		strings.Repeat("A well-known pangram: the quick brown fox jumps over the lazy dog, see https://example.com/fox 中文排版。 ", 20),
		"中文排版需要在任何两个汉字之间断行，但不能在句号。或逗号，之前断开「引号」内容（括号）也一样。",
		"日本語のテキストは、ひらがなとカタカナ、そして漢字が混在しています。ー長音符も！",
		"한국어 텍스트는 공백으로 단어를 구분합니다. 가나다라마바사아자차카타파하",
		"family " + family + " flags " + brazil + brazil + "\U0001F1EF skin \U0001F44D\U0001F3FD keycap 1\U0000FE0F\U000020E3 heart \U00002764\U0000FE0F text \U00002764\U0000FE0E",
		"\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467\U0000200D\U0001F466\U0001F3F3\U0000FE0F\U0000200D\U0001F308\U0001F9D1\U0001F3FD\U0000200D\U0001F4BB",
		"see https://example.com/docs/wrapping?query=a-b&c=d#frag and http://averyveryverylonghost.io/x/y/z (link) [ref] {obj}",
		"a package-level drive, well-known state-of-the-art non-breaking co-operate --flag -x -5 3.14 1,000 $5 5% 10°C €20",
		"  lead  and   gaps  with \U00000301 marks and \U0000200Bzero\U0000200Bwidth no\U000000A0break a\U00002060b",
		"wait   ! go (   b x  ) y  , z \U000000A0 q  \U00000301 r \U00000301\U00000301   s [  ]  {   }  x   /   y",
		"wait ! now ? ok ; yes : end . close ) open ( quote \" apostrophe ' tab\tnewline\r\nreturn\rdone",
		"e\U00000301e\U00000301e\U00000301 \U00000915\U0000094D\U00000937 \U00000E01\U00000E33 \U00001100\U00001161\U000011A8 invalid \xff\xfe bytes \xe4\xb8",
		"",
		" ",
		"\n\n",
		"supercalifragilisticexpialidocious-antidisestablishmentarianism/pneumonoultramicroscopicsilicovolcanoconiosis",
	}
	var chunk []string
	for line := range strings.Lines(string(data)) {
		rule, _, _ := strings.Cut(line, "#")
		var s strings.Builder
		for _, field := range strings.Fields(rule) {
			if code, err := strconv.ParseUint(field, 16, 32); err == nil {
				s.WriteRune(rune(code))
			}
		}
		if s.Len() == 0 {
			continue
		}
		chunk = append(chunk, s.String())
		if len(chunk) == 16 {
			corpus = append(corpus, strings.Join(chunk, " "), strings.Join(chunk, ""))
			chunk = chunk[:0]
		}
	}
	return append(corpus, strings.Join(chunk, " "))
}

func TestWrapCorpus(t *testing.T) {
	corpus := wrapCorpus(t)
	variants := []Widths{{}, {Flag: 1, ZWJ: 3, VS16: 1, Modifier: 1, Keycap: 1}}
	table := crc64.MakeTable(crc64.ECMA)
	var got strings.Builder
	for i, s := range corpus {
		var b []byte
		for _, v := range variants {
			b = fmt.Appendf(b, "%d\x02", v.MinContent(s))
			for width := range 81 {
				b = fmt.Appendf(b, "%q\x01", v.Wrap(s, width))
			}
		}
		fmt.Fprintf(&got, "item %d %016x\n", i, crc64.Checksum(b, table))
	}
	for width := range 81 {
		var b []byte
		for _, s := range corpus {
			for _, v := range variants {
				b = fmt.Appendf(b, "%q\x01", v.Wrap(s, width))
			}
		}
		fmt.Fprintf(&got, "width %d %016x\n", width, crc64.Checksum(b, table))
	}
	if os.Getenv("TWIND_WRITE_WRAP_FIXTURE") == "1" {
		if err := os.WriteFile(corpusFixture, []byte(got.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(corpusFixture)
	if err != nil {
		t.Fatal(err)
	}
	gotLines, wantLines := strings.Split(got.String(), "\n"), strings.Split(string(want), "\n")
	for i := range min(len(gotLines), len(wantLines)) {
		if gotLines[i] == wantLines[i] {
			continue
		}
		t.Errorf("%s, want %s", gotLines[i], wantLines[i])
		if i < len(corpus) {
			t.Errorf("item %d %+q wraps at 8 to %+q, min-content %d", i, corpus[i], Wrap(corpus[i], 8), MinContent(corpus[i]))
		}
		return
	}
	if len(gotLines) != len(wantLines) {
		t.Errorf("%d fixture lines, want %d", len(gotLines), len(wantLines))
	}
}
