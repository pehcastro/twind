package style_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/tailwind"
	"github.com/pehcastro/twind/twi/theme"
)

func TestPresetThemeSyntaxTokens(t *testing.T) {
	bin, err := filepath.Abs(filepath.Join("..", "..", "..", ".twind", "bin", "tailwindcss-"+runtime.GOOS+"-"+map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("no pinned Tailwind: %v", err)
	}
	tokens := []theme.Token{theme.SyntaxKeyword, theme.SyntaxString, theme.SyntaxNumber, theme.SyntaxComment, theme.SyntaxFunction, theme.SyntaxConstant, theme.SyntaxNamespace, theme.SyntaxParameter, theme.SyntaxPunctuation}
	dir := t.TempDir()
	manifest, input, output := filepath.Join(dir, "manifest.txt"), filepath.Join(dir, "input.css"), filepath.Join(dir, "output.css")
	var classes string
	for _, token := range tokens {
		classes += "text-" + token.String() + "\n"
	}
	if err := os.WriteFile(manifest, []byte(classes), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte(tailwind.Input([]string{manifest})), 0o600); err != nil {
		t.Fatal(err)
	}
	if msg, err := exec.Command(bin, "-i", input, "-o", output).CombinedOutput(); err != nil {
		t.Fatalf("tailwind: %v\n%s", err, msg)
	}
	css, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	rules, _, err := tailwind.Compile(string(css))
	if err != nil {
		t.Fatal(err)
	}
	twind := map[style.Scheme]theme.Tokens{style.SchemeAny: theme.Default().Schemes[theme.Light], style.SchemeDark: theme.Default().Schemes[theme.Dark]}
	for _, token := range tokens {
		class := "text-" + token.String()
		found := map[style.Scheme]bool{}
		for _, r := range rules {
			if r.Class != class || len(r.Decls) == 0 {
				continue
			}
			found[r.When.Scheme] = true
			d := r.Decls[len(r.Decls)-1]
			if d.Token != token {
				t.Errorf("%s scheme %v: token %v, want %v", class, r.When.Scheme, d.Token, token)
			}
			if want := twind[r.When.Scheme][token]; d.Color != want {
				t.Errorf("%s scheme %v: colour %+v, want twind %+v", class, r.When.Scheme, d.Color.RGBA, want.RGBA)
			}
		}
		if !found[style.SchemeAny] || !found[style.SchemeDark] {
			t.Errorf("%s: rules for light %v and dark %v, want both", class, found[style.SchemeAny], found[style.SchemeDark])
		}
	}
}
