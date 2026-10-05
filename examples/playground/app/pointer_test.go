package app

import (
	"strings"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
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
	token := func(name string, tk theme.Token) color.RGBA { return builtinTheme(t, name).Tokens[tk].RGBA }
	d.Click(at("counter"))
	expect("a click on the counter tab opens its page", strings.HasSuffix(status(d), "focus counter counter"))
	d.Click(at("+ increment"))
	d.Click(at("+ increment"))
	expect("two clicks on increment count to two", strings.Contains(d.Frame().Text(), "▀▀█"))
	d.Move(at("twind-dark"))
	expect("hovering the theme button shows its tooltip", strings.Contains(d.Frame().Text(), "t or a click opens the picker"))
	d.Click(at("twind-dark"))
	d.Advance(300 * time.Millisecond)
	expect("a click on the theme button opens the picker", strings.Contains(d.Frame().Text(), "Enter keeps") && !strings.Contains(d.Frame().Text(), "opens the picker"))
	d.Move(pickerSpot(t, d, "dew"))
	d.Click(pickerSpot(t, d, "dew"))
	expect("a click on a row applies its theme", strings.Contains(status(d), "dew-dark") && !strings.Contains(d.Frame().Text(), "Enter keeps"))
	d.Click(at("dew-dark"))
	d.Move(pickerSpot(t, d, "cloud"))
	d.Advance(300 * time.Millisecond)
	expect("hovering a row previews its theme", cellAt(t, d, "Theme").Bg.RGBA == token("cloud-dark", theme.Popover))
	d.Click(1, 1)
	expect("a click outside the picker closes it and brings dew-dark back", !strings.Contains(d.Frame().Text(), "Enter keeps") && cellAt(t, d, "fullscreen").Bg.RGBA == token("dew-dark", theme.Background))
	d.Click(at("⌕ ui"))
	d.Type("tool")
	d.Click(at("tooltip"))
	expect("a click in the palette opens the component's page", strings.HasSuffix(status(d), " tooltip"))
}
