package sheet

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const (
	Page     = "bg-zinc-950 text-zinc-100"
	Truncate = "w-10 truncate"
	Clip     = "flex w-10 overflow-hidden"
	Nowrap   = "whitespace-nowrap"
	TextClip = "text-clip"
	Ellipsis = "w-10 text-ellipsis"
	Focus    = "p-1 border focus-visible:ring-2"
	Column   = "flex flex-col w-20"
	Centred  = "items-center"
	Row      = "flex h-3"
	Fit      = "w-fit h-fit bg-zinc-800"
	Video    = "w-20 aspect-video bg-zinc-800"
	Wrap     = "flex flex-wrap w-10 gap-x-1"
	Reverse  = "flex flex-wrap-reverse w-10 gap-x-1"
	Shrink0  = "shrink-0"
)
