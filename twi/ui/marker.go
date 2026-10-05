package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/style"
)

type MarkerVariant uint8

const (
	MarkerDefault MarkerVariant = iota
	MarkerSeparator
	MarkerBorder
)

func Marker(v MarkerVariant, children ...twi.NodeOption) twi.Node {
	options := append([]twi.NodeOption{twi.Data("slot", "marker"), twi.Data("variant", pick("marker", v, map[MarkerVariant]string{MarkerDefault: "default", MarkerSeparator: "separator", MarkerBorder: "border"}))}, children...)
	if v == MarkerSeparator {
		rule := part("h-1 min-w-0 flex-1 border-t border-border", nil)
		options = slices.Concat([]twi.NodeOption{rule}, options, []twi.NodeOption{rule})
	}
	return part("group/marker relative flex flex-row min-h-1 w-full items-center gap-1 text-left text-muted-foreground "+pick("marker", v, map[MarkerVariant]string{
		MarkerDefault:   "",
		MarkerSeparator: "",
		MarkerBorder:    "border-b border-border",
	}), options)
}

func MarkerIcon(children ...twi.NodeOption) twi.Node {
	return part("shrink-0", append([]twi.NodeOption{twi.Tag(style.ElementSVG)}, children...))
}

func MarkerContent(children ...twi.NodeOption) twi.Node {
	return part("min-w-0 wrap-break-word group-data-[variant=separator]/marker:flex-none group-data-[variant=separator]/marker:text-center", children)
}
