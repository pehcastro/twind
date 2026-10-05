package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/chart"
	"github.com/twind-dev/twind/twi/theme"
)

func ChartBarNegative(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Width, c.Height = chart.Bar, 56, 14
	c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	c.Series = []chart.Series{
		{Label: "Visitors", Color: theme.Chart1, Negative: theme.Chart2, Values: []float64{186, 205, -207, 173, -209, 214}},
	}
	return func() twi.Node { return c.Node() }
}
