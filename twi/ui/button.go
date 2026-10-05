package ui

import "github.com/pehcastro/twind/twi"

type ButtonVariant uint8

const (
	ButtonDefault ButtonVariant = iota
	ButtonDestructive
	ButtonOutline
	ButtonSecondary
	ButtonGhost
	ButtonLink
)

type ButtonSize uint8

const (
	ButtonSizeDefault ButtonSize = iota
	ButtonSizeXS
	ButtonSizeSM
	ButtonSizeLG
	ButtonSizeIcon
)

func Button(v ButtonVariant, s ButtonSize, children ...twi.NodeOption) twi.Node {
	return part(button(v, s, idleRing(v)+" "+focusRing), append([]twi.NodeOption{twi.Focusable()}, children...))
}

func idleRing(v ButtonVariant) string {
	if v == ButtonOutline {
		return "shadow-[0_0_0_1px_var(--color-border)]"
	}
	return ""
}

func button(v ButtonVariant, s ButtonSize, ring string) string {
	return "flex flex-row shrink-0 items-center justify-center gap-1 rounded-md font-medium select-none disabled:opacity-50 [&_svg]:shrink-0 [&_svg]:pointer-events-none " +
		pick("button", v, map[ButtonVariant]string{
			ButtonDefault:     "bg-primary text-primary-foreground hover:bg-primary/90",
			ButtonDestructive: "bg-destructive text-white hover:bg-destructive/90 dark:bg-destructive/60",
			ButtonOutline:     "bg-background hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:hover:bg-input/50",
			ButtonSecondary:   "bg-secondary text-secondary-foreground hover:bg-secondary/80",
			ButtonGhost:       "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
			ButtonLink:        "text-primary hover:underline",
		}) + " " +
		pick("button", s, map[ButtonSize]string{
			ButtonSizeDefault: "px-2",
			ButtonSizeXS:      "px-1",
			ButtonSizeSM:      "px-2",
			ButtonSizeLG:      "px-3",
			ButtonSizeIcon:    "w-3",
		}) + " " + ring
}
