# Chart

Bar, line and area charts drawn in cells, with a y axis, a legend and the values of a point under the pointer.

<Preview name="chart-bar" />

## Usage

```go
c := chart.New(rt)
c.Kind, c.Width, c.Height = chart.Bar, 56, 14
c.Labels = []string{"January", "February", "March", "April", "May", "June"}
c.Series = []chart.Series{
	{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73, 209, 214}},
	{Label: "Mobile", Color: theme.Chart2, Values: []float64{80, 200, 120, 190, 130, 140}},
}
```

Render `c.Node()` in the view. Each series takes one of the five chart colours of the theme, so the chart follows a theme change. Every series needs one value per label, and the chart needs room for its axis and legend inside `Width` and `Height`.

## Line

<Preview name="chart-line" />

## Area

`Stacked` puts each series on top of the one before it, for areas and bars.

<Preview name="chart-area" />

## API reference

<Props of="Chart" />
