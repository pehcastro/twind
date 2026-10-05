# Changelog

## Unreleased

### Breaking

- The pipeline packages are no longer importable. Moved under `internal/`: `twi/buffer`, `twi/css`, `twi/edit`, `twi/events`, `twi/fix`, `twi/graphics`, `twi/highlight`, `twi/layout`, `twi/paint`, `twi/raster`, `twi/runtime`, `twi/scene`, `twi/theme/shadcn`, `twi/theme/gen`, `twi/icon/gen`.
- `twi/terminal` keeps only `Graphics` and its values (`GraphicsNone`, `GraphicsSixel`, `GraphicsITerm2`, `GraphicsKitty`, `GraphicsGDI`); the backend, detection and the rest moved under `internal/`.
- `twi/tailwind` keeps only `Stale`; the compiler and `Checker` moved under `internal/`.
- `twi.Runtime` no longer embeds the runtime. It keeps `Run`, `SetTheme`, `Theme`, `Invalidate`, `Quit`, `Dispatch`, `After`, `Focus`, `HideFocusRings`, `ScrollIntoView`, `Viewport`, `ContentBox`, `Clicks`, `Copy`, `Widths` and `Remember`; `Restyle` and the embedded `Runtime` field are gone.
- The `twi/fix` API reference page is removed; `twind doctor -fix` is unchanged.
- `twi.New` takes `twi.Option` and `twi.Render` takes `twi.RenderOption`; an option the entry does not read no longer compiles. `ColorProfile`, `Styles`, `Theme` and `Graphics` return `twi.Setting` and fit both. Migrate: a `[]twi.RenderOption` passed to `New` becomes `[]twi.Option`, and `Width` goes only to `Render`.
- `twi.Fullscreen` is removed: `rt.Run` without a backend takes the terminal fullscreen. Migrate: `twi.New(twi.Fullscreen(), opts...)` becomes `twi.New(opts...)`.
- `twi.NoClipboard` is removed. Migrate: drop it; `rt.Copy` always writes the clipboard.
- `twi.Backend` takes a `twi.Terminal` and a `twi.Clock`; a terminal reports `Capabilities() terminal.Capabilities` (from `twi/terminal`) instead of `Sync() bool`. Migrate: replace `Sync() bool { return s }` with `Capabilities() terminal.Capabilities { return terminal.Capabilities{Sync: s} }`.
- `OnFocus`, `OnBlur`, `OnPointerEnter`, `OnPointerLeave`, `OnPointerDownOutside` and `OnFocusOutside` take `func(*twi.Event)`. Migrate: `func() { ... }` becomes `func(*twi.Event) { ... }`.
- `twi.OnKey` takes `func(*twi.Event)`; the key is `e.Key`, and `e.PreventDefault()` stops Enter from clicking and the arrows from scrolling. Migrate: `func(k input.KeyEvent) { ... }` becomes `func(e *twi.Event) { k := e.Key; ... }`.
- `twi.OnHotkey` takes `func(*twi.Event)` and takes a key with `e.PreventDefault()` instead of returning true. Migrate: `return true` becomes `e.PreventDefault()`.
- `twi.Input` and `twi.NewInput` are removed. Migrate: use `ui.NewInput(rt)`, which edits the same way and draws the field.
- `twi.NewRef` is removed; it ignored its runtime. Migrate: `twi.NewRef(rt)` becomes `&twi.Ref{}`.
- `twi.RenderString` returns `(string, error)` instead of panicking. Migrate: check the error.
- `rt.Copy` returns nothing; it never failed. Migrate: drop the error check.
- `rt.Focus` returns nothing and runs on the runtime's goroutine after the current handler, from any goroutine. Migrate: drop the result; the focused element's `OnFocus` reports the move.
- `rt.ContentBox` takes the `*twi.Event` whose handler runs. Migrate: `rt.ContentBox(e.Current())` becomes `rt.ContentBox(e)`.
- `drive.Frame.Cells` is removed; the frame itself has `Width`, `Height`, `At(x, y)` and `Row(y)`, which return `drive.Cell` (grapheme, `Fg`, `Bg`, `Attr` with `drive.Bold` and the rest, `Width` with `drive.Wide` and `drive.Continuation`). Migrate: `f.Cells().At(x, y)` becomes `f.At(x, y)`.
- `drive.Styles` is removed; `drive.With` passes any `twi.Option` to the driven runtime. Migrate: `drive.Styles(sheet)` becomes `drive.With(twi.Styles(sheet))`.
- `ui.Input` and `ui.Textarea` end an undo step after a pause in typing, on the runtime clock. `rt.Now` reads that clock.
- `ui.Variant` and `ui.Size` are gone. Each component takes its own type with shadcn's names, and a variant or size the component does not draw no longer compiles: `ButtonVariant` (`ButtonDefault`, `ButtonDestructive`, `ButtonOutline`, `ButtonSecondary`, `ButtonGhost`, `ButtonLink`) and `ButtonSize` (`ButtonSizeDefault`, `ButtonSizeXS`, `ButtonSizeSM`, `ButtonSizeLG`, `ButtonSizeIcon`), also taken by every `Trigger(v, s, ...)` and `Close(v, s, ...)`; `AlertVariant`; `BadgeVariant`; `AvatarSize`; `ItemVariant`, `ItemSize` and `ItemMediaVariant`; `EmptyMediaVariant`; `AttachmentMediaVariant`; `BubbleVariant`; `MarkerVariant`; `ToggleVariant` and `ToggleSize` for `Toggle` and `ToggleGroup`; `SidebarMenuButtonSize`. Migrate: `ui.Button(ui.Outline, ui.SizeSM, ...)` becomes `ui.Button(ui.ButtonOutline, ui.ButtonSizeSM, ...)`, `ui.Badge(ui.Secondary, ...)` becomes `ui.Badge(ui.BadgeSecondary, ...)`, and so on.
- `ui.Marker` variants take shadcn's names: `Ruled` becomes `MarkerSeparator`, `Bordered` becomes `MarkerBorder`.
- One alignment enum: `ui.Alignment` becomes `ui.Align` with `AlignCenter`, `AlignStart` and `AlignEnd`, and the input group's `ui.Align` is gone. `ui.Side` values are prefixed: `SideBottom`, `SideTop`, `SideRight`, `SideLeft`. `InputGroupAddon` takes a `ui.Side`. Migrate: `ui.Start` becomes `ui.AlignStart`, `ui.Right` becomes `ui.SideRight`, `ui.InlineStart` becomes `ui.SideLeft`, `ui.InlineEnd` becomes `ui.SideRight`, `ui.BlockStart` becomes `ui.SideTop` and `ui.BlockEnd` becomes `ui.SideBottom`.
- `Message`, `Bubble` and `BubbleReactions` draw every `Align` and `Side` instead of panicking on center or a left or right side.
- `ui.Upload` values are prefixed: `UploadDone`, `UploadIdle`, `UploadUploading`, `UploadProcessing`, `UploadFailed`.
- `ui.Input` and `ui.Textarea` call `OnSubmit`, renamed from `Submit`. Migrate: `in.Submit = f` becomes `in.OnSubmit = f`.
- Positional bools become named options: `PaginationLink`, `SidebarMenuButton` and `SidebarMenuSubButton` lose their bool and read `ui.Active(on)`; `Resizable.Handle` loses its bool and reads the new `Resizable.WithHandle` field. Migrate: `ui.PaginationLink(i == page, ...)` becomes `ui.PaginationLink(ui.Active(i == page), ...)`, `r.Handle(true)` becomes `r.WithHandle = true` and `r.Handle()`.
- `DropdownMenuSeparator`, `SelectSeparator`, `SidebarSeparator` and `InputOTPSeparator` take options like every other part.
- `Toaster.Show`, `Success` and `Error` take any number of `ToastAction`s, one button each. Migrate: `toaster.Show(t, d, ui.ToastAction{})` becomes `toaster.Show(t, d)`.
- `ui.ToastKind` and its values are unexported; nothing took them.

