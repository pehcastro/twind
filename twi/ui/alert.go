package ui

import "github.com/pehcastro/twind/twi"

type AlertVariant uint8

const (
	AlertDefault AlertVariant = iota
	AlertDestructive
)

func Alert(v AlertVariant, children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col w-full rounded-lg border bg-card px-2 py-1 "+pick("alert", v, map[AlertVariant]string{
		AlertDefault:     "text-card-foreground",
		AlertDestructive: "text-destructive",
	}), children)
}

func AlertTitle(children ...twi.NodeOption) twi.Node {
	return part("font-medium", children)
}

func AlertDescription(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}
