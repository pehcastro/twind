package app

import (
	"strings"
	"testing"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

func TestPointer(t *testing.T) {
	d := open(t)
	at := func(s string) (int, int) { return spot(t, d, s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s, status %q:\n%s", what, status(d), d.Frame().Text())
		}
	}
	token := func(name string, tk theme.Token) color.RGBA { return theme.Builtin()[themeIndex(name)].Tokens[tk].RGBA }
	d.Click(at("counter"))
	expect("a click on the counter tab opens its page", strings.HasSuffix(status(d), "focus counter counter"))
	d.Click(at("+ increment"))
	d.Click(at("+ increment"))
	expect("two clicks on increment count to two", strings.Contains(d.Frame().Text(), "▀▀█"))
	d.Move(at("zinc-dark"))
	expect("hovering the theme button shows its tooltip", strings.Contains(d.Frame().Text(), "t or a click opens the picker"))
	d.Click(at("zinc-dark"))
	d.Advance(300 * time.Millisecond)
	expect("a click on the theme button opens the picker", strings.Contains(d.Frame().Text(), "Enter keeps") && !strings.Contains(d.Frame().Text(), "opens the picker"))
	d.Move(at("slate-light"))
	d.Click(at("slate-light"))
	expect("a click on a row applies its theme", strings.Contains(status(d), "slate-light") && !strings.Contains(d.Frame().Text(), "Enter keeps"))
	d.Click(at("slate-light"))
	d.Move(at("rose-light"))
	expect("hovering a row previews its theme", cellAt(t, d, "Theme").Bg.RGBA == token("rose-light", theme.Popover))
	d.Click(1, 1)
	expect("a click outside the picker closes it and brings slate-light back", !strings.Contains(d.Frame().Text(), "Enter keeps") && cellAt(t, d, "fullscreen").Bg.RGBA == token("slate-light", theme.Background))
	d.Click(at("⌕ ui"))
	d.Type("tool")
	d.Click(at("tooltip"))
	expect("a click in the palette opens the component's page", strings.HasSuffix(status(d), " tooltip"))
}
