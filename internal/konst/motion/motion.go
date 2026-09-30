package motion

import "time"

const (
	BezierNewtonSteps  = 8
	BezierBisectSteps  = 40
	BezierEpsilon      = 1e-7
	BezierSlopeFloor   = 1e-6
	SeriesReach        = 1.0 / 16
	SharedResponseBits = 3
	SharedResponseMix  = 0x9E3779B97F4A7C15
	RestDelta          = 1e-3
	RestSpeed          = 1e-2
	SettleSample       = time.Millisecond
	SettleLimit        = 10 * time.Second
)
