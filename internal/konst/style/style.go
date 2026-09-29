package style

const (
	IRMagic         = "TWIR"
	IRVersion       = 1
	TailwindVersion = "4.3.3"
	OpaquePercent   = 100
	NormalWeight    = 400
	BoldMinWeight   = 600
	BoldWeight      = 700
	MaxVarDepth     = 32
	GeneratedFile   = "twir_gen.go"
)

const PresetTheme = `@theme {
  --spacing: 1px;
  --breakpoint-sm: 80px;
  --breakpoint-md: 100px;
  --breakpoint-lg: 140px;
  --breakpoint-xl: 180px;
}
`
