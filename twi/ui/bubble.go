package ui

import "github.com/twind-dev/twind/twi"

func BubbleGroup(children ...twi.NodeOption) twi.Node {
	return part("flex min-w-0 flex-col gap-1", children)
}

func Bubble(v Variant, a Alignment, children ...twi.NodeOption) twi.Node {
	return part("group/bubble relative flex w-fit max-w-[80%] min-w-0 flex-col group-data-[align=end]/message:self-end "+pick("bubble", a, map[Alignment]string{
		Start: "",
		End:   "self-end",
	})+" "+pick("bubble", v, map[Variant]string{
		Default:     "*:data-[slot=bubble-content]:bg-primary *:data-[slot=bubble-content]:text-primary-foreground",
		Secondary:   "*:data-[slot=bubble-content]:bg-secondary *:data-[slot=bubble-content]:text-secondary-foreground",
		Muted:       "*:data-[slot=bubble-content]:bg-muted",
		Tinted:      "*:data-[slot=bubble-content]:bg-primary/15 *:data-[slot=bubble-content]:text-foreground dark:*:data-[slot=bubble-content]:bg-primary/30",
		Outline:     "*:data-[slot=bubble-content]:bg-background *:data-[slot=bubble-content]:shadow-[0_0_0_1px_var(--color-border)]",
		Ghost:       "max-w-full *:data-[slot=bubble-content]:rounded-none *:data-[slot=bubble-content]:bg-transparent *:data-[slot=bubble-content]:px-0",
		Destructive: "*:data-[slot=bubble-content]:bg-destructive/10 *:data-[slot=bubble-content]:text-destructive dark:*:data-[slot=bubble-content]:bg-destructive/20",
	}), append([]twi.NodeOption{twi.Data("slot", "bubble"), aligned(a)}, children...))
}

func BubbleContent(children ...twi.NodeOption) twi.Node {
	return part("w-fit max-w-full min-w-0 overflow-hidden rounded-lg px-1 wrap-break-word group-data-[align=end]/bubble:self-end", append([]twi.NodeOption{twi.Data("slot", "bubble-content")}, children...))
}

func BubbleReactions(s Side, a Alignment, children ...twi.NodeOption) twi.Node {
	return part("absolute z-10 flex flex-row w-fit shrink-0 items-center justify-center gap-1 rounded-full bg-muted px-1 "+pick("bubble reactions", s, map[Side]string{
		Bottom: "-bottom-1",
		Top:    "-top-1",
	})+" "+pick("bubble reactions", a, map[Alignment]string{
		Start: "left-1",
		End:   "right-1",
	}), children)
}
