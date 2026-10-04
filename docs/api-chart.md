# twi/chart

Every exported name in `twi/chart`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## Chart

type Kind uint8\
const Bar Kind\
const Line Kind\
const Area Kind\
type Series struct\
Series.Label string\
Series.Color theme.Token\
Series.Values \[\]float64\
type Chart struct\
func New(rt \*twi.Runtime) \*Chart\
Chart.Kind Kind\
Chart.Stacked bool\
Chart.Labels \[\]string\
Chart.Series \[\]Series\
Chart.Width int\
Chart.Height int\
func (\*Chart) Node(options ...twi.NodeOption) twi.Node
