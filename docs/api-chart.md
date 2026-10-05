# twi/chart

Every exported name in `twi/chart`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## Chart

type Kind uint8\
const Bar Kind\
const Line Kind\
const Area Kind\
const Pie Kind\
const Donut Kind\
const Radial Kind\
const Radar Kind\
const Map Kind\
type Series struct\
Series.Label string\
Series.Color theme.Token\
Series.Negative theme.Token\
Series.Values \[\]float64\
type Chart struct\
func New(rt \*twi.Runtime) \*Chart\
Chart.Kind Kind\
Chart.Stacked bool\
Chart.Horizontal bool\
Chart.Step bool\
Chart.Labelled bool\
Chart.Dither bool\
Chart.Brush bool\
Chart.From int\
Chart.To int\
Chart.Labels \[\]string\
Chart.Series \[\]Series\
Chart.Width int\
Chart.Height int\
func (\*Chart) Node(options ...twi.NodeOption) twi.Node
