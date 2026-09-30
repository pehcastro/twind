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
	EncodeBuckets      = 4096
)

const (
	PingPeak     = 0.75
	PingScale    = 2
	PulseMiddle  = 0.5
	PulseOpacity = 0.5
	BounceMiddle = 0.5
	BounceLift   = -25

	BounceFallX1 = 0.8
	BounceFallY1 = 0
	BounceFallX2 = 1
	BounceFallY2 = 1
	BounceRiseX1 = 0
	BounceRiseY1 = 0
	BounceRiseX2 = 0.2
	BounceRiseY2 = 1
)
