package main

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func TestDrivenFrame(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []theme.Scheme{theme.Light, theme.Dark} {
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			for _, th := range theme.Builtin() {
				if th.Name == "zinc" && th.Scheme == scheme {
					rt.SetTheme(th)
				}
			}
			return func() twi.Node { return showcase(twi.Class("")) }
		}, drive.Size(150, 45), drive.Styles(sheet))
		frame := d.Frame().Text()
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		t.Logf("scheme %d:\n%s", scheme, frame)
		for _, s := range []string{"Destructive", "Login to your account", "Sign Up", "Unable to process your payment.", "No Projects Yet", "Ctrl", "CN", "LR"} {
			if !strings.Contains(frame, s) {
				t.Errorf("no %q in the frame", s)
			}
		}
	}
}
