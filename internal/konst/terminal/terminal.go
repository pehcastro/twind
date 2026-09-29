package terminal

const (
	CSI          = "\x1b["
	SyncBegin    = CSI + "?2026h"
	SyncEnd      = CSI + "?2026l"
	Reset        = CSI + "0m"
	FgBase       = 30
	FgBrightBase = 90
	FgExtended   = 38
	FgDefault    = 39
	BgOffset     = 10
	Palette256   = 5
	PaletteRGB   = 2
	BaseColors   = 8
	SGRReset     = 0
	SGRBold      = 1
	SGRDim       = 2
	SGRItalic    = 3
	SGRUnderline = 4
	SGRInverse   = 7
	SGRStrike    = 9
)
