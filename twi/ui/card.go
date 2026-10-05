package ui

import "github.com/pehcastro/twind/twi"

func Card(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col gap-1 rounded-xl border bg-card text-card-foreground shadow-sm", children)
}

func CardHeader(children ...twi.NodeOption) twi.Node {
	return part("grid auto-rows-min items-start px-2 has-data-[slot=card-action]:grid-cols-[1fr_auto] has-data-[slot=card-action]:gap-x-1 has-data-[slot=card-description]:grid-rows-[auto_auto]", children)
}

func CardTitle(children ...twi.NodeOption) twi.Node {
	return part("font-semibold", children)
}

func CardDescription(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}

func CardAction(children ...twi.NodeOption) twi.Node {
	return part("col-start-2 row-span-2 row-start-1 self-start justify-self-end", append([]twi.NodeOption{twi.Data("slot", "card-action")}, children...))
}

func CardContent(children ...twi.NodeOption) twi.Node {
	return part("px-2", children)
}

func CardFooter(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center px-2", children)
}
