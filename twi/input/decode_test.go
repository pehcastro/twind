package input

import (
	"image"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func run(d *Decoder, writes ...string) []Event {
	var got []Event
	for _, w := range writes {
		got = append(got, d.Decode([]byte(w))...)
	}
	return got
}

func TestDecode(t *testing.T) {
	long := "\x1b[" + strings.Repeat("0", 5000) + "11~"
	cases := []struct {
		name   string
		writes []string
		want   []Event
		quiet  []Event
	}{
		{"rune", []string{"a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"wide rune split", []string{"\xe4\xb8", "\xad"}, []Event{KeyEvent{Rune: '中'}}, nil},
		{"ctrl c", []string{"\x03"}, []Event{KeyEvent{Rune: 'c', Modifiers: ModCtrl}}, nil},
		{"ctrl space", []string{"\x00"}, []Event{KeyEvent{Rune: ' ', Modifiers: ModCtrl}}, nil},
		{"enter", []string{"\r"}, []Event{KeyEvent{Key: KeyEnter}}, nil},
		{"tab", []string{"\t"}, []Event{KeyEvent{Key: KeyTab}}, nil},
		{"backspace del", []string{"\x7f"}, []Event{KeyEvent{Key: KeyBackspace}}, nil},
		{"backspace bs", []string{"\x08"}, []Event{KeyEvent{Key: KeyBackspace}}, nil},
		{"arrows csi and ss3", []string{"\x1b[A\x1bOB\x1b[C\x1bOD"}, []Event{
			KeyEvent{Key: KeyArrowUp}, KeyEvent{Key: KeyArrowDown}, KeyEvent{Key: KeyArrowRight}, KeyEvent{Key: KeyArrowLeft},
		}, nil},
		{"csi split", []string{"\x1b", "[", "A"}, []Event{KeyEvent{Key: KeyArrowUp}}, nil},
		{"home end", []string{"\x1b[H\x1b[F\x1b[1~\x1b[4~\x1bOH"}, []Event{
			KeyEvent{Key: KeyHome}, KeyEvent{Key: KeyEnd}, KeyEvent{Key: KeyHome}, KeyEvent{Key: KeyEnd}, KeyEvent{Key: KeyHome},
		}, nil},
		{"f1 f5 f12", []string{"\x1bOP\x1b[15~\x1b[24~"}, []Event{KeyEvent{Key: KeyF1}, KeyEvent{Key: KeyF5}, KeyEvent{Key: KeyF12}}, nil},
		{"alt x", []string{"\x1bx"}, []Event{KeyEvent{Rune: 'x', Modifiers: ModAlt}}, nil},
		{"alt x split", []string{"\x1b", "x"}, []Event{KeyEvent{Rune: 'x', Modifiers: ModAlt}}, nil},
		{"alt up", []string{"\x1b\x1b[A"}, []Event{KeyEvent{Key: KeyArrowUp, Modifiers: ModAlt}}, nil},
		{"shift tab", []string{"\x1b[Z"}, []Event{KeyEvent{Key: KeyTab, Modifiers: ModShift}}, nil},
		{"ctrl up", []string{"\x1b[1;5A"}, []Event{KeyEvent{Key: KeyArrowUp, Modifiers: ModCtrl}}, nil},
		{"shift delete", []string{"\x1b[3;2~"}, []Event{KeyEvent{Key: KeyDelete, Modifiers: ModShift}}, nil},
		{"kitty shift enter", []string{"\x1b[13;2u"}, []Event{KeyEvent{Key: KeyEnter, Modifiers: ModShift}}, nil},
		{"kitty ctrl c", []string{"\x1b[99;5u"}, []Event{KeyEvent{Rune: 'c', Modifiers: ModCtrl}}, nil},
		{"kitty release", []string{"\x1b[97;1:3u"}, []Event{KeyEvent{Rune: 'a', Release: true}}, nil},
		{"kitty repeat arrow", []string{"\x1b[1;1:2A"}, []Event{KeyEvent{Key: KeyArrowUp, Repeat: true}}, nil},
		{"kitty release arrow", []string{"\x1b[1;3:3D"}, []Event{KeyEvent{Key: KeyArrowLeft, Modifiers: ModAlt, Release: true}}, nil},
		{"kitty text", []string{"\x1b[97;2;65u"}, []Event{KeyEvent{Rune: 'A', Modifiers: ModShift}}, nil},
		{"kitty shifted key", []string{"\x1b[49:33;2u"}, []Event{KeyEvent{Rune: '!', Modifiers: ModShift}}, nil},
		{"kitty shift letter", []string{"\x1b[97;2u"}, []Event{KeyEvent{Rune: 'A', Modifiers: ModShift}}, nil},
		{"kitty super and meta", []string{"\x1b[107;9u\x1b[107;33u"}, []Event{
			KeyEvent{Rune: 'k', Modifiers: ModMeta}, KeyEvent{Rune: 'k', Modifiers: ModMeta},
		}, nil},
		{"kitty escape", []string{"\x1b[27u"}, []Event{KeyEvent{Key: KeyEscape}}, nil},
		{"kitty keypad", []string{"\x1b[57399u\x1b[57414u\x1b[57419;5u"}, []Event{
			KeyEvent{Rune: '0'}, KeyEvent{Key: KeyEnter}, KeyEvent{Key: KeyArrowUp, Modifiers: ModCtrl},
		}, nil},
		{"kitty caps lock ignored", []string{"\x1b[97;65u"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"modify other keys", []string{"\x1b[27;5;13~"}, []Event{KeyEvent{Key: KeyEnter, Modifiers: ModCtrl}}, nil},
		{"sgr press release", []string{"\x1b[<0;10;5M\x1b[<0;10;5m"}, []Event{
			MouseEvent{X: 9, Y: 4, Button: MouseLeft, Action: MousePress},
			MouseEvent{X: 9, Y: 4, Button: MouseLeft, Action: MouseRelease},
		}, nil},
		{"sgr wheel", []string{"\x1b[<64;3;4M\x1b[<65;3;4M"}, []Event{
			MouseEvent{X: 2, Y: 3, Button: MouseWheelUp, Action: MouseScroll},
			MouseEvent{X: 2, Y: 3, Button: MouseWheelDown, Action: MouseScroll},
		}, nil},
		{"sgr move and ctrl right", []string{"\x1b[<35;1;1M\x1b[<18;2;2M"}, []Event{
			MouseEvent{Button: MouseNone, Action: MouseMove},
			MouseEvent{X: 1, Y: 1, Button: MouseRight, Action: MousePress, Modifiers: ModCtrl},
		}, nil},
		{"sgr zero cell dropped", []string{"\x1b[<0;0;1Ma"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"paste with escape", []string{"\x1b[200~a\x1bb\x1b[201~"}, []Event{PasteEvent{Text: "a\x1bb"}}, nil},
		{"paste with sgr", []string{"\x1b[200~\x1b[31mred\x1b[A\x1b[201~x"}, []Event{
			PasteEvent{Text: "\x1b[31mred\x1b[A"}, KeyEvent{Rune: 'x'},
		}, nil},
		{"paste markers split", []string{"\x1b[20", "0~hi\x1b[20", "1~x"}, []Event{PasteEvent{Text: "hi"}, KeyEvent{Rune: 'x'}}, nil},
		{"paste open at quiet", []string{"\x1b[200~ab\x1b"}, nil, nil},
		{"paste end alone dropped", []string{"\x1b[201~a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"focus", []string{"\x1b[I\x1b[O"}, []Event{FocusEvent{Focused: true}, FocusEvent{Focused: false}}, nil},
		{"da1", []string{"\x1b[?62;22c"}, []Event{ReplyEvent{Kind: ReplyPrimaryAttributes, Params: []int{62, 22}}}, nil},
		{"da2", []string{"\x1b[>1;4000;0c"}, []Event{ReplyEvent{Kind: ReplySecondaryAttributes, Params: []int{1, 4000, 0}}}, nil},
		{"decrpm 2026", []string{"\x1b[?2026;2$y"}, []Event{ReplyEvent{Kind: ReplyMode, Params: []int{2026, 2}}}, nil},
		{"cursor position", []string{"\x1b[12;40R\x1b[?1;5;1R"}, []Event{
			ReplyEvent{Kind: ReplyCursorPosition, Params: []int{12, 40}},
			ReplyEvent{Kind: ReplyCursorPosition, Params: []int{1, 5, 1}},
		}, nil},
		{"kitty flags", []string{"\x1b[?1u"}, []Event{ReplyEvent{Kind: ReplyKeyboardFlags, Params: []int{1}}}, nil},
		{"lone escape", []string{"\x1b"}, nil, []Event{KeyEvent{Key: KeyEscape}}},
		{"alt escape", []string{"\x1b\x1b"}, nil, []Event{KeyEvent{Key: KeyEscape, Modifiers: ModAlt}}},
		{"alt bracket", []string{"\x1b["}, nil, []Event{KeyEvent{Rune: '[', Modifiers: ModAlt}}},
		{"alt O", []string{"\x1bO"}, nil, []Event{KeyEvent{Rune: 'O', Modifiers: ModAlt}}},
		{"incomplete csi dropped at quiet", []string{"\x1b[1;"}, nil, nil},
		{"incomplete rune dropped at quiet", []string{"\xe4\xb8"}, nil, nil},
		{"unknown csi", []string{"\x1b[5za"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"csi cut by escape", []string{"\x1b[1;2\x1b[A"}, []Event{KeyEvent{Key: KeyArrowUp}}, nil},
		{"invalid utf8", []string{"\xff\x9ba"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"osc st", []string{"\x1b]11;rgb:0000/0000/0000\x1b\\a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"osc bel", []string{"\x1b]11;rgb:0000/0000/0000\x07a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"huge parameter", []string{"\x1b[99999999999999999999999ua"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"surrogate code", []string{"\x1b[55296u\x1b[1114112ua"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"private use code", []string{"\x1b[57441;2ua"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"overlong sequence", []string{long + "a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
		{"overlong sequence split", []string{long[:4500], long[4500:] + "a"}, []Event{KeyEvent{Rune: 'a'}}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var d Decoder
			got := run(&d, c.writes...)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("writes %q\n got %#v\nwant %#v", c.writes, got, c.want)
			}
			if q := d.Quiet(); !reflect.DeepEqual(q, c.quiet) {
				t.Fatalf("quiet after %q\n got %#v\nwant %#v", c.writes, q, c.quiet)
			}
		})
	}
}

func TestDecodeInBand(t *testing.T) {
	key := KeyEvent{Rune: 'a'}
	for _, c := range []struct {
		write string
		want  []Event
	}{
		{"\x1b[48;34;120;680;1200t", []Event{ResizeEvent{Width: 120, Height: 34, Cell: image.Pt(10, 20)}}},
		{"\x1b[48;34;120;0;0t", []Event{ResizeEvent{Width: 120, Height: 34}}},
		{"\x1b[48;34;120;690;1205t", []Event{ResizeEvent{Width: 120, Height: 34, Cell: image.Pt(10, 20)}}},
		{"\x1b[48;34;120;680;50t", []Event{ResizeEvent{Width: 120, Height: 34}}},
		{"\x1b[48;0;0;680;1200ta", []Event{key}},
		{"\x1b[48;34;120ta", []Event{key}},
		{"\x1b[48;34;120;680;1200;1ta", []Event{key}},
		{"\x1b[6;20;10t", []Event{ReplyEvent{Kind: ReplyWindow, Params: []int{6, 20, 10}}}},
		{"\x1b[4;680;1200t", []Event{ReplyEvent{Kind: ReplyWindow, Params: []int{4, 680, 1200}}}},
		{"\x1b[?6;20;10ta", []Event{key}},
	} {
		var d Decoder
		if got := append(run(&d, c.write), d.Quiet()...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q\n got %#v\nwant %#v", c.write, got, c.want)
		}
	}
}

func TestDecodeMenu(t *testing.T) {
	menu := KeyEvent{Key: KeyMenu}
	shifted := KeyEvent{Key: KeyMenu, Modifiers: ModShift}
	for _, c := range []struct {
		writes []string
		want   []Event
	}{
		{[]string{"\x1b[29~"}, []Event{menu}},
		{[]string{"\x1b[2", "9~"}, []Event{menu}},
		{[]string{"\x1b[29;2~"}, []Event{shifted}},
		{[]string{"\x1b[57363u"}, []Event{menu}},
		{[]string{"\x1b[57363;2u"}, []Event{shifted}},
		{[]string{"\x1b[25~\x1b[26~\x1b[28~\x1b[30~"}, nil},
	} {
		var d Decoder
		if got := append(run(&d, c.writes...), d.Quiet()...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("writes %q\n got %#v\nwant %#v", c.writes, got, c.want)
		}
	}
	if got := string(Encode(menu)); got != "\x1b[29~" {
		t.Errorf("Encode(menu) = %q", got)
	}
}

func TestDecodeEncoded(t *testing.T) {
	keys := []KeyEvent{{Rune: 'a'}, {Rune: '中'}, {Rune: ' '}, {Rune: 'A'}, {Rune: '['}, {Rune: 'O'}}
	for k := KeyEnter; k <= KeyMenu; k++ {
		keys = append(keys, KeyEvent{Key: k})
	}
	var all []KeyEvent
	for _, k := range keys {
		for m := Modifiers(0); m <= ModShift|ModAlt|ModCtrl|ModMeta; m++ {
			for _, kind := range []KeyEvent{{}, {Repeat: true}, {Release: true}} {
				k.Modifiers, k.Repeat, k.Release = m, kind.Repeat, kind.Release
				all = append(all, k)
			}
		}
	}
	for _, k := range all {
		var d Decoder
		b := Encode(k)
		got := append(d.Decode(b), d.Quiet()...)
		want := k
		if k.Modifiers&ModShift != 0 {
			want.Rune = unicode.ToUpper(k.Rune)
		}
		if !reflect.DeepEqual(got, []Event{want}) {
			t.Errorf("%#v encoded %q decoded %#v", k, b, got)
		}
	}
}

func FuzzDecode(f *testing.F) {
	for _, s := range []string{"a\x1b[A", "\x1b[200~\x1b[31m\x1b[201~", "\x1b[<0;1;1M", "\x1b[97;1:3u", "\x1b]11;x\x1b\\", "\xe4\xb8\xad"} {
		f.Add([]byte(s), uint8(1))
	}
	f.Fuzz(func(t *testing.T, b []byte, cut uint8) {
		var whole, split Decoder
		want := append(whole.Decode(b), whole.Quiet()...)
		at := int(cut) % (len(b) + 1)
		got := append(run(&split, string(b[:at]), string(b[at:])), split.Quiet()...)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%q cut at %d\n split %#v\n whole %#v", b, at, got, want)
		}
	})
}
