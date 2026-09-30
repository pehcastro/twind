package ui

import "github.com/twind-dev/twind/twi"

func Separator(o Orientation, children ...twi.NodeOption) twi.Node {
	return part("shrink-0 border-border "+pick("separator", o, map[Orientation]string{
		Horizontal: "w-full border-t",
		Vertical:   "self-stretch border-l",
	}), children)
}
