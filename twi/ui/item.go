package ui

import "github.com/twind-dev/twind/twi"

func Item(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center rounded-md border px-1 "+
		pick("item", v, map[Variant]string{
			Default: "border-transparent bg-transparent",
			Outline: "border-border",
			Muted:   "border-transparent bg-muted/50",
		})+" "+
		pick("item", s, map[Size]string{
			SizeDefault: "gap-2",
			SizeSM:      "gap-1",
		}), children)
}

func ItemMedia(v Variant, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row shrink-0 items-center justify-center gap-1 "+pick("item media", v, map[Variant]string{
		Default: "bg-transparent",
		Icon:    "h-1 w-3 rounded-sm bg-muted shadow-[0_0_0_1px_var(--color-border)]",
		Image:   "h-3 w-5 overflow-hidden rounded-sm",
	}), children)
}

func ItemContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-1 flex-col", children)
}

func ItemTitle(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 font-medium", children)
}

func ItemDescription(children ...twi.NodeOption) twi.Node {
	return part("font-normal text-muted-foreground", children)
}

func ItemActions(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}

func ItemGroup(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col", children)
}

func ItemSeparator(children ...twi.NodeOption) twi.Node {
	return Separator(Horizontal, children...)
}
