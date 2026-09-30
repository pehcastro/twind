package ui

import "github.com/twind-dev/twind/twi"

func Kbd(children ...twi.NodeOption) twi.Node {
	return part("flex flex-rowmin-w-3 items-center justify-center gap-1 rounded-sm bg-muted px-1 font-medium text-muted-foreground select-none", children)
}

func KbdGroup(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}
