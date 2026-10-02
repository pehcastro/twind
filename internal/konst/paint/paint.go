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
	FirstLineGlyph = '─'
	LastBlockGlyph = '▟'
	CellEighths    = 8
	CoverAlpha     = 0.6
	ShadowHalfCell = 0.25
	ShadowFullCell = 0.75
	OneRowTint     = 0.35
	OneRowLine     = 0.6
	OneRowLiftLuma = 20 * 1000

	OneRowInsetCell = 1.0 / 16
)

const (
	MediumShade      = "▒"
	LightShade       = "░"
	MediumShadeAlpha = 0.5
	ShadeDark        = 0.25
	ShadeLight       = 0.125
)

const (
	ConsoleGlyphs   = " ¯×•…‹›←↑→↓↕√≡─│┄┆┈┊┌┐└┘═║╔╗╚╝╭╮╯╰▀▄█▌▐■▪◊○●◦▴▾"
	ConsoleMissing  = "⌃⌄⊗✕◧✓⌕◐↗⇅⇧⌘◆◎▣▤⣾⣽⣻⢿⡿⣟⣯⣷▁▂▃▅▆▇▔▏▎▍▋▊▉▕‖⊟⊠∿⋮⌂□▦▭▯▱△▽►◇◉◍◔◠◌⇗➤⇘⇕↷↻↶↺☆☻☼☾♪✥✦✱✻❐⑂◷⚷ϟ⠶⠿"
	ConsoleStandIns = "▴▾××≡√>●↑↕↑#◊○■≡|/-\\|/-\\_▄▄▄██¯│▌▌▌██│║_@~:^■#■■#^v>◊○○○○○↑>↓↕→○←○*○*●•+***■Y○-!::"
	ContrastSteps   = 16
)

const (
	TileColumns     = 8
	BridgedTiles    = 6
	KittyFirstImage = 0x74770000
	EraseCells      = "X"
	DefaultBg       = "\x1b[49m"
)
