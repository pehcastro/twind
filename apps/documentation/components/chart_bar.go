package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/chart"
	"github.com/twind-dev/twind/twi/theme"
)

func ChartBar(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Width, c.Height = chart.Bar, 56, 14
	c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	c.Series = []chart.Series{
		{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73, 209, 214}},
		{Label: "Mobile", Color: theme.Chart2, Values: []float64{80, 200, 120, 190, 130, 140}},
	}
	return func() twi.Node { return c.Node() }
}
