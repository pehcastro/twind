package ui

import "github.com/pehcastro/twind/twi"

func MessageGroup(children ...twi.NodeOption) twi.Node {
	return part("flex min-w-0 flex-col gap-1", children)
}

func Message(a Alignment, children ...twi.NodeOption) twi.Node {
	return part("group/message relative flex w-full min-w-0 gap-1 "+pick("message", a, map[Alignment]string{
		Start: "flex-row",
		End:   "flex-row-reverse",
	}), append([]twi.NodeOption{twi.Data("slot", "message"), aligned(a)}, children...))
}

func aligned(a Alignment) twi.NodeOption {
	return twi.Data("align", pick("align", a, map[Alignment]string{Start: "start", End: "end"}))
}

func MessageAvatar(children ...twi.NodeOption) twi.Node {
	return part("flex w-fit min-w-2 shrink-0 items-center justify-center self-end overflow-hidden rounded-full bg-muted", children)
}

func MessageContent(children ...twi.NodeOption) twi.Node {
	return part("flex w-full min-w-0 flex-col wrap-break-word group-data-[align=end]/message:items-end", children)
}

func MessageHeader(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row max-w-full min-w-0 items-center px-1 font-medium text-muted-foreground", children)
}

func MessageFooter(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row max-w-full min-w-0 items-center px-1 font-medium text-muted-foreground group-data-[align=end]/message:justify-end", children)
}
