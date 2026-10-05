package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/chart"
	"github.com/pehcastro/twind/twi/theme"
)

func ChartArea(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Stacked, c.Width, c.Height = chart.Area, true, 56, 14
	c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	c.Series = []chart.Series{
		{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73, 209, 214}},
		{Label: "Mobile", Color: theme.Chart2, Values: []float64{80, 200, 120, 190, 130, 140}},
	}
	return func() twi.Node { return c.Node() }
}
