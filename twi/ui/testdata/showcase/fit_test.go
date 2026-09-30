package main

import (
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func TestPagesFit(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{120, 34}, {150, 45}} {
		for _, name := range []string{"wave1", "tables", "breadcrumbs", "pagination", "items", "button-groups", "fields", "form", "overlays", "tabs", "wave3b"} {
			d := drive.New(func(rt *twi.Runtime) func() twi.Node {
				rt.SetTheme(zinc(theme.Light))
				body, ok := page(rt, name, "", "")
				if !ok {
					t.Fatalf("no page %q", name)
				}
				return func() twi.Node {
					return el("flex flex-row h-full overflow-x-auto", twi.Focusable(), twi.AutoFocus(), screen(twi.Class("flex-1"), body()))
				}
			}, drive.Size(size[0], size[1]), drive.Styles(sheet))
			before := d.Frame().Text()
			d.Press("right")
			if d.Frame().Text() != before {
				t.Errorf("%s at %dx%d runs past the right edge, it scrolls sideways:\n%s", name, size[0], size[1], before)
			}
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
