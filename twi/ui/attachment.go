package ui

import "github.com/pehcastro/twind/twi"

type Upload uint8

const (
	UploadDone Upload = iota
	UploadIdle
	UploadUploading
	UploadProcessing
	UploadFailed
)

func Attachment(u Upload, o Orientation, children ...twi.NodeOption) twi.Node {
	state := pick("attachment", u, map[Upload]string{UploadDone: "done", UploadIdle: "idle", UploadUploading: "uploading", UploadProcessing: "processing", UploadFailed: "error"})
	return part("group/attachment relative flex w-fit max-w-full min-w-0 shrink-0 gap-1 rounded-xl border bg-card px-1 text-card-foreground has-focus-visible:border-ring data-[state=error]:border-destructive/30 data-[state=idle]:border-dashed "+pick("attachment", o, map[Orientation]string{
		Horizontal: "flex-row min-w-20 items-start",
		Vertical:   "flex-col w-15",
	}), append([]twi.NodeOption{twi.Data("slot", "attachment"), twi.Data("state", state), twi.Data("orientation", pick("attachment", o, map[Orientation]string{Horizontal: "horizontal", Vertical: "vertical"}))}, children...))
}

type AttachmentMediaVariant uint8

const (
	AttachmentMediaIcon AttachmentMediaVariant = iota
	AttachmentMediaImage
)

func AttachmentMedia(v AttachmentMediaVariant, children ...twi.NodeOption) twi.Node {
	return part("relative flex shrink-0 items-center justify-center overflow-hidden bg-muted text-foreground group-data-[orientation=vertical]/attachment:w-full group-data-[orientation=vertical]/attachment:h-3 group-data-[state=error]/attachment:bg-destructive/10 group-data-[state=error]/attachment:text-destructive "+pick("attachment media", v, map[AttachmentMediaVariant]string{
		AttachmentMediaIcon:  "h-1 w-3 rounded-sm",
		AttachmentMediaImage: "h-2 w-4 rounded-lg opacity-60 group-data-[state=done]/attachment:opacity-100 group-data-[state=idle]/attachment:opacity-100",
	}), append([]twi.NodeOption{twi.Data("slot", "attachment-media")}, children...))
}

func AttachmentContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col max-w-full min-w-0 flex-1", children)
}

func AttachmentTitle(children ...twi.NodeOption) twi.Node {
	return part("max-w-full min-w-0 truncate font-medium group-data-[state=processing]/attachment:text-muted-foreground group-data-[state=uploading]/attachment:text-muted-foreground", children)
}

func AttachmentDescription(children ...twi.NodeOption) twi.Node {
	return part("max-w-full min-w-0 truncate text-muted-foreground group-data-[state=error]/attachment:text-destructive/80", children)
}

func AttachmentActions(children ...twi.NodeOption) twi.Node {
	return part("relative z-20 flex flex-row shrink-0 items-center group-data-[orientation=vertical]/attachment:absolute group-data-[orientation=vertical]/attachment:top-0 group-data-[orientation=vertical]/attachment:right-1", children)
}

func AttachmentAction(children ...twi.NodeOption) twi.Node {
	return Button(ButtonGhost, ButtonSizeXS, children...)
}

func AttachmentTrigger(children ...twi.NodeOption) twi.Node {
	return part("absolute inset-0 z-10 rounded-xl "+focusRing, append([]twi.NodeOption{twi.Focusable()}, children...))
}

func AttachmentGroup(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row min-w-0 gap-2 overflow-x-auto *:data-[slot=attachment]:flex-none", children)
}
