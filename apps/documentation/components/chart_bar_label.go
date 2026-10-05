package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/chart"
	"github.com/pehcastro/twind/twi/theme"
)

func ChartBarLabel(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Labelled, c.Width, c.Height = chart.Bar, true, 56, 14
	c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
	c.Series = []chart.Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73, 209, 214}}}
	return func() twi.Node { return c.Node() }
}
