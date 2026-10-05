package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/theme"
)

func TestAlertTextKeepsOffItsBorder(t *testing.T) {
	d := overlayDriver(t, 50, 16, func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
				Alert(Default, AlertTitle(twi.Text("Heads up")), AlertDescription(twi.Text("Themes switch at runtime."))),
				Alert(Destructive, AlertTitle(twi.Text("Payment failed")), AlertDescription(twi.Text("Check your card."))))
		}
	})
	lines := strings.Split(d.Frame().Text(), "\n")
	t.Logf("two alerts, 50x16:\n%s", d.Frame().Text())
	for title, last := range map[string]string{"Heads up": "Themes switch", "Payment failed": "Check your card"} {
		x, y, _ := at(d.Frame(), title)
		_, below, _ := at(d.Frame(), last)
		border := slices.Index([]rune(lines[y]), '│')
		if border < 0 || x-border != 3 {
			t.Errorf("%s: text at column %d, left border at %d, want two blank columns between", title, x, border)
		}
		if top := []rune(lines[y-2]); y < 2 || x >= len(top) || top[x] != '─' || strings.TrimSpace(string([]rune(lines[y-1])[border+1:x+len(title)])) != "" {
			t.Errorf("%s: want a blank row between the top border and the title", title)
		}
		if bottom := []rune(lines[below+2]); x >= len(bottom) || bottom[x] != '─' || strings.TrimSpace(string([]rune(lines[below+1])[border+1:x+len(last)])) != "" {
			t.Errorf("%s: want a blank row between the description and the bottom border", title)
		}
	}
}

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
	}, drive.Size(40, 24), drive.With(twi.Styles(sheet)))
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
