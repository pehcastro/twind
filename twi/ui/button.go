package ui

import "github.com/twind-dev/twind/twi"

type onPress struct {
	twi.NodeOption
	act func()
}

func OnPress(act func()) twi.NodeOption {
	return onPress{twi.OnClick(func(*twi.Event) { act() }), act}
}

func Button(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return buttonWith(button(v, s, idleRing(v)+" "+focusRing), children)
}

func buttonWith(classes string, children []twi.NodeOption) twi.Node {
	options := []twi.NodeOption{twi.Focusable()}
	for _, c := range children {
		if p, ok := c.(onPress); ok {
			options = append(options, twi.OnKeyDown(func(e *twi.Event) {
				if !e.Key.Release && press(e.Key) {
					e.PreventDefault()
					e.StopPropagation()
					p.act()
				}
			}))
		}
	}
	return part(classes, append(options, children...))
}

func idleRing(v Variant) string {
	if v == Outline {
		return "shadow-[0_0_0_1px_var(--color-border)]"
	}
	return ""
}

func button(v Variant, s Size, ring string) string {
	return "flex flex-row shrink-0 items-center justify-center gap-1 rounded-md font-medium select-none disabled:opacity-50 " +
		pick("button", v, map[Variant]string{
			Default:     "bg-primary text-primary-foreground hover:bg-primary/90",
			Destructive: "bg-destructive text-white hover:bg-destructive/90 dark:bg-destructive/60",
			Outline:     "bg-background hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:hover:bg-input/50",
			Secondary:   "bg-secondary text-secondary-foreground hover:bg-secondary/80",
			Ghost:       "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
			Link:        "text-primary hover:underline",
		}) + " " +
		pick("button", s, map[Size]string{
			SizeDefault: "px-2",
			SizeXS:      "px-1",
			SizeSM:      "px-2",
			SizeLG:      "px-3",
			SizeIcon:    "w-3",
		}) + " " + ring
}
