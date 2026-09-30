package ui

import "github.com/twind-dev/twind/twi"

func Alert(v Variant, children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col w-full rounded-lg border bg-card px-1 "+pick("alert", v, map[Variant]string{
		Default:     "text-card-foreground",
		Destructive: "text-destructive",
	}), children)
}

func AlertTitle(children ...twi.NodeOption) twi.Node {
	return part("font-medium", children)
}

func AlertDescription(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}
