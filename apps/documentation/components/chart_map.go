package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/chart"
	"github.com/pehcastro/twind/twi/theme"
)

func ChartMap(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Width, c.Height = chart.Map, 56, 16
	c.Labels = []string{"USA", "CAN", "MEX", "BRA", "ARG", "GBR", "FRA", "DEU", "ESP", "IND", "CHN", "JPN", "AUS", "ZAF", "NGA"}
	c.Series = []chart.Series{
		{Label: "Visitors", Color: theme.Chart1, Values: []float64{305, 120, 90, 237, 73, 186, 150, 209, 110, 280, 260, 140, 130, 60, 80}},
	}
	return func() twi.Node { return c.Node() }
}
