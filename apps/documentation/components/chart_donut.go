package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/chart"
	"github.com/pehcastro/twind/twi/theme"
)

func ChartDonut(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Width, c.Height = chart.Donut, 56, 14
	c.Labels = []string{"Visitors"}
	c.Series = []chart.Series{
		{Label: "Chrome", Color: theme.Chart1, Values: []float64{275}},
		{Label: "Safari", Color: theme.Chart2, Values: []float64{200}},
		{Label: "Firefox", Color: theme.Chart3, Values: []float64{287}},
		{Label: "Edge", Color: theme.Chart4, Values: []float64{173}},
		{Label: "Other", Color: theme.Chart5, Values: []float64{190}},
	}
	return func() twi.Node { return c.Node() }
}
