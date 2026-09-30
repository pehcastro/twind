package ui

import "github.com/twind-dev/twind/twi"

func Empty(children ...twi.NodeOption) twi.Node {
	return part("flex flex-1 flex-col min-w-0 items-center justify-center gap-1 rounded-lg px-3 py-1 text-center md:px-6 md:py-3", children)
}

func EmptyHeader(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col max-w-48 items-center text-center", children)
}

func EmptyMedia(v Variant, children ...twi.NodeOption) twi.Node {
	return part("mb-1 flex shrink-0 items-center justify-center "+pick("empty media", v, map[Variant]string{
		Default: "bg-transparent",
		Icon:    "h-3 w-5 rounded-lg bg-muted text-foreground",
	}), children)
}

func EmptyTitle(children ...twi.NodeOption) twi.Node {
	return part("font-medium", children)
}

func EmptyDescription(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}

func EmptyContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col w-full max-w-48 min-w-0 items-center gap-1", children)
}
