package blocks

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

type figures struct {
	revenue, change string
	goal            int
}

func DashboardCard(rt *twi.Runtime) func() twi.Node {
	period := ui.NewTabs(rt)
	byPeriod := map[string]figures{
		"month": {"$15,231.89", "+20.1% from last month", 72},
		"year":  {"$182,904.10", "+12.4% from last year", 88},
	}
	stat := func(title, value, change string) twi.Node {
		return ui.Card(twi.Class("flex-1 py-1"),
			ui.CardHeader(
				ui.CardDescription(twi.Text(title)),
				ui.CardTitle(twi.Text(value)),
				ui.CardAction(ui.Badge(ui.Outline, twi.Text("↗"))),
			),
			ui.CardFooter(twi.Class("text-muted-foreground"), twi.Text(change)),
		)
	}
	return func() twi.Node {
		shown, ok := byPeriod[period.Value]
		if !ok {
			shown = byPeriod["month"]
		}
		return twi.Element(twi.Class("flex flex-col w-full gap-1"),
			period.Node(period.List(period.Trigger("month", twi.Text("This month")), period.Trigger("year", twi.Text("This year")))),
			twi.Element(twi.Class("flex flex-row gap-2"),
				stat("Total revenue", shown.revenue, shown.change),
				stat("Subscriptions", "+2,350", "+180.1% from last month"),
			),
			twi.Element(twi.Class("flex flex-row items-center gap-2"),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text("Goal")),
				ui.Progress(shown.goal),
				twi.Element(twi.Class("text-muted-foreground"), twi.Text(strconv.Itoa(shown.goal)+"%")),
			),
		)
	}
}
