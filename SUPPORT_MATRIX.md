# Support matrix: Windows

Measured on Windows 10 19045, 2026-10-01, every terminal run on screen with the probe
(`.local/planning/scripts/probe/probe.ps1`, launched by `.local/planning/scripts/termtour.ps1 -Probe`).
Screenshots: `.local/prints/dev/terminals/<terminal>/probe/*.png` (docs tours: `<terminal>/base/`, later builds `<terminal>/v1/` and on). Hover was done by hand.

The target look is "H": real text in cells, with rounded borders, pills and soft shadows drawn as pixels around it. A card's fill stays strictly inside its border: zero overflow at the corners or edges, no square cell background behind a rounded corner, no half-block slivers.

## Verdict

| Terminal | Verdict | How Twind reaches the target look |
|---|---|---|
| Windows Terminal | good | sixel for borders, pills, shadows |
| WezTerm | good | sixel (kitty graphics is off by default) |
| Contour | good | sixel |
| PowerShell / cmd window (conhost) | good | GDI drawing on the console window |
| PowerShell + oh-my-posh | good | same as conhost; oh-my-posh changes nothing |
| Rio with a newer conpty.dll | good | sixel |
| Alacritty with a newer conpty.dll | fair | overlay window over the terminal; cells alone have square corners |
| VS Code (terminal.integrated.enableImages on) | good | sixel |
| Zed terminal (Windows) | good | overlay window over the panel, placed by reading faint marker cells off the screen (no clicks); follows Zed's own cell edges |
| Rio (stock) | bad | no mouse, images print as text |
| mintty (Git Bash, MSYS2 zsh) | bad | no mouse; overlay works but drifts |
| Alacritty (stock) | worst | no mouse, no images, square box corners |

Policy: mintty is not supported. Alacritty is the gate: every rendering or component fix is checked in Alacritty before anyone says it works.

"Stock" Alacritty, Rio and mintty run through the Windows 10 inbox ConPTY. That layer, not the terminal, drops mouse input, image protocols and colour replies, and answers DA1 itself.

## Capabilities

| Terminal | ConPTY | 24-bit | Mouse | Hover | Sixel | OSC 10/11/4 | Pixel size (14t/16t) | Rounded box glyphs | Powerline glyphs |
|---|---|---|---|---|---|---|---|---|---|
| Windows Terminal | own | yes | yes | yes | yes | yes | yes | yes | yes |
| WezTerm | own | yes | yes | yes | yes | yes | yes | yes | yes |
| Contour | own | yes | yes | yes | yes | yes | yes | yes | yes |
| conhost | none | yes | yes | yes | no | no | no | yes | yes |
| Rio + newer conpty | newer | yes | yes | yes | yes (and iTerm2) | yes | yes | yes | yes |
| Alacritty + newer conpty | newer | yes | yes | yes | no | yes | yes | square corners | no (font) |
| Zed | own | yes | yes | yes | no | yes | 14t only, and wrong (reports 7x16, real 7.8x16.81) | yes | no (font) |
| VS Code, images on | own | yes | yes | yes | yes | n/m | n/m | yes | yes |
| Rio | inbox | yes | no | no | printed as text | no | no | yes | yes |
| mintty | inbox | yes | no | no | printed as text | mangled | no | yes | yes |
| mintty, MSYS=disable_pcon | none (pipes) | yes | n/m | n/m | yes | n/m | n/m | yes | yes |
| Alacritty | inbox | yes | no | no | no | no | no | square corners | no (font) |

n/m: not measured. VS Code without `terminal.integrated.enableImages`: not measured.

## Per element, best method by path

| Element | sixel | conhost GDI | overlay window | cells only |
|---|---|---|---|---|
| Card border | ring cells as sixel, text cells inside | ring drawn on console window | ring drawn on overlay | rounded box glyphs, border cells on page colour (B2) |
| Input border | 3-row ring | same | same | rounded box glyphs (I3) |
| Badge / pill | end caps as sixel | same | same | background only (B1); Powerline caps only where the font has them |
| Shadow | soft shadow in the ring image | same | same | half-block shadow (S2) |
| Hover | redraw the cells and the caps | same | same | redraw the cells |

Rules learnt:
- Sixel pixels in the page colour must be transparent: sixel colour is in percent, so a baked page colour shows as a darker box.
- conhost repaints after Twind draws: GDI must redraw after each frame settles, or parts get erased.
- Never draw with the 16 ANSI colours or the default background: PowerShell turns them navy, Zed turns them purple. Read OSC 11 when the host background is needed.
- Half discs (U+25D6/7) render wrong in Cascadia and Consolas: do not use.

