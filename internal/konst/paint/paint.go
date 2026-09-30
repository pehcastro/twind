package paint

const TabStop = 8

const (
	DamageColumns = 8
	HashPrime     = 0x9e3779b97f4a7c15
	HashRotate    = 31
)

const (
	SquareCorners  = "┌┐└┘"
	RoundedCorners = "╭╮╰╯"
	DoubleCorners  = "╔╗╚╝"
	SingleLines    = "─│─│"
	DashedLines    = "┄┆┄┆"
	DottedLines    = "┈┊┈┊"
	DoubleLines    = "═║═║"
	Hairlines      = "▁▏▔▕"
	UpperHalf      = "▀"
	PillCaps       = "▐▌"
)

const (
	TileColumns     = 8
	KittyFirstImage = 0x74770000
	EraseCells      = "X"
	DefaultBg       = "\x1b[49m"
)
