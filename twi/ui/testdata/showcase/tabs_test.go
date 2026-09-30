package main

import (
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/theme"
)

func TestTabsPage(t *testing.T) {
	d := driven(t, "tabs", theme.Light)
	defer func() {
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	has := func(s string) bool { return strings.Contains(d.Frame().Text(), s) }
	expect := func(what string, ok bool) {
		t.Helper()
		if !ok {
			t.Errorf("%s:\n%s", what, d.Frame().Text())
		}
	}
	at := func(anchor, s string) (int, int) {
		p := near(t, d.Frame(), anchor, s)
		return p.x, p.y
	}

	d.Press("tab")
	d.Press("right")
	expect("right on the focused tab list shows the password card", has("Change your password here.") && !has("Make changes to your account"))
	d.Press("tab")
	d.Press("tab")
	d.Press("tab")
	d.Press("enter")
	expect("tab reaches the card's button through the two inputs, enter presses it", has("Saved: Save password"))
	d.Click(at("Tabs", "Account"))
	expect("a click on Account shows the account card", has("Make changes to your account") && !has("Change your password"))
	t.Logf("tabs, Account clicked, zinc light, 150x45:\n%s", d.Frame().Text())

	d.Click(at("Select", "Select a fruit"))
	expect("a click opens the fruit select over the page", has("Fruits") && has("Pineapple"))
	d.Press("down")
	d.Press("enter")
	expect("down and enter choose Banana", has("Fruit: banana") && !has("Pineapple"))
	d.Press("enter")
	d.Press("g")
	t.Logf("select open, typeahead g on Grapes, zinc light, 150x45:\n%s", d.Frame().Text())
	d.Press("enter")
	expect("enter reopens it, g moves to Grapes, enter chooses it", has("Fruit: grapes") && !has("Pineapple"))
	d.Press("p")
	expect("p on the closed trigger chooses Pineapple", has("Fruit: pineapple"))
	d.Click(at("Time zone", "Coordinated Universal Time"))
	d.Click(at("Other", "India Standard Time"))
	expect("a click opens the time zone select and another chooses India", has("India Standard Time") && !has("Greenwich"))
}
