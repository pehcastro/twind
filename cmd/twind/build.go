package main

import (
	"fmt"
	"io"
	"os/exec"
)

func build(args []string, stdout io.Writer) error {
	patterns, err := packages("build", "Runs the twirgen go:generate line of each package, which calls the pinned Tailwind in .twind/bin,\nthen checks every Style IR in the packages as twind check does.", args)
	if err != nil {
		return err
	}
	generate := exec.Command("go", append([]string{"generate", "-run", "twirgen"}, patterns...)...)
	generate.Stdout, generate.Stderr = stdout, stdout
	if err := generate.Run(); err != nil {
		return fmt.Errorf("go generate: %w", err)
	}
	return check(patterns, stdout)
}
