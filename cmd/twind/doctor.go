package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

func doctor(args []string, stdout io.Writer) error {
	fs := flags("doctor", "", "Asks the terminal on stdin and stdout what it supports and prints the answers and how long they took.\nTWIND_GRAPHICS set to none, sixel, iterm2 or kitty overrides the graphics answer, as it does for every Twind program.")
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
	start := time.Now()
	caps, _, err := terminal.Query(os.Stdin, os.Stdout)
	took := time.Since(start)
	if err != nil {
		return err
	}
	cell := "unknown"
	if caps.CellPixels.X > 0 {
		cell = strconv.Itoa(caps.CellPixels.X) + "x" + strconv.Itoa(caps.CellPixels.Y) + " pixels"
	}
	keyboard := "legacy"
	if caps.KittyKeyboard {
		keyboard = "kitty"
	}
	var widths []string
	for class, name := range [text.Classes]string{text.Flag: "flag", text.ZWJ: "zwj", text.VS16: "vs16", text.Modifier: "modifier", text.Keycap: "keycap"} {
		measured := "table"
		if caps.Widths[class] > 0 {
			measured = strconv.Itoa(caps.Widths[class])
		}
		widths = append(widths, name+" "+measured)
	}
	_, err = fmt.Fprintf(stdout, "size       %dx%d cells\ncolour     %s\nsync       %t\ngraphemes  %t\nemoji      %s\nkeyboard   %s\ngraphics   %s\ncell       %s\ndetected   in %s\n",
		width, height, profileName(terminal.Profile(os.Stdout, os.Getenv)), caps.Sync, caps.Graphemes, strings.Join(widths, ", "), keyboard, graphicsName(caps.Graphics), cell, took.Round(time.Millisecond))
	return err
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
