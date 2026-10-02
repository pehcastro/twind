package konst

import "time"

const (
	TailwindRelease = "https://github.com/tailwindlabs/tailwindcss/releases/download/v"
	DevEnv          = "TWIND_DEV"
	StateEnv        = "TWIND_DEV_STATE"
	AltScreen       = "\x1b[?1049h"
	Poll            = 100 * time.Millisecond
	NotifyBuffer    = 64 << 10
	Settle          = 10 * time.Millisecond
	StopGrace       = time.Second
	SnapshotVersion = 1
	ErrorStyle      = "\x1b[97;41m"
	SaveCursor      = "\x1b7"
	RestoreCursor   = "\x1b8"
	EraseRight      = "\x1b[K"
	KeptBuilds      = 16
	StillActive     = 259
	NoInline        = "-l"
	NoDWARF         = "-w"
	ArchiveMagic    = "!<arch>\n"
	ArchiveHeader   = 60
	ArchiveNameEnd  = 16
	ArchiveSizeAt   = 48
	ArchiveSizeEnd  = 58
)
