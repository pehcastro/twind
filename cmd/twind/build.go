package main

import (
	"io"
	"os"
)

func build(args []string, stdout io.Writer) error {
	patterns, err := packages("build", "Regenerates the Style IR of each package that is stale or missing one, through its twirgen go:generate\nline, then checks every Style IR in the packages as twind check does. Tailwind comes from .twind/bin in\nthis or a parent directory, or else is fetched once, pinned and checksum-checked, into the user cache.", args)
	if err != nil {
		return err
	}
	dirs, irs, err := survey(patterns)
	if err != nil {
		return err
	}
	fresh := map[string]bool{}
	for _, ir := range irs {
		fresh[ir.dir] = true
	}
	for _, ir := range irs {
		if ir.stale {
			fresh[ir.dir] = false
		}
	}
	var st *styler
	for _, dir := range dirs {
		if fresh[dir] {
			continue
		}
		if st == nil {
			work, err := os.MkdirTemp("", "twind-build-")
			if err != nil {
				return err
			}
			defer func() { _ = os.RemoveAll(work) }()
			st = newStyler(work, stdout)
		}
		if err := st.regenerate(dir); err != nil {
			return err
		}
	}
	return check(patterns, stdout)
}
