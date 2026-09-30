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
	EnterScreen   = CSI + "?1049h" + CSI + "?25l" + CSI + "?2004h" + CSI + "?1004h"
	LeaveScreen   = CSI + "?1004l" + CSI + "?2004l" + CSI + "?25h" + CSI + "?1049l" + Reset
	MouseOn       = CSI + "?1003h" + CSI + "?1006h"
	MouseOff      = CSI + "?1006l" + CSI + "?1003l"
	KittyPush     = CSI + ">1u"
	KittyPop      = CSI + "<u"
	CellQuery     = CSI + "16t" + CSI + "14t" + CSI + "c"
	KittyQuery    = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	KittyOK       = "\x1b_Gi=31;OK\x1b\\"
	MarginsQuery  = CSI + "?69$p"
	Queries       = KittyQuery + CSI + "?2026$p" + CSI + "?2027$p" + MarginsQuery + CSI + "?u" + CellQuery
	InlineQueries = KittyQuery + CSI + "?2026$p" + MarginsQuery + CellQuery
	CursorHome    = CSI + "H"
	CursorQuery   = CSI + "6n"
	GraphemesOn   = CSI + "?2027h"
	GraphemesOff  = CSI + "?2027l"
	ProbeBegin    = SyncBegin + CSI + "?7l" + "\x1b7" + CursorQuery
	ProbeStep     = CursorQuery + "\x1b8"
	ProbeEnd      = CSI + "K" + CSI + "?7h" + SyncEnd
	Probes        = ProbeBegin +
		"\U0001F1E7\U0001F1F7" + ProbeStep +
		"\U0001F468\U0000200D\U0001F469\U0000200D\U0001F467" + ProbeStep +
		"\U00002764\U0000FE0F" + ProbeStep +
		"\U0001F44D\U0001F3FD" + ProbeStep +
		"1\U0000FE0F\U000020E3" + ProbeStep +
		ProbeEnd
	OriginOn       = CSI + "?6h"
	OriginOff      = CSI + "?6l"
	RegionRows     = "r"
	RegionColumns  = "s"
	RegionReset    = CSI + RegionRows
	MarginsOn      = CSI + "?69h"
	MarginsOff     = CSI + "?69l"
	ScrollUp       = "S"
	ScrollDown     = "T"
	MarginMode     = 69
	CellReport     = "6"
	WindowReport   = "4"
	SixelAttribute = 4
	SyncMode       = 2026
	GraphemeMode   = 2027
	ModeSet        = 1
	ModeReset      = 2
	ModeKeptSet    = 3
)

const (
	QueryTimeout    = 100 * time.Millisecond
	EscapeTimeout   = 50 * time.Millisecond
	EventBuffer     = 256
	ReplyBuffer     = 16
	ReadBuffer      = 4096
	WheelLines      = 3
	WheelUpReport   = 64
	WheelDownReport = 65
	ArrowLines      = 1
)
