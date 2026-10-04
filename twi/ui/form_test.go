package ui

import (
	"maps"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
)

type signup struct {
	*drive.Driver
	form            *Form
	email, password *Input
	terms           *Checkbox
	sent            []FormValues
}

func signupDriver(t *testing.T, email string) *signup {
	t.Helper()
	s := &signup{}
	s.Driver = overlayDriver(t, 48, 20, func(rt *twi.Runtime) func() twi.Node {
		s.form, s.email, s.password, s.terms = NewForm(rt), NewInput(rt), NewInput(rt), NewCheckbox(rt)
		s.email.Insert(email)
		required := func(name string) func(string) string {
			return func(v string) string {
				if v == "" {
					return name + " is required"
				}
				return ""
			}
		}
		s.form.Input("email", s.email, required("Email"), func(v string) string {
			if !strings.Contains(v, "@") {
				return "Enter a valid email"
			}
			return ""
		})
		s.form.Input("password", s.password, required("Password"), func(v string) string {
			if len(v) < 8 {
				return "Use at least 8 characters"
			}
			return ""
		})
		s.form.Checkbox("terms", s.terms, func(on bool) string {
			if !on {
				return "Accept the terms"
			}
			return ""
		})
		s.form.OnSubmit = func(v FormValues) { s.sent = append(s.sent, v) }
		return func() twi.Node {
			f := s.form
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				f.Item("email", f.Label("email", twi.Text("Email")), f.Control("email", twi.AutoFocus())),
				f.Item("password", f.Label("password", twi.Text("Password")), f.Control("password")),
				f.Item("terms", Field(Horizontal, f.Control("terms"), f.Label("terms", twi.Text("Accept terms")))),
			)
		}
	})
	return s
}

func (s *signup) shows(t *testing.T, want map[string]bool) {
	t.Helper()
	frame := s.Frame().Text()
	for message, shown := range want {
		if strings.Contains(frame, message) != shown {
			t.Fatalf("message %q shown is %v, want %v:\n%s", message, !shown, shown, frame)
		}
	}
}

func TestFormBlurShowsTheFirstFailingRule(t *testing.T) {
	s := signupDriver(t, "")
	s.shows(t, map[string]bool{"Email is required": false})
	hit(s.Driver, "tab")
	s.shows(t, map[string]bool{"Email is required": true, "Enter a valid email": false, "Password is required": false})
	if !s.email.Invalid || s.password.Invalid {
		t.Fatalf("after blur email invalid %v, password invalid %v; want only email", s.email.Invalid, s.password.Invalid)
	}
	hit(s.Driver, "shift+tab type:bob tab")
	s.shows(t, map[string]bool{"Email is required": false, "Enter a valid email": true})
	hit(s.Driver, "shift+tab type:@x.io tab")
	s.shows(t, map[string]bool{"Enter a valid email": false})
	if s.email.Invalid {
		t.Fatalf("a valid email kept the destructive ring:\n%s", s.Frame().Text())
	}
}

func TestFormSubmitWithErrorsFocusesTheFirstInvalidField(t *testing.T) {
	s := signupDriver(t, "")
	hit(s.Driver, "type:bob@x.io enter")
	if len(s.sent) != 0 {
		t.Fatalf("an invalid form submitted %v", s.sent)
	}
	s.shows(t, map[string]bool{"Password is required": true, "Accept the terms": true, "Enter a valid email": false})
	hit(s.Driver, "type:z")
	if s.password.Value() != "z" || s.email.Value() != "bob@x.io" {
		t.Fatalf("after submit the typed z went to email %q, password %q; want the password focused:\n%s", s.email.Value(), s.password.Value(), s.Frame().Text())
	}
}

func TestFormValidSubmitSendsTheValuesOnce(t *testing.T) {
	s := signupDriver(t, "")
	hit(s.Driver, "type:bob@x.io tab type:hunter22 tab space shift+tab enter")
	t.Logf("submitted:\n%s", s.Frame().Text())
	want := FormValues{Text: map[string]string{"email": "bob@x.io", "password": "hunter22"}, Checked: map[string]bool{"terms": true}}
	if len(s.sent) != 1 || !maps.Equal(s.sent[0].Text, want.Text) || !maps.Equal(s.sent[0].Checked, want.Checked) {
		t.Fatalf("submitted %v, want once %v", s.sent, want)
	}
}

func TestFormSelectIgnoresTheBlurOfItsOwnPopover(t *testing.T) {
	var (
		f    *Form
		plan *Select
	)
	d := overlayDriver(t, 48, 16, func(rt *twi.Runtime) func() twi.Node {
		f, plan = NewForm(rt), NewSelect(rt)
		plan.Placeholder = "Choose"
		f.Select("plan", plan, func(v string) string {
			if v == "" {
				return "Pick a plan first"
			}
			return ""
		})
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				f.Item("plan", f.Label("plan", twi.Text("Plan")), plan.Node(f.Control("plan", twi.AutoFocus()), plan.Content(plan.Item("free", "Free"), plan.Item("pro", "Pro")))),
				twi.Element(twi.Focusable(), twi.Text("after")),
			)
		}
	})
	hit(d, "enter")
	if !plan.Open || plan.Invalid {
		t.Fatalf("opening the select (open %v) marked it invalid:\n%s", plan.Open, d.Frame().Text())
	}
	hit(d, "escape tab")
	if !strings.Contains(d.Frame().Text(), "Pick a plan first") {
		t.Fatalf("leaving an empty select showed no message:\n%s", d.Frame().Text())
	}
}

func TestFormResetRestoresValuesAndClearsMessages(t *testing.T) {
	s := signupDriver(t, "me@x.io")
	hit(s.Driver, "tab type:abc tab space")
	s.shows(t, map[string]bool{"Use at least 8 characters": true})
	s.form.Reset()
	s.Advance(settleTime)
	s.shows(t, map[string]bool{"Use at least 8 characters": false, "Accept the terms": false})
	if s.email.Value() != "me@x.io" || s.password.Value() != "" || s.terms.Checked || s.password.Invalid {
		t.Fatalf("reset left email %q, password %q, terms %v, password invalid %v", s.email.Value(), s.password.Value(), s.terms.Checked, s.password.Invalid)
	}
}
