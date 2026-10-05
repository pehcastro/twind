package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pehcastro/twind/internal/shadcn"
)

func main() {
	pkg := flag.String("pkg", "main", "package of the generated file")
	out := flag.String("out", "theme_gen.go", "Go file to write")
	flag.Parse()
	fail := func(err error) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var files []shadcn.File
	for _, arg := range flag.Args() {
		paths := []string{arg}
		if info, err := os.Stat(arg); err == nil && info.IsDir() {
			paths, _ = filepath.Glob(filepath.Join(arg, "*.css"))
		}
		for _, p := range paths {
			src, err := os.ReadFile(p)
			if err != nil {
				fail(err)
			}
			files = append(files, shadcn.File{Path: p, Src: src})
		}
	}
	if len(files) == 0 {
		fail(fmt.Errorf("no CSS files in %v", flag.Args()))
	}
	src, warnings, err := shadcn.Generate(*pkg, files)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "%s:%d: %s ignored: %s\n", w.File, w.Line, w.Name, w.Reason)
	}
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fail(err)
	}
}
