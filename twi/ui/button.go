package ui

import "github.com/twind-dev/twind/twi"

func Button(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row shrink-0 items-center justify-center gap-1 rounded-md font-medium select-none disabled:opacity-50 "+
		pick("button", v, map[Variant]string{
			Default:     "bg-primary text-primary-foreground hover:bg-primary/90",
			Destructive: "bg-destructive text-white hover:bg-destructive/90 dark:bg-destructive/60",
			Outline:     "bg-background shadow-[0_0_0_1px_var(--color-border)] hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:hover:bg-input/50",
			Secondary:   "bg-secondary text-secondary-foreground hover:bg-secondary/80",
			Ghost:       "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
			Link:        "text-primary hover:underline",
		})+" "+
		pick("button", s, map[Size]string{
			SizeDefault: "px-2",
			SizeXS:      "px-1",
			SizeSM:      "px-2",
			SizeLG:      "px-3",
			SizeIcon:    "w-3",
		}), children)
}
