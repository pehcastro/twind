package ui

import "github.com/twind-dev/twind/twi"

func FieldSet(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col gap-1", children)
}

func FieldLegend(children ...twi.NodeOption) twi.Node {
	return part("mb-1 font-medium", children)
}

func FieldGroup(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col w-full gap-1", children)
}

func Field(o Orientation, children ...twi.NodeOption) twi.Node {
	return part("flex w-full gap-1 "+pick("field", o, map[Orientation]string{
		Vertical:   "flex-col",
		Horizontal: "flex-row items-center",
	}), children)
}

func FieldLabel(children ...twi.NodeOption) twi.Node {
	return Label(children...)
}

func FieldDescription(children ...twi.NodeOption) twi.Node {
	return part("font-normal text-muted-foreground", children)
}

func FieldError(children ...twi.NodeOption) twi.Node {
	return part("font-normal text-destructive", children)
}

func FieldSeparator(children ...twi.NodeOption) twi.Node {
	content := []twi.NodeOption{part("absolute top-0 right-0 left-0 border-t border-border", nil)}
	if len(children) > 0 {
		content = append(content, part("relative bg-background px-1 text-muted-foreground", children))
	}
	return part("relative flex flex-row h-1 justify-center", content)
}
