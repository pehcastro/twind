package ui

import "github.com/twind-dev/twind/twi"

func ButtonGroup(o Orientation, children ...twi.NodeOption) twi.Node {
	return part("flex items-stretch "+pick("button group", o, map[Orientation]string{
		Horizontal: "flex-row",
		Vertical:   "flex-col",
	}), children)
}

func ButtonGroupSeparator(o Orientation, children ...twi.NodeOption) twi.Node {
	return part("relative shrink-0 self-stretch border-input "+pick("button group separator", o, map[Orientation]string{
		Horizontal: "border-t",
		Vertical:   "border-l",
	}), children)
}

func ButtonGroupText(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 rounded-md bg-muted px-2 font-medium shadow-[0_0_0_1px_var(--color-border)]", children)
}
