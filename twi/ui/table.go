package ui

import "github.com/pehcastro/twind/twi"

func Table(children ...twi.NodeOption) twi.Node {
	return part("relative w-full overflow-x-auto", []twi.NodeOption{part("flex flex-col w-full", children)})
}

func TableHeader(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col", children)
}

func TableBody(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col", children)
}

func TableFooter(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col bg-muted/50 font-medium", children)
}

func TableRow(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row items-center border-b hover:bg-muted/50", children)
}

func TableHead(children ...twi.NodeOption) twi.Node {
	return part("flex-1 px-1 text-left font-medium text-foreground", children)
}

func TableCell(children ...twi.NodeOption) twi.Node {
	return part("flex-1 px-1", children)
}

func TableCaption(children ...twi.NodeOption) twi.Node {
	return part("mt-1 text-center text-muted-foreground", children)
}