## v0.5.0 (2026-10-05)

The first public release: try every demo with one command, right-to-left text, pixel charts, forms.

- Module path is now `github.com/pehcastro/twind`. The library requires only `golang.org/x/mod` and `golang.org/x/sys`.
- Try it: `docs.ps1` and `docs.sh` (and one pair per example) download the `twind` binary for your system from the GitHub release, check its sha256 and run it. `twind try` runs the landing pages, portfolio, playground and gallery from the same binary.
- `twi/chart` draws in pixels through the new `twi.Canvas`: pie, donut, radial, radar, a world map by country, dithered areas, horizontal, negative and labelled bars, step areas, and an area chart with a range brush that zooms.
- Right-to-left: text is shown in visual order (all 91,707 Unicode bidi conformance lines pass); `twi.Dir` and `ui.Direction` set a paragraph's direction.
- `ui.Form` binds fields to rules, messages, submit and reset.
- Select, native select and combobox open from a click anywhere on the control, and their lists draw above any card that clips.
- One-row buttons and pills read as one pill in Windows Terminal. The PowerShell window draws letters its console font lacks (Hebrew, symbols) from a fallback font.
- `twind doctor -fix` takes `y`. Known: two Twind apps in two tabs of one Zed window still draw over each other.
- Sticky positioning; End and PgDn scroll the page; scroll thumbs stay under overlays; toasts clear open sheets.
- One highlighter colours code in the docs and the portfolio. The docs have an API reference generated from the source.
- Speed: event paths built once per frame (up to 1.37x less garbage); layout cost of sticky and half rows paid only where used.

