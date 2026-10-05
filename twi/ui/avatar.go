package ui

import "github.com/pehcastro/twind/twi"

func Avatar(s Size, children ...twi.NodeOption) twi.Node {
	return part("relative flex shrink-0 overflow-hidden rounded-full select-none "+pick("avatar", s, map[Size]string{
		SizeSM:      "h-1 w-2 mx-1 bg-muted shadow-[0_0_0_4px_var(--color-muted)]",
		SizeDefault: "h-3 w-6",
		SizeLG:      "h-5 w-10",
	}), children)
}

func AvatarFallback(children ...twi.NodeOption) twi.Node {
	return part("flex h-full w-full items-center justify-center rounded-full bg-muted text-muted-foreground", children)
}
