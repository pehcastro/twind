# CLI

The `twind` command creates, builds, checks and drives Twind programs. Run it from a clone with `go run ./cmd/twind`, or install it:

```bash
go install github.com/twind-dev/twind/cmd/twind@latest
```

Every verb prints its own help with `twind <verb> -h`. Packages are `go list` patterns, the current directory by default.

## twind new

Writes a working fullscreen app into a new or empty directory:

```bash
twind new myapp
twind new -module example.com/myapp myapp
```

The app has a sidebar, cards, a dialog and a theme picker, a test that drives it, and a `README.md` that says how to run, test and drive it.

## twind build

Regenerates each package's Style IR, the `twir_gen.go` file that holds every rule your classes need:

```bash
twind build ./...
```

It runs the package's `//go:generate` line for twirgen, which calls the pinned Tailwind executable. Commit the result: the program then builds with the Go toolchain alone.

## twind check

Reports every stale Style IR without running Tailwind, and exits with status 1 when one is stale:

```bash
twind check ./...
```

Run it in CI. A test in the package can do the same with `tailwind.Stale`.

## twind drive

Runs a driver script against an app package headlessly, under a fake clock, and writes each `frame` as text and as ANSI:

```bash
twind drive -out frames tour.twd ./myapp
```

The package declares `func App(rt *twi.Runtime) func() twi.Node` and has its `Styles` function. See [Driving](driving.md) for the script verbs.

## twind docs

Opens this documentation:

```bash
twind docs
twind docs -page button -theme dream-light
```

## twind doctor

Prints what the current terminal supports: colour depth, synchronized output, graphics protocols and the cell size in pixels.

## Exit status

| Status | Meaning |
| --- | --- |
| 0 | done |
| 1 | a stale IR, or a failed build, run or query |
| 2 | a usage error |
