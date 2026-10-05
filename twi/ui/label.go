package ui

import "github.com/pehcastro/twind/twi"

func Label(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 font-medium select-none", children)
}
