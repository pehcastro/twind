# Changelog

## v0.3.0 (2026-09-30)

Run a fullscreen app whose surfaces are pixels under crisp terminal text.

- `twi.NewRuntime`, `rt.Run`, `twi.NewSignal(rt, v)` and `rt.Dispatch`: a retained app that redraws only what changed, with key events, runtime themes (`rt.SetTheme`, every shadcn palette, light and dark) and a headless driver (`twi/drive`) with a fake clock and a script format (`size`, `press`, `type`, `wait`, `resize`, `frame`).
- Backgrounds, borders, radius, soft shadows, gradients and opacity are rasterised into images and sent as Sixel, Kitty graphics or iTerm2 images, whichever the terminal speaks, with text drawn as real cells on top. Terminals without graphics, `NO_COLOR` and 256 or 16 colours fall back to cells.
- Terminal detection at start: graphics protocol, cell pixel size, synchronized output, emoji widths and grapheme mode.
- `position` absolute, relative and fixed, z-index, overflow clipping.
- The `twind` CLI: `build`, `check`, `drive`, `doctor`.
- Examples: `examples/playground` (fullscreen app with a theme picker) and `examples/gallery`.

v0.2.0 was not tagged; its surfaces work is part of this release.

## v0.1.0 (2026-09-29)

Render a DOM once, in static mode, through real Tailwind.

- `twi.Element`, `twi.Text`, `twi.Class` build a tree; `twi.Render(w, node, opts...)` and `twi.RenderString` print it once, sized to the terminal or 80 columns, with `Width`, `ColorProfile` and `Styles` options.
- Styles come from the official Tailwind 4.3.3 build, compiled at `go generate` time into typed Go (`internal/twirgen`). A program builds and tests without the Tailwind executable, and a test fails when the generated styles are stale.
- Supported in this release: flex row and column, grow, shrink, basis, gap, padding, margin, sizes with min and max, alignment, colours, borders with rounded corners, bold, italic, underline, strikethrough.
- Text is sanitised before it reaches the terminal. Widths follow Unicode 17.0.0 with grapheme clusters.
- Colour follows `NO_COLOR`, `FORCE_COLOR`, `COLORTERM` and Windows Terminal, and downgrades to 256, 16 or none.
- Windows is supported: virtual terminal output is enabled for the call and restored after.

Not yet: fullscreen and inline modes, input, events, focus, grid, runtime themes, the CLI.
