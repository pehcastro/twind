package ui

import "github.com/pehcastro/twind/twi"

func Pagination(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row w-full justify-center", children)
}

func PaginationContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center gap-1", children)
}

func PaginationItem(children ...twi.NodeOption) twi.Node {
	return part("", children)
}

func PaginationLink(isActive bool, children ...twi.NodeOption) twi.Node {
	if isActive {
		return Button(Outline, SizeIcon, children...)
	}
	return Button(Ghost, SizeIcon, children...)
}

func PaginationPrevious(children ...twi.NodeOption) twi.Node {
	return Button(Ghost, SizeDefault, append([]twi.NodeOption{twi.Class("gap-1 px-1 sm:pl-1"), icon("‹", ""), part("hidden sm:block", []twi.NodeOption{twi.Text("Previous")})}, children...)...)
}

func PaginationNext(children ...twi.NodeOption) twi.Node {
	return Button(Ghost, SizeDefault, append([]twi.NodeOption{twi.Class("gap-1 px-1 sm:pr-1"), part("hidden sm:block", []twi.NodeOption{twi.Text("Next")}), icon("›", "")}, children...)...)
}

func PaginationEllipsis(children ...twi.NodeOption) twi.Node {
	return BreadcrumbEllipsis(children...)
}
