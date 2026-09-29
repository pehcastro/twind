package input

const MaxSequenceBytes = 4096

const (
	NUL = 0x00
	BEL = 0x07
	BS  = 0x08
	TAB = 0x09
	CR  = 0x0d
	ESC = 0x1b
	SP  = 0x20
	DEL = 0x7f
	CSI = "\x1b["
	SS3 = "\x1bO"
)

const (
	CtrlLetterBase = 'a' - 1
	CtrlSymbolBase = '@'
)

const (
	PasteStart = 200
	PasteEnd   = "\x1b[201~"
)

const (
	ModifyOtherKeys    = 27
	InBandResize       = 48
	InBandResizeParams = 5
	LegacyKeyParam     = 1
)

const (
	KittyModifierMask = 0x0f
	KittyMeta         = 0x20
	KittyKeypadFirst  = 57399
	EventPress        = 1
	EventRepeat       = 2
	EventRelease      = 3
)

const (
	MouseButtonMask    = 0x03
	MouseNoButton      = 0x03
	MouseModifierShift = 2
	MouseModifierMask  = 0x07
	MouseMotion        = 0x20
	MouseWheel         = 0x40
	MouseExtraButtons  = 0x80
	MouseParams        = 3
)
