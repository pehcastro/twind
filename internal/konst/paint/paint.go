package paint

const TabStop = 8

const (
	SelectionRed   = 51
	SelectionGreen = 144
	SelectionBlue  = 255
	SelectionAlpha = 128
)

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
	HalfEdges      = "▄▌▀▐"
	UpperHalf      = "▀"
	CellEighths    = 8
	CoverAlpha     = 0.6
	ShadowHalfCell = 0.25
	ShadowFullCell = 0.75
)

const (
	ConsoleGlyphs   = " ¯×•…‹›←↑→↓↕√≡─│┄┆┈┊┌┐└┘═║╔╗╚╝╭╮╯╰▀▄█▌▐■▪◊○●◦▴▾"
	ConsoleMissing  = "⌃⌄⊗✕◧✓⌕◐↗⇅⇧⌘◆◎▣▤⣾⣽⣻⢿⡿⣟⣯⣷▁▂▃▅▆▇▔▏▎▍▋▊▉▕"
	ConsoleStandIns = "▴▾××≡√>●↑↕↑#◊○■≡|/-\\|/-\\_▄▄▄██¯│▌▌▌██│"
	ContrastSteps   = 16
)

const (
	TileColumns     = 8
	KittyFirstImage = 0x74770000
	EraseCells      = "X"
	DefaultBg       = "\x1b[49m"
)
