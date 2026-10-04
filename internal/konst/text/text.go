package text

const (
	Ellipsis        = "…"
	BidiVisible     = "<U+%04X>"
	RecordSize      = 4
	BlockShift      = 8
	BlockSize       = 1 << BlockShift
	WidthMask       = 0b11
	ConjunctShift   = 2
	ConjunctMask    = 0b11
	PictographicBit = 1 << 4
	EmojiBit        = 1 << 5
	RTLBit          = 1 << 6
	Unprintable     = 0xff
	NoRoom          = -1
	LeapMin         = 16
	ByteLanes       = 8
	LowBits         = 0x0101010101010101
	HighBits        = 0x8080808080808080
)

const (
	BidiClassMask = 0x1f
	BidiOpenBit   = 1 << 5
	BidiCloseBit  = 1 << 6
	BidiMirrorBit = 1 << 7
	BidiMaxDepth  = 125
	BracketDepth  = 63
	PairRuneBytes = 3
	PairSize      = 2 * PairRuneBytes
)
