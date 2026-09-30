package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/tailwind"
)

func check(args []string, stdout io.Writer) error {
	patterns, err := packages("check", "Reports each Style IR in the packages as fresh or stale against the classes the package and its twi/ui imports use.\nA Style IR is any .go file whose first line is the twirgen header. Tailwind is not needed.\nExits 1 when one is stale or when the packages hold none.", args)
	if err != nil {
		return err
	}
	dirs, err := goList(patterns, "{{.Dir}}")
	if err != nil {
		return err
	}
	found, stale := 0, 0
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() || filepath.Ext(path) != ".go" {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !bytes.HasPrefix(src, []byte("// "+konst.IRMagic+" ")) {
				continue
			}
			found++
			isStale, err := tailwind.Stale(dir, e.Name())
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			state := "fresh"
			if isStale {
				state = "stale"
				stale++
			}
			if _, err := fmt.Fprintln(stdout, state, path); err != nil {
				return err
			}
		}
	}
	if found == 0 {
		return fmt.Errorf("no Style IR in %s", strings.Join(patterns, " "))
	}
	if stale > 0 {
		return fmt.Errorf("%d of %d Style IRs stale: run twind build", stale, found)
	}
	return nil
}
