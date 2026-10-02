package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	konst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

const doctorHelp = `Asks the terminal on stdin and stdout what it supports and prints the answers and how long they took.
TWIND_GRAPHICS set to none, sixel, iterm2 or kitty overrides the graphics answer, as it does for every Twind program.
truecolor says whether the terminal echoed a 24-bit colour back through DECRQSS; colour is what Twind will use.
identity names the terminal when a rule recognises it, and the call or reply that decided it.
grid is the size the terminal itself reports; Twind lays out to it when it differs from size, the pty's.
kitty is the terminal's answer to a raw and a zlib kitty image; Twind uses kitty only when zlib is OK.
glyphs lists the width the terminal gives each glyph Twind draws, ? where it gave none.`

const doctorGlyphs = "─ │ ╭ ┼ █ ▀ ▄ ▌ ▏ ▕ ▁ ▦ ‹ › ⌄ ▸ ⌘ ⇧ ⌃ ✓ ✕ • ● ◆ ◐ ◦ ⊗ ⌕ … 中 🙂"

type answers struct {
	version   string
	primary   string
	secondary string
	keyboard  bool
	mouseAny  bool
	mouseSGR  bool
	paste     bool
	truecolor bool
	clipboard bool
	pointer   string
	grid      string
	kittyRaw  string
	kittyZlib string
	widths    []int
}

func doctor(args []string, stdout io.Writer) error {
	fs := flags("doctor", "[-o file]", doctorHelp)
	file := fs.String("o", "", "also write the report to this file")
	positional, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(positional) > 0 {
		fs.Usage()
		return usageError("doctor takes no arguments")
	}
	width, height, err := terminal.Size(os.Stdout)
	if err != nil {
		return fmt.Errorf("stdout is not a terminal: %w", err)
	}
	glyphs := strings.Fields(doctorGlyphs)
	start := time.Now()
	raw, skipped, err := doctorAsk(func(q string) ([]byte, error) { return terminal.Probe(os.Stdin, os.Stdout, q) }, glyphs)
	if err != nil {
		return err
	}
	answered := time.Since(start)
	caps, _, err := terminal.Query(os.Stdin, os.Stdout)
	took := time.Since(start)
	if err != nil {
		return err
	}
	cell := "unknown"
	if caps.CellPixels.X > 0 {
		cell = strconv.Itoa(caps.CellPixels.X) + "x" + strconv.Itoa(caps.CellPixels.Y) + " pixels"
	}
	var widths []string
	for class, name := range [text.Classes]string{text.Flag: "flag", text.ZWJ: "zwj", text.VS16: "vs16", text.Modifier: "modifier", text.Keycap: "keycap"} {
		measured := "table"
		if caps.Widths[class] > 0 {
			measured = strconv.Itoa(caps.Widths[class])
		}
		widths = append(widths, name+" "+measured)
	}
	a := parseAnswers(raw, len(glyphs), width)
	keyboard := "legacy"
	if caps.KittyKeyboard || a.keyboard {
		keyboard = "kitty"
	}
	var report strings.Builder
	fmt.Fprintf(&report, "size       %dx%d cells\nenv        TERM=%s COLORTERM=%s TERM_PROGRAM=%s WT_SESSION=%t\ncolour     %s\nidentity   %s\nsync       %t\ngraphemes  %t\nfocus      %t\nmargins    %t\nemoji      %s\nkeyboard   %s\ngraphics   %s\ncell       %s\n",
		width, height, text.Sanitize(os.Getenv("TERM"), text.ShowBidi), text.Sanitize(os.Getenv("COLORTERM"), text.ShowBidi), text.Sanitize(os.Getenv("TERM_PROGRAM"), text.ShowBidi), os.Getenv("WT_SESSION") != "",
		profileName(terminal.Profile(os.Stdout, os.Getenv)), identityName(caps.Identity), caps.Sync, caps.Graphemes, caps.Focus, caps.Margins, strings.Join(widths, ", "), keyboard, graphicsName(caps.Graphics), cell)
	fmt.Fprintf(&report, "%sdetected   in %s, first answer after %s\n%s", answerLines(a, glyphs), took.Round(time.Millisecond), answered.Round(time.Millisecond), identityFix(caps.Identity)+overlayLine(caps.Identity, terminal.OverlayHost))
	if skipped {
		report.WriteString("skipped    truecolor, clipboard and kitty: no DA1, or conhost's 1;0, which prints DCS and APC as text\n")
	}
	if *file != "" {
		if err := os.WriteFile(*file, []byte(report.String()), 0o600); err != nil {
			return err
		}
	}
	_, err = io.WriteString(stdout, report.String())
	return err
}