## How to identify each case at startup

1. `GetConsoleWindow()` is visible and its class is `ConsoleWindowClass`: conhost. Use GDI. (Under WT and every ConPTY the class is `PseudoConsoleWindow`.)
2. Query DA1, XTVERSION, `CSI 14t`, `CSI 16t`, `OSC 10/11/4`.
3. DA1 is exactly `ESC[?1;0c` and no `16t` reply: the Windows 10 inbox ConPTY. No mouse, no images. Tell the user (see Fixes).
4. DA1 lists attribute `4`: sixel. Use the sixel path.
5. Otherwise find the terminal's window (walk parent processes to the first with a main window) and its cell size (`14t` text area / cols). If both are known, use the overlay path.
6. Otherwise cells only.

Env vars (`TERM_PROGRAM`, `ZED_TERM`, `WT_SESSION`) leak into child windows started from another terminal: use them as hints, never as proof.

## Fixes a user can apply

- Alacritty, Rio and any ConPTY terminal on Windows 10: put a newer `conpty.dll` and `OpenConsole.exe` beside the terminal's exe (both ship with WezTerm and Windows Terminal). Proven for Alacritty and Rio: mouse, colour replies, and sixel in Rio.
- mintty: `MSYS=disable_pcon` bypasses ConPTY; sixel then works. Twind needs a pipe input path for this case.
- WezTerm: `enable_kitty_graphics = true` if kitty graphics are wanted.

## Open

- Overlay: done for Zed on Windows (TWI-207). Not yet for Alacritty. DPI above 100% and several Zed windows not measured.
- Contour clicks work (TWI-199, 2026-10-01): the earlier failure was the tour clicking between two sidebar rows. `twi/terminal/record_windows_test.go` records raw console input inside any terminal (`TWIND_RECORD=<file>`).
- VS Code with images off: not measured.


## Zed (Windows, macOS, Linux)

Zed has no image protocol on any platform. It draws block elements (U+2580-259F) and sextants (U+1FB00-1FB3B) itself as exact rectangles, and shades ░▒▓ as 25/50/75% alpha fills of the foreground. Box drawing, rounded corners and Powerline come from the font. Cell sizes are fractional (7.8 x 16.81 px at 13 px); `CSI 14t` truncates them and `16t` gets no reply. 24-bit colours bypass `minimum_contrast`; palette colours 0-15 do not. Source: `.local/sources/zed`, `crates/terminal_view/src/terminal_element.rs` 181-865.

Bench: `.local/planning/scripts/probe/zed_probe.ps1 -Tag zed`, run inside Zed. Result `zed/probe/variants.png`.

| Element | Use in Zed | Avoid |
|---|---|---|
| Card | rounded box glyphs, border cells on page colour (Z1); eighth-block outline (Z2) when the font is unknown | sextant corners (stepped) |
| Badge | background only, one cell of padding each side | sextant, shade or Powerline caps |
| Shadow | two-step shade: ▒ next to the card, ░ one cell further (S3) | full black cells |
| Input | rounded box glyphs (I1), eighth outline (I2), or an eighth underline (I3) | sextant field |
| Rounded smooth card | overlay window, Windows only (Z6) | |

Always emit 24-bit colours in Zed so `minimum_contrast` leaves them alone.



## Status after the night of 2026-10-01

Done, each checked live in Alacritty, WT and the PowerShell window:
- e4fec90: every Windows console gets 24-bit colour; Twind names the terminal (conhost, InboxConPTY, Zed, Other).
- dc59971, b603928: without pixels, cards get rounded box borders with the fill inside; badges and one-row controls get a fill or tint, no half-block slivers, no side bars.
- 4621944: sixel leaves the page colour to the terminal, so no darker box around cards.
- a39ce3b: the PowerShell window asks its font which characters exist and draws stand-ins for the rest.
- f374ed1: in Zed only, cell shadows are ▒ then ░ (waiting for your look).
- 00cf027: `twind doctor` tells users of the inbox ConPTY how to get mouse and images back.
- Docs: sidebar follows the palette (62fb85c), previews stay in their frame (e5f6288), card header grid (a14b1e7), header fits narrow windows (33dd3f7).

Not started, needs your go-ahead: drawing rounded rings with pixels where the terminal has none (GDI on the conhost window, an overlay window for Alacritty and Zed on Windows). Both are Windows-only and fragile (see the probe results above).
