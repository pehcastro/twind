package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func MotionPresence(rt *twi.Runtime) func() twi.Node {
	popover, loading := ui.NewPopover(rt), false
	load := twi.OnClick(func(*twi.Event) {
		loading = !loading
		rt.Invalidate()
	})
	return func() twi.Node {
		rows := twi.Element(twi.Class("text-muted-foreground"), twi.Text("loaded: the pulse and its frames are gone"))
		if loading {
			rows = ui.Skeleton(twi.Class("h-1 w-full animate-pulse"))
		}
		return twi.Element(twi.Class("flex flex-row w-full h-9 items-start gap-2"),
			popover.Node(
				popover.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Fade and zoom")),
				popover.Content(twi.Text("fade-in-0 zoom-in-95 on open, the same played back on close")),
			),
			twi.Element(twi.Class("flex flex-col flex-1 gap-1"),
				ui.Button(ui.Outline, ui.SizeDefault, load, twi.Text("Toggle loading")),
				rows,
			),
		)
	}
}
