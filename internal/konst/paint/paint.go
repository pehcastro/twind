package paint

const TabStop = 8

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
