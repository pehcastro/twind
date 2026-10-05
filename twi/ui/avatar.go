package ui

import "github.com/pehcastro/twind/twi"

type AvatarSize uint8

const (
	AvatarSizeDefault AvatarSize = iota
	AvatarSizeSM
	AvatarSizeLG
)

func Avatar(s AvatarSize, children ...twi.NodeOption) twi.Node {
	return part("relative flex shrink-0 overflow-hidden rounded-full select-none "+pick("avatar", s, map[AvatarSize]string{
		AvatarSizeSM:      "h-1 w-2 mx-1 bg-muted shadow-[0_0_0_4px_var(--color-muted)]",
		AvatarSizeDefault: "h-3 w-6",
		AvatarSizeLG:      "h-5 w-10",
	}), children)
}

func AvatarFallback(children ...twi.NodeOption) twi.Node {
	return part("flex h-full w-full items-center justify-center rounded-full bg-muted text-muted-foreground", children)
}
