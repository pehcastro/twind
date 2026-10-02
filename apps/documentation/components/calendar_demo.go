package components

import (
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func CalendarDemo(rt *twi.Runtime) func() twi.Node {
	calendar := ui.NewCalendar(rt)
	calendar.Today = time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	return func() twi.Node {
		picked := "none"
		if !calendar.Selected.IsZero() {
			picked = calendar.Selected.Format("Monday 2 January")
		}
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			calendar.Node(twi.Class("rounded-lg border shadow-sm")),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("selected: "+picked)),
		)
	}
}
