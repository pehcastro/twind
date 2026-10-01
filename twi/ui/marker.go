package ui

import (
	"slices"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/style"
)

func Marker(v Variant, children ...twi.NodeOption) twi.Node {
	options := append([]twi.NodeOption{twi.Data("slot", "marker"), twi.Data("variant", pick("marker", v, map[Variant]string{Default: "default", Ruled: "separator", Bordered: "border"}))}, children...)
	if v == Ruled {
		rule := part("h-1 min-w-0 flex-1 border-t border-border", nil)
		options = slices.Concat([]twi.NodeOption{rule}, options, []twi.NodeOption{rule})
	}
	return part("group/marker relative flex flex-row min-h-1 w-full items-center gap-1 text-left text-muted-foreground "+pick("marker", v, map[Variant]string{
		Default:  "",
		Ruled:    "",
		Bordered: "border-b border-border",
	}), options)
}

func MarkerIcon(children ...twi.NodeOption) twi.Node {
	return part("shrink-0", append([]twi.NodeOption{twi.Tag(style.ElementSVG)}, children...))
}

func MarkerContent(children ...twi.NodeOption) twi.Node {
	return part("min-w-0 wrap-break-word group-data-[variant=separator]/marker:flex-none group-data-[variant=separator]/marker:text-center", children)
}
