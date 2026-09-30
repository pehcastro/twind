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
	Unprintable     = 0xff
	NoRoom          = -1
	LeapMin         = 16
	ByteLanes       = 8
	LowBits         = 0x0101010101010101
	HighBits        = 0x8080808080808080
)
