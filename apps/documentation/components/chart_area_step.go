package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/chart"
	"github.com/twind-dev/twind/twi/theme"
)

func ChartAreaStep(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Step, c.Width, c.Height = chart.Area, true, 56, 14
	c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	c.Series = []chart.Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73, 209, 214}}}
	return func() twi.Node { return c.Node() }
}
