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
	Stack    = "flex flex-col md:flex-row gap-x-1"
	Padded   = "md:p-2 lg:p-4"
	Footer   = "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end"
	Button   = "px-2 bg-zinc-800"
	Dialog   = "border p-1 w-full"
	Upward   = "flex flex-col-reverse h-4"
	Leftward = "flex flex-row-reverse w-10 gap-x-1"
	Wrapped  = "flex flex-row-reverse flex-wrap"
	Middle   = "justify-center"
)
