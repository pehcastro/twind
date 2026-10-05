package docsapp

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func TestRestoredPageAndThemeShow(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		page, theme       string
		wantPage, wantBar string
	}{
		{"button", "dream-light", "Button", "Theme: dream-light"},
		{"nosuch", "nosuch-dark", "Introduction", "Theme: twind-dark"},
	} {
		var saved [2]string
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			s, err := build(rt, Start{Page: "introduction", Theme: "twind-dark"})
			if err != nil {
				t.Fatal(err)
			}
			s.recallPage(tc.page)
			s.recallTheme(tc.theme)
			saved = [2]string{s.pageMemo(), s.themeMemo()}
			return s.view
		}, drive.Size(120, 36), drive.With(twi.Styles(sheet)))
		text := d.Frame().Text()
		if !strings.Contains(text, tc.wantPage) || !strings.Contains(text, tc.wantBar) {
			t.Errorf("restored %s and %s: want %q and %q on screen:\n%s", tc.page, tc.theme, tc.wantPage, tc.wantBar, text)
		}
		if tc.page == "button" && saved != [2]string{"button", "dream-light"} {
			t.Errorf("memo %v, want the restored page and theme", saved)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
