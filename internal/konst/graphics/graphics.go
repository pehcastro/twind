package graphics

const (
	SixelRegisters = 256
	SixelBand      = 6
	SixelZero      = '?'
	SixelMinRepeat = 4
	SixelPercent   = 100
	SixelCacheBits = 8
	SixelCacheHash = 0x9e3779b1
	SixelWord      = 8
	SixelByteBits  = 8
	SixelSpread    = 0x0101010101010101
	KittyChunk     = 4096
	KittyRawChunk  = KittyChunk / 4 * 3
	Base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	Base64Pairs    = 1 << 12
	Base64Group    = 6

	ZlibHeader             = "\x78\x01"
	AdlerModulus           = 65521
	ByteHighBits           = 0x80808080
	DeflateWindow          = 1 << 15
	DeflateMinMatch        = 3
	DeflateMaxMatch        = 258
	DeflateEndOfBlock      = 256
	DeflateLiterals        = 286
	DeflateDistances       = 30
	DeflateCodeLengths     = 19
	DeflateMaxBits         = 15
	DeflateCodeLengthBits  = 7
	DeflateDynamicFinal    = 0b101
	DeflateFixedFinal      = 0b011
	DeflateFixedLiterals   = 288
	DeflateFixedNineFrom   = 144
	DeflateFixedEightFrom  = 280
	DeflateFixedTokens     = 64
	DeflateRepeat          = 16
	DeflateZeros           = 17
	DeflateLongZeros       = 18
	DeflateRepeatMin       = 3
	DeflateRepeatMax       = 6
	DeflateZerosMax        = 10
	DeflateLongZerosMax    = 138
	DeflateCodeLengthOrder = "\x10\x11\x12\x00\x08\x07\x09\x06\x0a\x05\x0b\x04\x0c\x03\x0d\x02\x0e\x01\x0f"
	DeflateFlatSpan        = 64
	DeflateTokenBytes      = 8

	TokenSlot     = 10
	TokenMask     = 1<<TokenSlot - 1
	TokenRepeat   = 3 * TokenSlot
	TokenDistance = DeflateFixedLiterals
	DistanceSlots = 4
	TokenNone     = TokenDistance + DistanceSlots
	TokenLength   = 1 << (TokenSlot - 1)
	SymbolLength  = 32
	SymbolBits    = 1<<SymbolLength - 1
	ShiftMask     = 63
	ByteShiftMask = 56
	RunBlock      = 8
	RunGuard      = 8
	RunGuess      = 32

	Workers      = 8
	JobPixels    = 1 << 14
	CellBytes    = 2
	LineMemoBits = 6
	RunChunk     = 128
	LineSlots    = 16
	AlphaPair    = 0xff000000ff000000

	PNGSignature = "\x89PNG\r\n\x1a\n"
	PNGHeader    = "\x00\x00\x00\x0dIHDR"
	PNGDepth     = 8
	PNGOpaque    = 2
	PNGAlpha     = 6
	PNGUpFilter  = 2
)
