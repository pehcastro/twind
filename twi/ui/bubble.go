package ui

import "github.com/pehcastro/twind/twi"

type BubbleVariant uint8

const (
	BubbleDefault BubbleVariant = iota
	BubbleSecondary
	BubbleMuted
	BubbleTinted
	BubbleOutline
	BubbleGhost
	BubbleDestructive
)

func BubbleGroup(children ...twi.NodeOption) twi.Node {
	return part("flex min-w-0 flex-col gap-1", children)
}

func Bubble(v BubbleVariant, a Align, children ...twi.NodeOption) twi.Node {
	return part("group/bubble relative flex w-fit max-w-[80%] min-w-0 flex-col group-data-[align=end]/message:self-end "+pick("bubble", a, map[Align]string{
		AlignStart:  "",
		AlignCenter: "self-center",
		AlignEnd:    "self-end",
	})+" "+pick("bubble", v, map[BubbleVariant]string{
		BubbleDefault:     "*:data-[slot=bubble-content]:bg-primary *:data-[slot=bubble-content]:text-primary-foreground",
		BubbleSecondary:   "*:data-[slot=bubble-content]:bg-secondary *:data-[slot=bubble-content]:text-secondary-foreground",
		BubbleMuted:       "*:data-[slot=bubble-content]:bg-muted",
		BubbleTinted:      "*:data-[slot=bubble-content]:bg-primary/15 *:data-[slot=bubble-content]:text-foreground dark:*:data-[slot=bubble-content]:bg-primary/30",
		BubbleOutline:     "*:data-[slot=bubble-content]:bg-background *:data-[slot=bubble-content]:shadow-[0_0_0_1px_var(--color-border)]",
		BubbleGhost:       "max-w-full *:data-[slot=bubble-content]:rounded-none *:data-[slot=bubble-content]:bg-transparent *:data-[slot=bubble-content]:px-0",
		BubbleDestructive: "*:data-[slot=bubble-content]:bg-destructive/10 *:data-[slot=bubble-content]:text-destructive dark:*:data-[slot=bubble-content]:bg-destructive/20",
	}), append([]twi.NodeOption{twi.Data("slot", "bubble"), aligned(a)}, children...))
}

func BubbleContent(children ...twi.NodeOption) twi.Node {
	return part("w-fit max-w-full min-w-0 overflow-hidden rounded-lg px-1 wrap-break-word group-data-[align=end]/bubble:self-end", append([]twi.NodeOption{twi.Data("slot", "bubble-content")}, children...))
}

func BubbleReactions(s Side, a Align, children ...twi.NodeOption) twi.Node {
	along := map[Align]string{AlignStart: "left-1", AlignCenter: "left-1/2 -translate-x-1/2", AlignEnd: "right-1"}
	if s == SideRight || s == SideLeft {
		along = map[Align]string{AlignStart: "top-0", AlignCenter: "top-1/2 -translate-y-1/2", AlignEnd: "bottom-0"}
	}
	return part("absolute z-10 flex flex-row w-fit shrink-0 items-center justify-center gap-1 rounded-full bg-muted px-1 "+pick("bubble reactions", s, map[Side]string{
		SideBottom: "-bottom-1",
		SideTop:    "-top-1",
		SideRight:  "left-full",
		SideLeft:   "right-full",
	})+" "+pick("bubble reactions", a, along), children)
}
