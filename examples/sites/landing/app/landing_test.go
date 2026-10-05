package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/tailwind"
	"github.com/pehcastro/twind/twi/theme"
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
	if err := drive.RunScript(strings.NewReader(string(script)), app, out, drive.With(twi.Styles(sheet)), drive.Size(width, height)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		frame          string
		look           site
		shown, missing []string
	}{
		{"platform", platform, []string{"Quarry", "⊙ quarry.example", "█▀▀▀▄ █   █ ▄▀▀▀▀ █   █", "The whole stack, live from one git push.", "Start deploying", "npx quarry deploy"}, []string{"Plainsheet", "atelier nine"}},
		{"platform-features", platform, []string{"quarry.example/deployments/main", "Deploying to 38 regions", "Orbit Labs"}, []string{"The whole stack"}},
		{"platform-footer", platform, []string{"Your next deploy is one command away.", "Create a free project", "© 2026 ▲ Quarry"}, nil},
		{"product", product, []string{"Plainsheet", "⊙ plainsheet.example", "█▀▀▀▄ █     ▄▀▀▀▄ █▄  █", "Skip the busywork.", "app.plainsheet.example", "Onboarding", "Amara"}, []string{"Quarry"}},
		{"product-pricing", product, []string{"Simple plans that grow with you", "Monthly", "Most popular", "$12", "$29", "Choose Team"}, []string{"Skip the busywork."}},
		{"product-yearly", product, []string{"$10", "$24", "Choose Team"}, []string{"$12", "$29"}},
		{"product-end", product, []string{"Where is my data stored?", "Give your team one calm page.", "© 2026 ◧ Plainsheet"}, []string{"Drop in a CSV"}},
		{"studio", studio, []string{"atelier nine", "Taking projects for spring", "W E   D E S I G N", "██████", "Start a project"}, []string{"Plainsheet"}},
		{"studio-work", studio, []string{"Fieldnote", "Open Orchard", "Interaction Annual, 2024"}, nil},
		{"platform-again", platform, []string{"The whole stack, live from one git push."}, []string{"atelier nine"}},
		{"event", event, []string{"Fieldwork 26", "12 to 13 November", "Two days on interface engineering", "█   █  ▀▀▀█", "Early bird", "Get tickets →"}, []string{"█   █ █▀▀▀▄"}},
		{"event-later", event, []string{"█   █ █▀▀▀▄", "Early bird"}, []string{"█   █  ▀▀▀█"}},
		{"store", store, []string{"Hearth mug", "Colour Lake", "Side", "Front", "Top", "16 oz"}, []string{"Your cart"}},
		{"store-large", store, []string{"Add to cart · $32"}, []string{"Add to cart · $28"}},
		{"store-cart", store, []string{"Your cart", "1 item", "Lake · 16 oz × 1", "Subtotal", "Added to cart"}, nil},
		{"project", project, []string{"taskn", "❯ taskn ", "v2.4.0 · remote cache is here", "18.4k"}, []string{"taskn run build", "318 passed"}},
		{"project-output", project, []string{"❯ taskn run build --watch", "taskn 2.4.0 · 12 tasks", "318 passed"}, nil},
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
