package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/theme"
)

func TestCardActionKeepsItsColumnDriven(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		header := func(action ...twi.NodeOption) twi.Node {
			return Card(twi.Class("w-24"), CardHeader(append([]twi.NodeOption{CardTitle(twi.Text("Login to your account")), CardDescription(twi.Text("Enter your email"))}, action...)...))
		}
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col items-start gap-1 p-1 w-full h-full bg-background text-foreground"),
				header(CardAction(twi.Text("Sign up"))),
				header(),
			)
		}
	}, drive.Size(40, 24), drive.Styles(sheet))
	f := d.Frame()
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("40x24:\n%s", f.Text())
	var inner [][]string
	for _, l := range strings.Split(f.Text(), "\n") {
		row := []rune(l)
		left := slices.Index(row, '│')
		if left < 0 || left+23 >= len(row) || row[left+23] != '│' {
			continue
		}
		inner = append(inner, strings.Fields(string(row[left+1:left+23])))
	}
	starts := slices.IndexFunc(inner, func(w []string) bool { return slices.Contains(w, "Login") })
	if starts < 0 {
		t.Fatalf("no card title row in the frame")
	}
	want := [][]string{{"Login", "to", "Sign", "up"}, {"your"}, {"account"}, {"Enter", "your"}, {"email"}}
	if got := inner[starts:min(len(inner), starts+len(want))]; !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("card with an action: rows %q, want %q: the title wraps in its own column and Sign up sits right of its first row", got, want)
	}
	again := slices.IndexFunc(inner[starts+1:], func(w []string) bool { return slices.Contains(w, "Login") })
	if again < 0 {
		t.Fatalf("no second card title row in the frame")
	}
	alone := [][]string{{"Login", "to", "your"}, {"account"}, {"Enter", "your", "email"}}
	if got := inner[starts+1+again : min(len(inner), starts+1+again+len(alone))]; !slices.EqualFunc(got, alone, slices.Equal) {
		t.Errorf("card without an action: rows %q, want %q: no column is kept for a missing action", got, alone)
	}
}
