package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
)

type recorder struct{ writes []string }

func (r *recorder) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

func literal(r, g, b uint8) color.Color {
	return color.Color{Kind: color.Literal, RGBA: color.RGBA{R: r, G: g, B: b, A: 0xff}}
}

func put(b *buffer.Buffer, x, y int, s string, c buffer.Cell) {
	for i := range len(s) {
		c.Grapheme = s[i : i+1]
		b.Set(x+i, y, c)
	}
}

func TestOneCell(t *testing.T) {
	style := buffer.Cell{Fg: literal(0x7d, 0xd3, 0xfc)}
	blank, one, two := buffer.New(80, 24), buffer.New(80, 24), buffer.New(80, 24)
	put(one, 36, 11, "count: 1", style)
	put(two, 36, 11, "count: 2", style)
	rec := &recorder{}
	w := &Writer{Out: rec, Profile: color.TrueColor, Sync: true}
	if err := w.Diff(blank, one); err != nil {
		t.Fatal(err)
	}
	if err := w.Diff(one, two); err != nil {
		t.Fatal(err)
	}
	frame := rec.writes[1]
	t.Logf("one-cell change, 80x24, sync on: %d bytes %q", len(frame), frame)
	if want := "\x1b[?2026h\x1b[12;44H2\x1b[?2026l"; frame != want {
		t.Errorf("frame %q, want %q", frame, want)
	}
	if len(frame) >= 32 {
		t.Errorf("%d bytes, budget is under 32", len(frame))
	}
}

func TestOneWritePerFrame(t *testing.T) {
	prev, cur := buffer.New(20, 4), buffer.New(20, 4)
	put(cur, 0, 0, "top", buffer.Cell{Fg: literal(1, 2, 3)})
	put(cur, 5, 2, "middle", buffer.Cell{Attr: buffer.Bold})
	put(cur, 15, 3, "end", buffer.Cell{Bg: literal(4, 5, 6)})
	rec := &recorder{}
	w := &Writer{Out: rec, Profile: color.TrueColor, Sync: true}
	if err := w.Diff(prev, cur); err != nil {
		t.Fatal(err)
	}
	if len(rec.writes) != 1 {
		t.Fatalf("diff frame took %d writes, want 1", len(rec.writes))
	}
	if err := w.Static(cur); err != nil {
		t.Fatal(err)
	}
	if len(rec.writes) != 2 {
		t.Fatalf("static frame took %d writes, want 1", len(rec.writes)-1)
	}
	if err := w.Diff(cur, cur); err != nil {
		t.Fatal(err)
	}
	if len(rec.writes) != 2 {
		t.Errorf("unchanged frame wrote %q, want nothing", rec.writes[2:])
	}
}

func TestSync(t *testing.T) {
	prev, cur := buffer.New(10, 2), buffer.New(10, 2)
	put(cur, 2, 1, "x", buffer.Cell{})
	for _, sync := range []bool{true, false} {
		rec := &recorder{}
		w := &Writer{Out: rec, Profile: color.TrueColor, Sync: sync}
		if err := w.Diff(prev, cur); err != nil {
			t.Fatal(err)
		}
		if err := w.Static(cur); err != nil {
			t.Fatal(err)
		}
		frame, static := rec.writes[0], rec.writes[1]
		wrapped := strings.HasPrefix(frame, "\x1b[?2026h") && strings.HasSuffix(frame, "\x1b[?2026l")
		if wrapped != sync || strings.Count(frame, "2026") != 2*strings.Count(frame, "\x1b[?2026h") {
			t.Errorf("sync %v: diff frame %q", sync, frame)
		}
		if strings.Contains(static, "2026") {
			t.Errorf("sync %v: static frame %q sets a mode", sync, static)
		}
	}
}

func TestStatic(t *testing.T) {
	b := buffer.New(10, 2)
	put(b, 0, 0, "hello", buffer.Cell{})
	put(b, 0, 1, "world", buffer.Cell{})
	rec := &recorder{}
	if err := (&Writer{Out: rec, Profile: color.None}).Static(b); err != nil {
		t.Fatal(err)
	}
	if got := rec.writes[0]; got != "hello\nworld\n" {
		t.Errorf("profile none: %q", got)
	}

	put(b, 0, 0, "hello", buffer.Cell{Fg: literal(1, 2, 3)})
	b.Set(9, 1, buffer.Cell{Grapheme: " ", Bg: literal(4, 5, 6)})
	if err := (&Writer{Out: rec, Profile: color.TrueColor, Sync: true}).Static(b); err != nil {
		t.Fatal(err)
	}
	want := "\x1b[38;2;1;2;3mhello\x1b[0m\n" + "world    \x1b[48;2;4;5;6m \x1b[0m\n"
	if got := rec.writes[1]; got != want {
		t.Errorf("truecolor:\n got %q\nwant %q", got, want)
	}
}

