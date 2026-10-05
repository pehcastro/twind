# Installation

Twind needs Go and nothing else. A program that uses it builds from a clean checkout with the Go toolchain alone.

## Start a new app

The `twind` command writes a working fullscreen app into an empty directory:

```bash
go run github.com/pehcastro/twind/cmd/twind new myapp
cd myapp
go run .
```

The app has a sidebar, cards, a dialog and a theme picker. Its `README.md` says how to run it, test it and drive it.

## Add Twind to a module

```bash
go get github.com/pehcastro/twind
```

Then render a tree:

```go
func main() {
	rt := twi.New(twi.Fullscreen(), twi.Styles(sheet))
	rt.Run(func() twi.Node {
		return twi.Element(twi.Class("p-1"), twi.Text("Hello"))
	})
}
```

## Styles

Tailwind classes are compiled when you build, not when the program runs. Add this line to the package that holds your classes:

```go
//go:generate go run github.com/pehcastro/twind/internal/twirgen
```

Then run `twind build`. It writes `twir_gen.go`, a `Styles()` function with every rule your classes need. Commit it: the program then builds without Tailwind. `twind check` says whether it is stale, and a test can say the same.

## Commands

| Command | What it does |
| --- | --- |
| `twind new dir` | writes a new app |
| `twind build` | regenerates stale Style IR |
| `twind check` | reports stale Style IR |
| `twind drive script` | runs a script headless and writes its frames |
| `twind doctor` | prints what this terminal supports |
| `twind docs` | opens this documentation |

The [CLI](cli.md) page has each command's flags.
