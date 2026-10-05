package ui

import (
	"slices"

	"github.com/pehcastro/twind/twi"
)

func Pagination(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row w-full justify-center", children)
}

func PaginationContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}

func PaginationItem(children ...twi.NodeOption) twi.Node {
	return part("", children)
}

func PaginationLink(children ...twi.NodeOption) twi.Node {
	v := ButtonGhost
	if slices.ContainsFunc(children, func(o twi.NodeOption) bool { a, ok := o.(active); return ok && a.on }) {
		v = ButtonOutline
	}
	return Button(v, ButtonSizeIcon, children...)
}

func PaginationPrevious(children ...twi.NodeOption) twi.Node {
	return Button(ButtonGhost, ButtonSizeDefault, append([]twi.NodeOption{twi.Class("gap-1 px-1 sm:pl-1"), icon("‹", ""), part("hidden sm:block", []twi.NodeOption{twi.Text("Previous")})}, children...)...)
}

func PaginationNext(children ...twi.NodeOption) twi.Node {
	return Button(ButtonGhost, ButtonSizeDefault, append([]twi.NodeOption{twi.Class("gap-1 px-1 sm:pr-1"), part("hidden sm:block", []twi.NodeOption{twi.Text("Next")}), icon("›", "")}, children...)...)
}

func PaginationEllipsis(children ...twi.NodeOption) twi.Node {
	return BreadcrumbEllipsis(children...)
}
