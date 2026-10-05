# twi/drive

Every exported name in `twi/drive`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## drive.go

type App func(rt \*twi.Runtime) func() twi.Node\
type Option func(\*config)\
func CellPixels(size image.Point) Option\
func Size(width, height int) Option\
func Widths(w text.Widths) Option\
func With(opts ...twi.Option) Option\
type Driver struct\
func New(app App, opts ...Option) \*Driver\
func (\*Driver) Advance(dt time.Duration)\
func (\*Driver) Click(x, y int)\
func (\*Driver) ClickWith(b input.MouseButton, x, y int)\
func (\*Driver) Clipboard() string\
func (\*Driver) Close() error\
func (\*Driver) Down(x, y int)\
func (\*Driver) DownWith(b input.MouseButton, x, y int)\
func (\*Driver) Err() error\
func (\*Driver) Frame() Frame\
func (\*Driver) Hold(m input.Modifiers)\
func (\*Driver) Move(x, y int)\
func (\*Driver) Press(name string)\
func (\*Driver) Resize(width, height int)\
func (\*Driver) Type(s string)\
func (\*Driver) Up(x, y int)\
func (\*Driver) UpWith(b input.MouseButton, x, y int)\
func (\*Driver) Wheel(x, y, notches int)

## screen.go

type Frame struct\
func (Frame) ANSI() string\
func (Frame) At(x, y int) Cell\
func (Frame) Height() int\
func (Frame) Row(y int) \[\]Cell\
func (Frame) Text() string\
func (Frame) Width() int\
type Attr uint8\
const Bold Attr\
const Dim Attr\
const Italic Attr\
const Underline Attr\
const Strikethrough Attr\
const Inverse Attr\
type Width uint8\
const Narrow Width\
const Wide Width\
const Continuation Width\
type Cell struct\
Cell.Grapheme string\
Cell.Fg color.Color\
Cell.Bg color.Color\
Cell.Attr Attr\
Cell.Width Width

## script.go

func RunScript(r io.Reader, app App, out string, opts ...Option) (err error)
