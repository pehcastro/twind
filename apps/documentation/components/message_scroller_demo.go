package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func MessageScrollerDemo(rt *twi.Runtime) func() twi.Node {
	scroller, sent := ui.NewMessageScroller(rt), 10
	return func() twi.Node {
		items := []twi.NodeOption{twi.Class("px-1"), scroller.Item(ui.Marker(ui.Ruled, ui.MarkerContent(twi.Text("Today"))))}
		for i := 1; i <= sent; i++ {
			a, v := ui.Start, ui.Muted
			if i%2 == 0 {
				a, v = ui.End, ui.Default
			}
			items = append(items, scroller.Item(ui.Message(a, ui.MessageContent(ui.Bubble(v, a, ui.BubbleContent(twi.Text("Message "+strconv.Itoa(i))))))))
		}
		return twi.Element(twi.Class("flex flex-col w-56 gap-1"),
			scroller.Node(twi.Class("h-12 rounded-lg border"), scroller.Viewport(items...), scroller.Button()),
			ui.Button(ui.Outline, ui.SizeSM, twi.OnClick(func(*twi.Event) {
				sent++
				rt.Invalidate()
			}), twi.Text("Send a message")),
		)
	}
}
