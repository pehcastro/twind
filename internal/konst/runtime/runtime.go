package runtime

import "time"

const (
	FrameInterval  = time.Second / 60
	MotionInterval = time.Second / 30
	MeasurePasses  = 2

	GoroutineHeader = 64
)