func doctorAsk(probe func(string) ([]byte, error), glyphs []string) ([]byte, bool, error) {
	raw, err := probe(konst.ProbeBegin + strings.Join(glyphs, konst.ProbeStep) + konst.ProbeStep + konst.ProbeEnd + konst.VersionQuery + konst.SecondaryQuery + konst.KeyboardQuery + konst.DoctorModes + konst.PointerQuery + konst.GridQuery)
	if primary := parseAnswers(raw, len(glyphs), 0).primary; err != nil || primary == "" || primary == joinParams([]int{konst.ConhostClass, konst.ConhostOption}) {
		return raw, true, err
	}
	printable, err := probe(konst.TruecolorQuery + konst.ClipboardQuery + konst.KittyQuery + konst.KittyRawQuery)
	return append(raw, printable...), false, err
}

func parseAnswers(raw []byte, glyphs, columns int) answers {
	a := answers{widths: make([]int, glyphs)}
	for rest := raw; ; {
		i := bytes.IndexByte(rest, konst.ESC)
		if i < 0 || i+1 == len(rest) {
			break
		}
		introducer := rest[i+1]
		rest = rest[i+2:]
		if introducer != 'P' && introducer != ']' && introducer != '_' {
			continue
		}
		end, next := bytes.IndexByte(rest, konst.BELByte), 1
		if st := bytes.Index(rest, []byte(konst.ST)); st >= 0 && (end < 0 || st < end) {
			end, next = st, len(konst.ST)
		}
		if end < 0 {
			break
		}
		body := string(rest[:end])
		rest = rest[end+next:]
		switch {
		case introducer == ']':
			if pointer, ok := strings.CutPrefix(body, konst.PointerAnswer); ok {
				a.pointer = text.Sanitize(pointer, text.ShowBidi)
			}
		case strings.HasPrefix(body, konst.KittyZlibAnswer):
			a.kittyZlib = text.Sanitize(body[len(konst.KittyZlibAnswer):], text.ShowBidi)
		case strings.HasPrefix(body, konst.KittyRawAnswer):
			a.kittyRaw = text.Sanitize(body[len(konst.KittyRawAnswer):], text.ShowBidi)
		case strings.HasPrefix(body, konst.VersionAnswer):
			a.version = text.Sanitize(body[len(konst.VersionAnswer):], text.ShowBidi)
		case strings.HasPrefix(body, konst.TruecolorAnswer):
			sgr := strings.ReplaceAll(body, ":", ";")
			a.truecolor = strings.Contains(sgr, "38;2;"+konst.TruecolorProbe) || strings.Contains(sgr, "38;2;;"+konst.TruecolorProbe)
		case strings.HasPrefix(body, konst.ClipboardAnswer):
			a.clipboard = a.clipboard || strings.HasPrefix(strings.ToLower(body[len(konst.ClipboardAnswer):]), konst.ClipboardCap)
		}
	}
	var d input.Decoder
	var cursors [][]int
	for _, ev := range append(d.Decode(raw), d.Quiet()...) {
		r, ok := ev.(input.ReplyEvent)
		if !ok {
			continue
		}
		switch r.Kind {
		case input.ReplyPrimaryAttributes:
			a.primary = joinParams(r.Params)
			a.clipboard = a.clipboard || slices.Contains(r.Params, konst.ClipboardAttribute)
		case input.ReplySecondaryAttributes:
			if len(r.Params) >= konst.SecondaryParams {
				a.secondary = joinParams(r.Params)
			}
		case input.ReplyMode:
			if len(r.Params) != 2 || r.Params[1] < konst.ModeSet || r.Params[1] > konst.ModeKeptSet {
				continue
			}
			switch r.Params[0] {
			case konst.MouseAnyMode:
				a.mouseAny = true
			case konst.MouseSGRMode:
				a.mouseSGR = true
			case konst.PasteMode:
				a.paste = true
			}
		case input.ReplyKeyboardFlags:
			a.keyboard = true
		case input.ReplyCursorPosition:
			if len(r.Params) == 2 {
				cursors = append(cursors, r.Params)
			}
		case input.ReplyWindow:
			if len(r.Params) == konst.WindowParams && r.Params[0] == konst.GridReport && r.Params[1] > 0 && r.Params[2] > 0 {
				a.grid = strconv.Itoa(r.Params[2]) + "x" + strconv.Itoa(r.Params[1])
			}
		}
	}
	if len(cursors) == glyphs+1 {
		origin := cursors[0]
		for g, end := range cursors[1:] {
			if advance := end[1] - origin[1]; end[0] == origin[0] && advance > 0 && end[1] < columns {
				a.widths[g] = advance
			}
		}
	}
	return a
}

