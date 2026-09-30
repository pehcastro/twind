package ui

import "github.com/twind-dev/twind/twi"

func Breadcrumb(children ...twi.NodeOption) twi.Node {
	return part("", children)
}

func BreadcrumbList(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1 text-muted-foreground", children)
}

func BreadcrumbItem(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}

func BreadcrumbLink(children ...twi.NodeOption) twi.Node {
	return part("hover:text-foreground", children)
}

func BreadcrumbPage(children ...twi.NodeOption) twi.Node {
	return part("font-normal text-foreground", children)
}

func BreadcrumbSeparator(children ...twi.NodeOption) twi.Node {
	if len(children) == 0 {
		children = []twi.NodeOption{twi.Text("›")}
	}
	return part("", children)
}

func BreadcrumbEllipsis(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row w-3 items-center justify-center", append([]twi.NodeOption{twi.Text("…")}, children...))
}
