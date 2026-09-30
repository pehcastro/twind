package ui

import "github.com/twind-dev/twind/twi"

func AspectRatio(children ...twi.NodeOption) twi.Node {
	return part("relative flex w-full aspect-square overflow-hidden", children)
}
