package main

import (
	"fmt"
	"os"

	"github.com/pehcastro/twind/examples/sites/portfolio/app"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
