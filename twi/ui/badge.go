package ui

import "github.com/pehcastro/twind/twi"

type BadgeVariant uint8

const (
	BadgeDefault BadgeVariant = iota
	BadgeSecondary
	BadgeDestructive
	BadgeOutline
	BadgeGhost
	BadgeLink
)

func Badge(v BadgeVariant, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full px-1 font-medium "+
		pick("badge", v, map[BadgeVariant]string{
			BadgeDefault:     "bg-primary text-primary-foreground",
			BadgeSecondary:   "bg-secondary text-secondary-foreground",
			BadgeDestructive: "bg-destructive text-white dark:bg-destructive/60",
			BadgeOutline:     "text-foreground shadow-[0_0_0_1px_var(--color-border)]",
			BadgeGhost:       "",
			BadgeLink:        "text-primary",
		}), children)
}