func joinParams(params []int) string {
	s := make([]string, len(params))
	for i, p := range params {
		s[i] = strconv.Itoa(p)
	}
	return strings.Join(s, ";")
}

func answerLines(a answers, glyphs []string) string {
	orNone := func(s string) string {
		if s == "" {
			return "no answer"
		}
		return s
	}
	measured := make([]string, len(glyphs))
	for i, g := range glyphs {
		measured[i] = g + " ?"
		if a.widths[i] > 0 {
			measured[i] = g + " " + strconv.Itoa(a.widths[i])
		}
	}
	return fmt.Sprintf("terminal   %s\nprimary    %s\nsecondary  %s\nmouse      any-event %t, sgr %t\npaste      %t\ntruecolor  %t\nclipboard  %t\npointer    %s\ngrid       %s\nkitty      raw %s, zlib %s\nglyphs     %s\n",
		orNone(a.version), orNone(a.primary), orNone(a.secondary), a.mouseAny, a.mouseSGR, a.paste, a.truecolor, a.clipboard, orNone(a.pointer), orNone(a.grid), orNone(a.kittyRaw), orNone(a.kittyZlib), strings.Join(measured, " "))
}

func profileName(p color.Profile) string {
	switch p {
	case color.None:
		return "none"
	case color.Attributes:
		return "attributes only"
	case color.ANSI16:
		return "16 colours"
	case color.ANSI256:
		return "256 colours"
	case color.TrueColor:
		return "truecolor"
	}
	panic("twind: unknown colour profile " + strconv.Itoa(int(p)))
}

func identityName(i terminal.Identity) string {
	switch i {
	case terminal.IdentityConhost:
		return "conhost, from the console window: visible, class " + konst.ConhostWindowClass
	case terminal.IdentityInboxConPTY:
		return "InboxConPTY, from DA1: exactly 1;0, no 16t reply, no visible conhost window"
	case terminal.IdentityZed:
		return "Zed, from TERM_PROGRAM=zed confirmed by a 14t reply and no sixel in DA1"
	case terminal.IdentityOther:
		return "Other, no rule matched: not a visible conhost window, DA1 not exactly 1;0, not Zed"
	}
	panic("twind: unknown terminal identity " + strconv.Itoa(int(i)))
}

func overlayLine(i terminal.Identity, host func() string) string {
	if i != terminal.IdentityZed {
		return ""
	}
	return "overlay    " + host() + "\n"
}

func identityFix(i terminal.Identity) string {
	switch i {
	case terminal.IdentityInboxConPTY:
		return "fix        the Windows 10 inbox ConPTY drops mouse input, images and colour replies here.\n" +
			"           Put a newer conpty.dll and OpenConsole.exe beside the terminal's exe.\n" +
			"           Both ship with Windows Terminal and WezTerm, or use one of those instead.\n" +
			"           In mintty, set MSYS=disable_pcon to skip ConPTY.\n"
	case terminal.IdentityConhost, terminal.IdentityZed, terminal.IdentityOther:
		return ""
	}
	panic("twind: unknown terminal identity " + strconv.Itoa(int(i)))
}

func graphicsName(g terminal.Graphics) string {
	switch g {
	case terminal.GraphicsNone:
		return "none"
	case terminal.GraphicsSixel:
		return "sixel"
	case terminal.GraphicsITerm2:
		return "iterm2"
	case terminal.GraphicsKitty:
		return "kitty"
	}
	panic("twind: unknown graphics protocol " + strconv.Itoa(int(g)))
}
