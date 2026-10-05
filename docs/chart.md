# Chart

Bar, line, area, pie, donut, radial, radar and map charts with axes, a legend and the values under the pointer. Where the terminal shows images the plot is drawn in pixels: smooth curves, gradient areas, rounded bars and anti-aliased slices. Elsewhere it is drawn in half blocks.

<Preview name="chart-bar" />

## Usage

```go
c := chart.New(rt)
c.Kind, c.Width, c.Height = chart.Bar, 56, 14
c.Labels = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"}
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

`Step` holds each value until halfway to the next point.

<Preview name="chart-area-step" />

## Interactive

`Brush` adds an overview of every point under the chart. Drag across it to show only that range; click it to show everything again. `From` and `To` set the range from code, for example from a select.

<Preview name="chart-area-interactive" />

## Bars

`Horizontal` lays the bars along the rows with their labels on the left. `Labelled` writes each value at the end of its bar.

<Preview name="chart-bar-horizontal" />

<Preview name="chart-bar-label" />

A series' `Negative` colour draws its values below zero.

<Preview name="chart-bar-negative" />

## Pie and donut

A pie, donut or radial chart takes one series per slice, each with one value, and one label naming what is counted. A donut writes the total and that label in its hole. The slice under the pointer grows and shows its value.

<Preview name="chart-pie" />

<Preview name="chart-donut" />

## Radial

Each series is a ring, the largest value a full turn.

<Preview name="chart-radial" />

## Radar

Each label is an axis and each series a filled shape. A radar needs three labels or more.

<Preview name="chart-radar" />

## Dither

`Dither` draws every fill as an ordered dither of solid dots, on any kind. Without images the dither runs on half blocks.

<Preview name="chart-dither" />

## Map

A map colours each country by its value, stronger for larger values; a country with no value is muted. The labels are ISO 3166 alpha-3 country codes, and the series is one. The pointer over a country names it.

<Preview name="chart-map" />

The country shapes are Natural Earth's 1:110m admin 0 countries, in the public domain, drawn on the Natural Earth projection without Antarctica.

## API reference

<Props of="Chart" />
