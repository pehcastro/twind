package ui

import "github.com/twind-dev/twind/twi"

func Skeleton(children ...twi.NodeOption) twi.Node {
	return part("rounded-md bg-accent", children)
}