func TestStaticAttributes(t *testing.T) {
	b := buffer.New(8, 3)
	put(b, 0, 0, "ab", buffer.Cell{Attr: buffer.Bold | buffer.Inverse, Fg: literal(1, 2, 3), Bg: literal(4, 5, 6)})
	put(b, 2, 0, "      ", buffer.Cell{Bg: literal(4, 5, 6)})
	put(b, 0, 1, "c ", buffer.Cell{Fg: literal(250, 2, 1), Bg: literal(4, 5, 6)})
	b.Set(2, 1, buffer.Cell{Grapheme: " ", Attr: buffer.Inverse})
	put(b, 0, 2, "d       ", buffer.Cell{Bg: literal(4, 5, 6), Attr: buffer.Inverse})
	for profile, want := range map[color.Profile]string{
		color.Attributes: "\x1b[1;7mab\x1b[0m\n" + "c \x1b[7m \x1b[0m\n" + "\x1b[7md       \x1b[0m\n",
		color.None:       "ab\n" + "c\n" + "d\n",
	} {
		var out bytes.Buffer
		if err := (&Writer{Out: &out, Profile: profile}).Static(b); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Errorf("profile %d:\n got %q\nwant %q", profile, out.String(), want)
		}
	}
}

func TestWideGlyph(t *testing.T) {
	prev, cur := buffer.New(10, 1), buffer.New(10, 1)
	cur.Set(0, 0, buffer.Cell{Grapheme: "世", Width: buffer.Wide})
	cur.Set(3, 0, buffer.Cell{Grapheme: "a"})
	rec := &recorder{}
	w := &Writer{Out: rec, Profile: color.None}
	if err := w.Diff(prev, cur); err != nil {
		t.Fatal(err)
	}
	if err := w.Static(cur); err != nil {
		t.Fatal(err)
	}
	if want := "\x1b[1;1H世\x1b[1Ca"; rec.writes[0] != want {
		t.Errorf("diff %q, want %q", rec.writes[0], want)
	}
	if want := "世 a\n"; rec.writes[1] != want {
		t.Errorf("static %q, want %q", rec.writes[1], want)
	}
}

func TestUnsafeGrapheme(t *testing.T) {
	b := buffer.New(6, 1)
	b.Set(0, 0, buffer.Cell{Grapheme: "\x1b[2J"})
	b.Set(1, 0, buffer.Cell{})
	b.Set(2, 0, buffer.Cell{Grapheme: "\u009b2J"})
	b.Set(3, 0, buffer.Cell{Grapheme: "x"})
	var out bytes.Buffer
	if err := (&Writer{Out: &out, Profile: color.None}).Static(b); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "   x\n" {
		t.Errorf("static %q, want %q", got, "   x\n")
	}
}

func TestAttrRemoval(t *testing.T) {
	blank, bold, plain := buffer.New(4, 1), buffer.New(4, 1), buffer.New(4, 1)
	put(bold, 0, 0, "a", buffer.Cell{Attr: buffer.Bold})
	put(plain, 0, 0, "b", buffer.Cell{})
	rec := &recorder{}
	w := &Writer{Out: rec, Profile: color.TrueColor}
	if err := w.Diff(blank, bold); err != nil {
		t.Fatal(err)
	}
	if err := w.Diff(bold, plain); err != nil {
		t.Fatal(err)
	}
	if want := "\x1b[1;1H\x1b[0;1ma"; rec.writes[0] != want {
		t.Errorf("first frame %q, want %q", rec.writes[0], want)
	}
	if want := "\x1b[1;1H\x1b[0mb"; rec.writes[1] != want {
		t.Errorf("second frame %q, want %q", rec.writes[1], want)
	}
}

func TestSameIndex(t *testing.T) {
	b := buffer.New(2, 1)
	b.Set(0, 0, buffer.Cell{Grapheme: "a", Fg: literal(255, 0, 0)})
	b.Set(1, 0, buffer.Cell{Grapheme: "b", Fg: literal(250, 2, 1)})
	for profile, want := range map[color.Profile]string{
		color.ANSI256: "\x1b[38;5;196mab\x1b[0m\n",
		color.ANSI16:  "\x1b[91mab\x1b[0m\n",
	} {
		var out bytes.Buffer
		if err := (&Writer{Out: &out, Profile: profile}).Static(b); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Errorf("profile %d: %q, want %q", profile, out.String(), want)
		}
	}
}

func TestClipboardPayloadIsInert(t *testing.T) {
	if got, want := string(Clipboard("héllo\tworld\n")), "\x1b]52;c;aMOpbGxvCXdvcmxkCg==\a"; got != want {
		t.Errorf("Clipboard: %q, want %q", got, want)
	}
	hostile := "a\x1b]52;c;ZXZpbA==\a\x1b[2J\x9b6n\x07b"
	seq := Clipboard(hostile)
	if bytes.Count(seq, []byte{0x1b}) != 1 || bytes.IndexByte(seq, 0x07) != len(seq)-1 {
		t.Errorf("Clipboard(%q) = %q: an escape or a bell of the text reached the terminal", hostile, seq)
	}
}
