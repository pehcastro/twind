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
	EnterScreen   = CSI + "?1049h" + CSI + "?25l" + CSI + "?2004h" + FocusOn
	LeaveScreen   = FocusOff + CSI + "?2004l" + CSI + "?25h" + CSI + "?1049l" + Reset
	FocusOn       = CSI + "?1004h"
	FocusOff      = CSI + "?1004l"
	MouseOn       = CSI + "?1003h" + CSI + "?1006h"
	MouseOff      = CSI + "?1006l" + CSI + "?1003l"
	KittyPush     = CSI + ">1u"
	KittyPop      = CSI + "<u"
	Fence         = CSI + "c"
	GridQuery     = CSI + "18t"
	CellQuery     = CSI + "16t" + CSI + "14t" + GridQuery + Fence
	KittyQuery    = "\x1b_Gi=31,s=1,v=1,a=q,t=d,f=24,o=z;eNpiYGAADAAAAwAB\x1b\\"
	KittyOK       = "\x1b_Gi=31;OK\x1b\\"
	MarginsQuery  = CSI + "?69$p"
	ModeQueries   = CSI + "?2026$p" + CSI + "?2027$p" + CSI + "?1004$p" + MarginsQuery + CSI + "?2048$p"
	InBandOn      = CSI + "?2048h"
	InBandOff     = CSI + "?2048l"
	KittyFenced   = KittyQuery + Fence
	ConhostClass  = 1
	ConhostOption = 0
	Queries       = ModeQueries + CSI + "?u" + CellQuery
	InlineQueries = ModeQueries + CellQuery
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
	CellReport     = 6
	WindowReport   = 4
	GridReport     = 8
	WindowParams   = 3
	InBandMode     = 2048
	SixelAttribute = 4
	SyncMode       = 2026
	GraphemeMode   = 2027
	FocusMode      = 1004
	ModeSet        = 1
	ModeReset      = 2
	ModeKeptSet    = 3
)

const (
	ConhostWindowClass = "ConsoleWindowClass"
	WindowClassLength  = 256
	ZedProgram         = "zed"
)

const (
	QueryTimeout    = 100 * time.Millisecond
	StartupTimeout  = time.Second
	EscapeTimeout   = 50 * time.Millisecond
	CellPoll        = 500 * time.Millisecond
	EventBuffer     = 256
	ReplyBuffer     = 16
	ReadBuffer      = 4096
	WheelLines      = 3
	WheelUpReport   = 64
	WheelDownReport = 65
	ArrowLines      = 1
	MultiClick      = 500 * time.Millisecond
	MultiClickSlack = 1
	WordClicks      = 2
	LineClicks      = 3
)

const (
	WheelLeftReport    = 66
	WheelRightReport   = 67
	LeftReport         = 0
	MiddleReport       = 1
	RightReport        = 2
	NoButtonReport     = 3
	MotionReport       = 32
	ModifierShift      = 2
	MouseReport        = CSI + "<"
	ConsoleRecordBytes = len(MouseReport + "127;32767;32767M")
)

const (
	OSC          = "\x1b]"
	BEL          = "\x07"
	ESC          = 0x1b
	BELByte      = 0x07
	DCS          = "\x1bP"
	ST           = "\x1b\\"
	ClipboardSet = OSC + "52;c;"
)

const (
	VersionQuery       = CSI + ">0q"
	SecondaryQuery     = CSI + ">c"
	MouseAnyMode       = 1003
	MouseSGRMode       = 1006
	PasteMode          = 2004
	DoctorModes        = CSI + "?1003$p" + CSI + "?1006$p" + CSI + "?2004$p"
	TruecolorProbe     = "1;2;3"
	TruecolorQuery     = CSI + "38;2;" + TruecolorProbe + "m" + DCS + "$qm" + ST + Reset
	TruecolorAnswer    = "1$r"
	ClipboardCap       = "4d73"
	ClipboardQuery     = DCS + "+q" + ClipboardCap + ST
	ClipboardAnswer    = "1+r"
	ClipboardAttribute = 52
	VersionAnswer      = ">|"
	SecondaryParams    = 3
	PointerQuery       = OSC + "22;?__current__" + ST
	PointerAnswer      = "22;"
	KeyboardQuery      = CSI + "?u"
	KittyRawQuery      = "\x1b_Gi=32,s=1,v=1,a=q,t=d,f=24;AAAA\x1b\\"
	KittyZlibAnswer    = "Gi=31;"
	KittyRawAnswer     = "Gi=32;"
	DoctorQueries      = VersionQuery + SecondaryQuery + KeyboardQuery + DoctorModes + TruecolorQuery + ClipboardQuery + PointerQuery + KittyQuery + KittyRawQuery + GridQuery
)
