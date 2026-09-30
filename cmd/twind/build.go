package main

import (
	"fmt"
	"io"
	"os/exec"
)

func build(args []string, stdout io.Writer) error {
	patterns, err := packages("build", "Runs the twirgen go:generate line of each package whose Style IR is stale or missing, which calls\nthe pinned Tailwind in .twind/bin, then checks every Style IR in the packages as twind check does.", args)
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
	var stale []string
	for _, dir := range dirs {
		if !fresh[dir] {
			stale = append(stale, dir)
		}
	}
	if len(stale) > 0 {
		generate := exec.Command("go", append([]string{"generate", "-run", "twirgen"}, stale...)...)
		generate.Stdout, generate.Stderr = stdout, stdout
		if err := generate.Run(); err != nil {
			return fmt.Errorf("go generate: %w", err)
		}
	}
	return check(patterns, stdout)
}
