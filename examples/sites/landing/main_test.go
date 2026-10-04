package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

const width, height = 120, 40

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run go generate", konst.GeneratedFile)
	}
}

func background(s site) string {
	c := s.theme().Tokens[theme.Background].RGBA
	return fmt.Sprintf("48;2;%d;%d;%d", c.R, c.G, c.B)
}

func TestTour(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("testdata", "tour.twd"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	app := func(rt *twi.Runtime) func() twi.Node { return landing(rt, platform) }
	if err := drive.RunScript(strings.NewReader(string(script)), app, out, drive.Styles(sheet), drive.Size(width, height)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		frame          string
		look           site
		shown, missing []string
	}{
		{"platform", platform, []string{"▲ Quarry", "Ship the whole stack", "Start deploying", "npx quarry deploy", "99.99%"}, []string{"Plainsheet", "atelier nine"}},
		{"platform-features", platform, []string{"Preview every branch", "Secrets per stage", "quarry.config.ts", "\"vite\""}, []string{"Ship the whole stack"}},
		{"platform-footer", platform, []string{"Create a free project", "© 2026 ▲ Quarry"}, nil},
		{"product", product, []string{"◧ Plainsheet", "Plan the work. Skip the busywork.", "app.plainsheet.example", "Billing migration", "Blocked", "Simple plans that grow with you"}, []string{"Quarry"}},
		{"product-pricing", product, []string{"Starter", "Team", "Business", "Most popular", "$12", "Choose Team"}, nil},
		{"product-faq", product, []string{"Frequently asked questions", "How does the free trial work?", "every Team feature for 14 days", "Where is my data stored?"}, []string{"Drop in a CSV"}},
		{"studio", studio, []string{"atelier nine", "W E   D E S I G N   Q U I E T", "Start a project"}, []string{"Plainsheet"}},
		{"studio-work", studio, []string{"Harbor Ledger", "Open Orchard", "L E T   U S   T A L K", "© 2026 atelier nine"}, nil},
		{"platform-again", platform, []string{"Ship the whole stack"}, []string{"atelier nine"}},
	} {
		read := func(ext string) string {
			b, err := os.ReadFile(filepath.Join(out, c.frame+ext))
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
		text := read(".txt")
		t.Logf("frame %s:\n%s", c.frame, text)
		for _, s := range append([]string{"Platform", "Product", "Studio"}, c.shown...) {
			if !strings.Contains(text, s) {
				t.Errorf("frame %s: no %q", c.frame, s)
			}
		}
		for _, s := range c.missing {
			if strings.Contains(text, s) {
				t.Errorf("frame %s: %q shown, want it gone", c.frame, s)
			}
		}
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		if len(lines) != height {
			t.Errorf("frame %s has %d rows, want %d", c.frame, len(lines), height)
		}
		for i, l := range lines {
			if w := len([]rune(strings.TrimRight(l, " "))); w > width {
				t.Errorf("frame %s: row %d is %d cells wide", c.frame, i, w)
			}
		}
		ansi := read(".ansi")
		if !strings.Contains(ansi, background(c.look)) {
			t.Errorf("frame %s: no %s background", c.frame, c.look)
		}
		for _, s := range sites() {
			if s.theme().Scheme != c.look.theme().Scheme && strings.Contains(ansi, background(s)) {
				t.Errorf("frame %s: %s background shown, want only the %s scheme", c.frame, s, c.look)
			}
		}
	}
}
