package terminal

import "time"

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

const (
	EnterScreen    = CSI + "?1049h" + CSI + "?25l" + CSI + "?2004h" + CSI + "?1004h"
	LeaveScreen    = CSI + "?1004l" + CSI + "?2004l" + CSI + "?25h" + CSI + "?1049l" + Reset
	MouseOn        = CSI + "?1003h" + CSI + "?1006h"
	MouseOff       = CSI + "?1006l" + CSI + "?1003l"
	KittyPush      = CSI + ">1u"
	KittyPop       = CSI + "<u"
	CellQuery      = CSI + "16t" + CSI + "14t" + CSI + "c"
	KittyQuery     = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	KittyOK        = "\x1b_Gi=31;OK\x1b\\"
	Queries        = KittyQuery + CSI + "?2026$p" + CSI + "?u" + CellQuery
	InlineQueries  = KittyQuery + CSI + "?2026$p" + CSI + "6n" + CellQuery
	OriginOn       = CSI + "?6h"
	OriginOff      = CSI + "?6l"
	RegionReset    = CSI + "r"
	CellReport     = "6"
	WindowReport   = "4"
	SixelAttribute = 4
	SyncMode       = 2026
	ModeSet        = 1
	ModeReset      = 2
)

const (
	QueryTimeout  = 100 * time.Millisecond
	EscapeTimeout = 50 * time.Millisecond
	EventBuffer   = 256
	ReplyBuffer   = 16
	ReadBuffer    = 4096
)
