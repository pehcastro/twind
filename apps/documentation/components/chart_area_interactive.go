package components

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/chart"
	"github.com/pehcastro/twind/twi/theme"
)

func ChartAreaInteractive(rt *twi.Runtime) func() twi.Node {
	c := chart.New(rt)
	c.Kind, c.Stacked, c.Brush, c.Width, c.Height = chart.Area, true, true, 56, 17
	desktop, mobile := []float64{}, []float64{}
	for day := range 60 {
		c.Labels = append(c.Labels, []string{"May ", "Jun "}[day/30]+strconv.Itoa(day%30+1))
		desktop = append(desktop, float64(160+(day*37)%180))
		mobile = append(mobile, float64(90+(day*53)%140))
	}
	c.From, c.To = 30, 60
	c.Series = []chart.Series{
		{Label: "Desktop", Color: theme.Chart1, Values: desktop},
		{Label: "Mobile", Color: theme.Chart2, Values: mobile},
	}
	return func() twi.Node { return c.Node() }
}
