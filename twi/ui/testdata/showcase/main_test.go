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
	for name, want := range map[string][]string{
		"wave1":         {"Destructive", "Login to your account", "Sign Up", "Unable to process your payment.", "No Projects Yet", "Ctrl", "CN", "LR"},
		"tables":        {"Invoice", "INV007", "Bank Transfer", "$2,500.00", "A list of your recent invoices."},
		"breadcrumbs":   {"Home", "›", "…", "Components", "/", "Breadcrumb"},
		"pagination":    {"‹ Previous", "Next ›", "…"},
		"items":         {"Basic Item", "Your profile has been verified.", "Muted Variant", "evilrabbit@vercel.com"},
		"button-groups": {"Archive", "Report", "Snooze", "Copy", "Paste", "https://"},
		"fields":        {"Payment Method", "Card Number", "Billing Address", "Submit", "Enter a valid email address.", "Or continue with"},
	} {
		body, ok := page(name)
		if !ok {
			t.Fatalf("no page %q", name)
		}
		for _, scheme := range []theme.Scheme{theme.Light, theme.Dark} {
			d := drive.New(func(rt *twi.Runtime) func() twi.Node {
				for _, th := range theme.Builtin() {
					if th.Name == "zinc" && th.Scheme == scheme {
						rt.SetTheme(th)
					}
				}
				return func() twi.Node { return screen(twi.Class(""), body) }
			}, drive.Size(150, 45), drive.Styles(sheet))
			frame := d.Frame().Text()
			if err := d.Close(); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s, scheme %d:\n%s", name, scheme, frame)
			for _, s := range want {
				if !strings.Contains(frame, s) {
					t.Errorf("%s: no %q in the frame", name, s)
				}
			}
		}
	}
	if _, ok := page("nothing"); ok {
		t.Error("a page that does not exist was found")
	}
}
