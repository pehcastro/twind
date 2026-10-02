package ui

import "time"

const (
	ToastDuration = 4 * time.Second
	ToastTick     = 100 * time.Millisecond
	VisibleToasts = 3
	CommandRows   = 9
	MatchBase     = 1000
	PageSteps     = 10
	SliderMax     = 100
	MergeTokens   = 32
	SpinnerTurn   = time.Second
	DaysInWeek    = 7
	MonthsInYear  = 12
	HoverOpen     = 200 * time.Millisecond
	HoverShut     = 150 * time.Millisecond
	CollisionPadX = 1
	CollisionPadY = 0
	PercentWhole  = 100
	PanelMin      = 10
	PanelStep     = 5
	WeeksShown    = 6
	TextareaRows  = 8

	PasteChipRunes = 160
	PasteTabSpaces = 4
	FieldRowParts  = 5

	DrawerCloseDivisor = 4
)
