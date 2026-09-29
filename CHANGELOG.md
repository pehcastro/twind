# Changelog

## v0.1.0 (2026-09-29)

Render a DOM once, in static mode, through real Tailwind.

- `twi.Element`, `twi.Text`, `twi.Class` build a tree; `twi.Render(w, node, opts...)` and `twi.RenderString` print it once, sized to the terminal or 80 columns, with `Width`, `ColorProfile` and `Styles` options.
- Styles come from the official Tailwind 4.3.3 build, compiled at `go generate` time into typed Go (`internal/twirgen`). A program builds and tests without the Tailwind executable, and a test fails when the generated styles are stale.
- Supported in this release: flex row and column, grow, shrink, basis, gap, padding, margin, sizes with min and max, alignment, colours, borders with rounded corners, bold, italic, underline, strikethrough.
- Text is sanitised before it reaches the terminal. Widths follow Unicode 17.0.0 with grapheme clusters.
- Colour follows `NO_COLOR`, `FORCE_COLOR`, `COLORTERM` and Windows Terminal, and downgrades to 256, 16 or none.
- Windows is supported: virtual terminal output is enabled for the call and restored after.

Not yet: fullscreen and inline modes, input, events, focus, grid, runtime themes, the CLI.