## v0.4.0 (2026-10-04)

Build a full app: events, focus, mouse, motion, the shadcn component set, a docs app and a dev mode, in every Windows terminal.

- Events in `twi`: `OnClick`, `OnPointerDown`, `OnPointerMove`, `OnPointerUp`, `OnPointerEnter`, `OnPointerLeave`, `OnPointerDownOutside`, `OnFocusOutside`, `OnScroll`, `OnPaste`, `OnHotkey`, `OnWidth`. Focus moves with Tab, events bubble, and `NonModalFocusScope` lets focus leave a popover.
- Also in `twi`: `Ref`, `NewRef` and `Measure` read where an element is and how big; `Focus`, `ScrollIntoView`, `Timer`, `TopLayer`, `At`, `Tag`, `Data`, `Classes`, `NoClipboard`.
- Text fields edit like tofu's prompt: click places the cursor, drag and double click select, word moves, undo and redo, history, Shift+Enter for a new line, long pastes as chips. Selected text copies with OSC 52.
- Styling: CSS grid, scroll containers, `sticky`, breakpoints by terminal width, `group-`, `peer-`, `has-` and child variants, transitions, keyframes, enter and exit animations, `animate-spin`, one corner rounded at a time, dashed borders, half-row spacing where the terminal draws pixels.
- `twi/ui`: the shadcn set, among them dialog, sheet, drawer, popover, tooltip, menus, menubar, command palette, toasts, tabs, select, combobox, calendar, sidebar, resizable, carousel, accordion, form controls, and chat parts. Focus rings show only from the keyboard.
- New packages: `twi/chart`, `twi/icon`, `twi/highlight` (Go, bash, JSON, TOML, CSS, YAML, Markdown, TypeScript, TSX), `twi/markdown` (`Parse`, `Render` with components), `twi/fix` (installs a newer ConPTY beside Alacritty or Rio, with undo).
- Themes: seven built-in themes with a light and dark toggle; a theme loads from a shadcn CSS file.
- `twi/drive`: `Click`, `Move`, `Down`, `Up`, `Hold`, `Wheel`, `Clipboard`; script verbs `widths`, `wheel`, `move`, `down`.
- CLI: `twind new`; `twind dev` rebuilds and swaps the app on save and keeps its page and state; `twind docs`; `twind doctor -fix` and `-undo`.
- Terminals: Windows Terminal, WezTerm, Contour and VS Code draw surfaces with sixel; the PowerShell window, Zed, Alacritty and Rio draw them on an overlay window; Windows consoles get 24-bit colour, mouse input and stand-ins for missing glyphs. `SUPPORT_MATRIX.md` lists every terminal.
- The docs app: `apps/documentation`, a page for every component and guide, live examples next to their code, search.
- Examples: playground pages for every component, the gallery dashboard, six landing pages in `examples/sites/landing`, a portfolio in `examples/sites/portfolio`.
- Speed: style, layout, text wrapping and image encoding are 1.5x to 2.9x faster; per-version numbers in the gains ledger.

Breaking:

- `twi/events`: `MouseDown`, `MouseMove`, `MouseUp` are now `PointerDown`, `PointerMove`, `PointerUp`.
- `twi/theme.Builtin()` returns the seven built-in themes instead of every shadcn palette.
- `twi/scene`: `Truncate` and `Walk` are removed.

## v0.3.0 (2026-09-30)

Run a fullscreen app whose surfaces are pixels under crisp terminal text.

- `twi.New`, `rt.Run`, `twi.NewSignal(rt, v)` and `rt.Dispatch`: a retained app that redraws only what changed, with key events, runtime themes (`rt.SetTheme`, every shadcn palette, light and dark) and a headless driver (`twi/drive`) with a fake clock and a script format (`size`, `press`, `type`, `wait`, `resize`, `frame`).
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
