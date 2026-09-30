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

type styleIR struct {
	dir, name string
	stale     bool
}

func check(args []string, stdout io.Writer) error {
	patterns, err := packages("check", "Reports each Style IR in the packages as fresh or stale against the classes the package and its twi/ui imports use.\nA Style IR is any .go file whose first line is the twirgen header. Tailwind is not needed.\nExits 1 when one is stale or when the packages hold none.", args)
	if err != nil {
		return err
	}
	_, irs, err := survey(patterns)
	if err != nil {
		return err
	}
	stale := 0
	for _, ir := range irs {
		state := "fresh"
		if ir.stale {
			state = "stale"
			stale++
		}
		if _, err := fmt.Fprintln(stdout, state, filepath.Join(ir.dir, ir.name)); err != nil {
			return err
		}
	}
	if len(irs) == 0 {
		return fmt.Errorf("no Style IR in %s", strings.Join(patterns, " "))
	}
	if stale > 0 {
		return fmt.Errorf("%d of %d Style IRs stale: run twind build", stale, len(irs))
	}
	return nil
}

func survey(patterns []string) ([]string, []styleIR, error) {
	dirs, err := goList(patterns, "{{.Dir}}")
	if err != nil {
		return nil, nil, err
	}
	var irs []styleIR
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, nil, err
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() || filepath.Ext(path) != ".go" {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return nil, nil, err
			}
			if !bytes.HasPrefix(src, []byte("// "+konst.IRMagic+" ")) {
				continue
			}
			stale, err := tailwind.Stale(dir, e.Name())
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", path, err)
			}
			irs = append(irs, styleIR{dir, e.Name(), stale})
		}
	}
	return dirs, irs, nil
}
