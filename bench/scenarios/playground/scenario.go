package playground

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen -o twir_gen.go

const (
	Columns = 120
	Rows    = 40
)

func App(rt *twi.Runtime) func() twi.Node {
	size := strconv.Itoa(Columns) + "x" + strconv.Itoa(Rows)
	return playground(rt, env{cwd: "F:/localhost/ephem-sh/twind", profile: "truecolor", size: func() string { return size }}, state{theme: themeIndex("zinc-dark"), focus: "input"}, "")
}
