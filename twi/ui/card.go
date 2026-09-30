package ui

import "github.com/twind-dev/twind/twi"

func Card(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col gap-1 rounded-xl border bg-card text-card-foreground shadow-sm", children)
}

func CardHeader(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col px-2", children)
}

func CardTitle(children ...twi.NodeOption) twi.Node {
	return part("font-semibold", children)
}

func CardDescription(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}

func CardAction(children ...twi.NodeOption) twi.Node {
	return part("absolute top-0 right-2", children)
}

func CardContent(children ...twi.NodeOption) twi.Node {
	return part("px-2", children)
}

func CardFooter(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center px-2", children)
}
