# twi

Every exported name in `twi`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## events.go

type Event = events.Event\[\*runtime.Elem\]\
type Ref = runtime.Ref\
func NewRef(\*Runtime) \*Ref

## Input

type Input struct\
func NewInput(rt \*Runtime) \*Input\
Input embeds edit.Buffer\
Input.Placeholder string\
Input.CursorClass string\
Input.SelectionClass string\
Input.PlaceholderClass string\
func (\*Input) Node(options ...NodeOption) Node\
Input.Mode edit.Mode\
Input.Wrap int\
Input.Widths text.Widths\
Input.Now func() time.Time\
func (\*Input) Apply(k input.KeyEvent) bool\
func (\*Input) At(row, column int) int\
func (\*Input) Cursor() (row, column int)\
func (\*Input) Drag(at int)\
func (\*Input) Expand(s string) string\
func (\*Input) Insert(s string)\
func (\*Input) Paste(s string)\
func (\*Input) Press(at int, u edit.Unit, extend bool)\
func (\*Input) Remember(s string)\
func (\*Input) Rows() \[\]edit.Row\
func (\*Input) Selection() (start, end int)\
func (\*Input) Set(s string)\
func (\*Input) Value() string

## run.go

type Timer = runtime.Timer\
type Runtime struct\
func New(opts ...RenderOption) \*Runtime\
Runtime embeds \*runtime.Runtime\
func (\*Runtime) Run(app func() Node) error\
func (\*Runtime) SetTheme(t theme.Theme)\
func (\*Runtime) Theme() theme.Theme\
func (\*Runtime) After(d time.Duration, fn func()) \*runtime.Timer\
func (\*Runtime) Clicks() int\
func (\*Runtime) ContentBox(e \*runtime.Elem) image.Rectangle\
func (\*Runtime) Copy(text string) error\
func (\*Runtime) Dispatch(f func())\
func (\*Runtime) Focus(key string) bool\
func (\*Runtime) HideFocusRings()\
func (\*Runtime) Invalidate()\
func (\*Runtime) Quit()\
func (\*Runtime) Remember(key string, save func() string, restore func(string))\
func (\*Runtime) Restyle(apply func())\
func (\*Runtime) ScrollIntoView(key string)\
func (\*Runtime) Viewport() image.Rectangle\
func (\*Runtime) Widths() text.Widths\
type Signal\[T any\] struct\
func NewSignal\[T any\](rt \*Runtime, value T) \*Signal\[T\]\
func (\*Signal\[T\]) Get() T\
func (\*Signal\[T\]) Set(value T)

## twi.go

type Node struct\
func Element(options ...NodeOption) Node\
func Text(s string) Node\
type NodeOption interface\
func At(x, y int) NodeOption\
func AutoFocus() NodeOption\
func Canvas(key uint64, paint func(dst \*image.RGBA, cell image.Point)) NodeOption\
func Class(classes ...string) NodeOption\
func Classes(options \[\]NodeOption) (classes \[\]string, rest \[\]NodeOption)\
func Data(name, value string) NodeOption\
func Dir(d text.Direction) NodeOption\
func Disabled() NodeOption\
func FocusScope() NodeOption\
func Focusable() NodeOption\
func Key(key string) NodeOption\
func MaxSize(width, height int) NodeOption\
func Measure(ref \*Ref) NodeOption\
func MinSize(width, height int) NodeOption\
func NonModalFocusScope() NodeOption\
func OnBlur(handler func()) NodeOption\
func OnClick(handler func(\*Event)) NodeOption\
func OnFocus(handler func()) NodeOption\
func OnFocusOutside(handler func()) NodeOption\
func OnHotkey(handler func(input.KeyEvent) bool) NodeOption\
func OnKey(handler func(input.KeyEvent)) NodeOption\
func OnKeyDown(handler func(\*Event)) NodeOption\
func OnPaste(handler func(text string)) NodeOption\
func OnPointerDown(handler func(\*Event)) NodeOption\
func OnPointerDownOutside(handler func()) NodeOption\
func OnPointerEnter(handler func()) NodeOption\
func OnPointerLeave(handler func()) NodeOption\
func OnPointerMove(handler func(\*Event)) NodeOption\
func OnPointerUp(handler func(\*Event)) NodeOption\
func OnScroll(handler func(offset image.Point)) NodeOption\
func OnWidth(handler func(contentWidth int)) NodeOption\
func Tag(element style.Element) NodeOption\
func TopLayer() NodeOption\
type RenderOption func(\*renderConfig)\
func Backend(b runtime.Backend, c runtime.Clock) RenderOption\
func ColorProfile(p color.Profile) RenderOption\
func Fullscreen() RenderOption\
func Graphics(mode terminal.Graphics) RenderOption\
func NoClipboard() RenderOption\
func Styles(sheet style.Sheet) RenderOption\
func Theme(t theme.Theme) RenderOption\
func Width(cells int) RenderOption\
func RenderString(node Node, opts ...RenderOption) string\
func Render(w io.Writer, node Node, opts ...RenderOption) (err error)
