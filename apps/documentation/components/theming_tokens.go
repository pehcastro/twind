package components

import "github.com/pehcastro/twind/twi"

func ThemingTokens(*twi.Runtime) func() twi.Node {
	swatch := func(classes, token string) twi.Node {
		return twi.Element(twi.Class("flex flex-col flex-1 items-center gap-1"),
			twi.Element(twi.Class("h-2 w-full rounded-md border "+classes)),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text(token)),
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row w-full gap-1"),
			swatch("bg-background", "background"),
			swatch("bg-card", "card"),
			swatch("bg-muted", "muted"),
			swatch("bg-accent", "accent"),
			swatch("bg-primary", "primary"),
			swatch("bg-secondary", "secondary"),
			swatch("bg-destructive", "destructive"),
		)
	}
}
