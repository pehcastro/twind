package konst

import "time"

const (
	DevEnv          = "TWIND_DEV"
	StateEnv        = "TWIND_DEV_STATE"
	AltScreen       = "\x1b[?1049h"
	Poll            = 100 * time.Millisecond
	StopGrace       = time.Second
	SnapshotVersion = 1
	ErrorStyle      = "\x1b[97;41m"
	SaveCursor      = "\x1b7"
	RestoreCursor   = "\x1b8"
	EraseRight      = "\x1b[K"
)
