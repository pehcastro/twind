package theme_test

import (
	"testing"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

func TestThemeBuiltin(t *testing.T) {
	seen := map[string]int{}
	for _, th := range theme.Builtin() {
		seen[th.Name]++
		for tok := theme.Background; tok <= theme.Ring; tok++ {
			if th.Tokens[tok].Kind != color.Literal {
				t.Errorf("%s %d: %s unset", th.Name, th.Scheme, tok)
			}
		}
	}
	for _, name := range []string{"neutral", "zinc", "slate", "stone", "rose", "blue", "green", "orange", "violet"} {
		if seen[name] != 2 {
			t.Errorf("%s: %d schemes, want light and dark", name, seen[name])
		}
	}
}
