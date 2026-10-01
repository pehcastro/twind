package highlight

const (
	StackDepth       = 256
	MaxRules         = 255
	MaxStates        = 1<<16 - 1
	FailedProbes     = 16
	ParamScanTokens  = 64
	CallScanTokens   = 200
	BackScanSteps    = 16
	EmbedPasses      = 16
	ReadableContrast = 4.5
)
