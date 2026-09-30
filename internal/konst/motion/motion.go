package motion

import "time"

const (
	BezierNewtonSteps = 8
	BezierBisectSteps = 40
	BezierEpsilon     = 1e-7
	BezierSlopeFloor  = 1e-6
	CriticalBand      = 1e-6
	RestDelta         = 1e-3
	RestSpeed         = 1e-2
	SettleSample      = time.Millisecond
	SettleLimit       = 10 * time.Second
	ResponseCache     = 8
)
