package ui

import "github.com/twind-dev/twind/twi"

func Badge(v Variant, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full px-1 font-medium "+
		pick("badge", v, map[Variant]string{
			Default:     "bg-primary text-primary-foreground",
			Secondary:   "bg-secondary text-secondary-foreground",
			Destructive: "bg-destructive text-white dark:bg-destructive/60",
			Outline:     "text-foreground shadow-[0_0_0_1px_var(--color-border)]",
			Ghost:       "",
			Link:        "text-primary",
		}), children)
}
