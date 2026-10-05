package chart

const (
	TickCount          = 5
	StepMantissa       = 5
	FirstDigitMantissa = 10
	StepShift          = 2
	CeilSlack          = 1e-9
	TopMargin          = 5
	StrokeWidth        = 2
	ActiveDotRadius    = 6
	BarRadius          = 4
	BarGap             = 4
	CategoryGap        = 0.1
	GridWidth          = 1
	GridAlpha          = 0.5
	CursorAlpha        = 0.5
	FillOpacity        = 0.4
	FillTop            = 0.8
	FillBottom         = 0.1
	Samples            = 4
	CurveStep          = 2
	MaxCurveSteps      = 64
	CellThreshold      = 0.5
	CellRows           = 2
	LegendGap          = 1
	MinPlot            = 1
	TooltipChrome      = 4
	TooltipGap         = 2
	ThousandsEvery     = 3
	DitherDot          = 2
	DitherBits         = 2
	DitherFloor        = 0.3
	DitherFlat         = 0.6
	BrushRows          = 2
	BrushDim           = 0.6
	BrushEdge          = 2
	BrushSalt          = 1
	MinWindow          = 2
	MaxArcSteps        = 256
	PolarFill          = 0.9
	PieGrow            = 0.08
	DonutHole          = 0.6
	RadialHole         = 0.27
	RadialGap          = 0.2
	RadarCols          = 1
	RadarRows          = 1.5
	RadarLabel         = 0.6
	RadarSide          = 0.1
	RadarFill          = 0.6
	RadarAxes          = 3
	MapFloor           = 0.25
	MapLower           = 1.0 / 3
	MapUpper           = 2.0 / 3
	CodeBytes          = 3
	LabelGap           = 1
	WorldMaxBytes      = 32 << 10
)

const (
	WorldNoCode   = "-99"
	WorldDropped  = "ATA"
	WorldGrid     = 2048
	WorldSimplify = 0.75
	WorldMinRing  = 3
	WorldFileMode = 0o644
	ProjX0        = 0.8707
	ProjX2        = -0.131979
	ProjX4        = -0.013791
	ProjX10       = 0.003971
	ProjX12       = -0.001529
	ProjY0        = 1.007226
	ProjY2        = 0.015085
	ProjY6        = -0.044475
	ProjY8        = 0.028874
	ProjY10       = -0.005916
)

const (
	Upper = "▀"
	Lower = "▄"
	Full  = "█"
	Grid  = "─"
	Key   = "■"
	Blank = " "
)
