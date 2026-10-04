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

## twind dev

Builds a package, runs it, and swaps the running program for a new build on every save:

```bash
twind dev ./myapp
twind dev -log .twind/cache/dev.log ./myapp -theme dream-light
```

It watches every Go and embedded file the package imports from your module. A save rebuilds in the background and regenerates any stale Style IR first; the new build then replaces the old program in the same terminal. A build that fails leaves the old program running and shows the first compiler error on the last row. Arguments after the package go to the program.

The swap keeps where you were. The focused element, every scroll offset and each value the app saves with `rt.Remember(key, save, restore)` are written before the old program stops and read back by the new one. The docs app remembers its page and theme this way.

`-log` appends a timed line for each step to a file: the saves, whether each IR was stale, the build and how long it took, the state carried across, and the swap. Read it when a swap feels slow.

From a clone of Twind, `pnpm run docs:dev` runs these docs under `twind dev`, logging to `.twind/cache/dev.log`, so a save in `docs/` or `apps/documentation/` shows on screen.

## twind docs

Opens this documentation:

```bash
twind docs
twind docs -page button -theme dream-light
```

## twind doctor

Prints what the current terminal supports: colour depth, synchronized output, graphics protocols and the cell size in pixels.

On Windows 10, Alacritty and Rio run through the ConPTY built into Windows, which drops mouse input, colour replies and images. `twind doctor -fix` puts Microsoft's newer ConPTY (`conpty.dll` and `OpenConsole.exe`, from a pinned and checksummed package) beside the terminal's exe, after you type `yes`. Restart the terminal afterwards.

```bash
twind doctor -fix
twind doctor -undo F:\Tools\Alacritty
```

`-undo` removes only the files the fix wrote and restores any it renamed to `.bak`. The terminal keeps them open while it runs, so close it and undo from another terminal, naming its folder. A folder under Program Files asks for administrator rights once, for the copy alone.

## Offering the fix in your app

People who run your app never install `twind`. The `fix` package gives your app the same fix, behind a flag of your own:

```go
if *fixTerminal {
	said, err := fix.ConPTY(context.Background(), fix.Ask(os.Stdin, os.Stdout))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(said)
}
```

`fix.Ask` prints what will happen (both files, the folder, the package and its sha256) and goes ahead only on `yes`. `fix.Undo("")` undoes the fix in the folder of the terminal it runs in, and `fix.Undo(dir)` undoes it in a named folder. Anything it will not do comes back as a `fix.Refused`: a terminal that ships its own ConPTY (Windows Terminal, WezTerm, VS Code, Zed), a Windows build whose own ConPTY is already new, an answer other than `yes`, or a folder that was already fixed. Twind never runs the fix on its own; only your flag does.

## Exit status

| Status | Meaning |
| --- | --- |
| 0 | done |
| 1 | a stale IR, or a failed build, run or query |
| 2 | a usage error |
