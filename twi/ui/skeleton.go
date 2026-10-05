package ui

import "github.com/pehcastro/twind/twi"

func Skeleton(children ...twi.NodeOption) twi.Node {
	return part("rounded-md bg-accent", children)
}
