package ui

import "github.com/pehcastro/twind/twi"

func AspectRatio(children ...twi.NodeOption) twi.Node {
	return part("relative flex w-full aspect-square overflow-hidden", children)
}
