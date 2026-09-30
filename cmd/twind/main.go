package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const usage = `twind creates, builds, checks and drives Twind programs.

usage: twind <verb> [arguments]

  new    [-module path] dir            write a new fullscreen app into a new or empty directory
  build  [packages]                    regenerate each package's Style IR through its twirgen go:generate line
  check  [packages]                    report every stale Style IR, without Tailwind
  drive  [-out dir] script [package]   run a driver script headless and write its frames
  doctor                               print what this terminal supports
  docs   [-page name] [-theme name]    open the documentation

Packages are go list patterns, the current directory by default.
twind <verb> -h prints the verb's help.

Exit status: 0 done, 1 a stale IR or a failed build, run or query, 2 a usage error.
`

func main() {
	verbs := map[string]func([]string, io.Writer) error{"new": newApp, "build": build, "check": check, "drive": drive, "doctor": doctor, "docs": docs}
	if len(os.Args) < 2 || verbs[os.Args[1]] == nil {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	err := verbs[os.Args[1]](os.Args[2:], os.Stdout)
	var usageErr usageError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
	case errors.As(err, &usageErr):
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "twind "+os.Args[1]+":", err)
		os.Exit(1)
	}
}

type usageError string

func (e usageError) Error() string { return string(e) }

func flags(name, args, help string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "usage: twind %s %s\n\n%s\n", name, args, help)
		fs.PrintDefaults()
	}
	return fs
}

func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError(err.Error())
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func packages(name, help string, args []string) ([]string, error) {
	patterns, err := parse(flags(name, "[packages]", help), args)
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	return patterns, err
}

func goList(patterns []string, format string) ([]string, error) {
	out, err := exec.Command("go", append([]string{"list", "-f", format}, patterns...)...).Output()
	if exit := (*exec.ExitError)(nil); errors.As(err, &exit) {
		return nil, fmt.Errorf("go list: %w\n%s", err, exit.Stderr)
	}
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(out), "\r\n"), "\n"), nil
}
