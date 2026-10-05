package ui

import "github.com/pehcastro/twind/twi"

type ItemVariant uint8

const (
	ItemDefault ItemVariant = iota
	ItemOutline
	ItemMuted
)

type ItemSize uint8

const (
	ItemSizeDefault ItemSize = iota
	ItemSizeSM
)

func Item(v ItemVariant, s ItemSize, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center rounded-md border px-1 "+
		pick("item", v, map[ItemVariant]string{
			ItemDefault: "border-transparent bg-transparent",
			ItemOutline: "border-border",
			ItemMuted:   "border-transparent bg-muted/50",
		})+" "+
		pick("item", s, map[ItemSize]string{
			ItemSizeDefault: "gap-2",
			ItemSizeSM:      "gap-1",
		}), children)
}

type ItemMediaVariant uint8

const (
	ItemMediaDefault ItemMediaVariant = iota
	ItemMediaIcon
	ItemMediaImage
)

func ItemMedia(v ItemMediaVariant, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row shrink-0 items-center justify-center gap-1 "+pick("item media", v, map[ItemMediaVariant]string{
		ItemMediaDefault: "bg-transparent",
		ItemMediaIcon:    "h-1 w-3 rounded-sm bg-muted shadow-[0_0_0_1px_var(--color-border)]",
		ItemMediaImage:   "h-3 w-5 overflow-hidden rounded-sm",
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
