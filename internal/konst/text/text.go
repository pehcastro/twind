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
)
