# twi

Every exported name in `twi`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## events.go

type Event = events.Event\[\*runtime.Elem\]\
type Ref = runtime.Ref

## Option

type Option interface\
func Backend(t Terminal, c Clock) Option\
type RenderOption interface\
func Width(cells int) RenderOption\
type Setting interface\
func ColorProfile(p color.Profile) Setting\
func Graphics(mode terminal.Graphics) Setting\
func Styles(sheet style.Sheet) Setting\
func Theme(t theme.Theme) Setting\
Setting embeds Option\
Setting embeds RenderOption\
type Terminal interface\
Terminal embeds io.Writer\
Terminal.Events func() \<-chan input.Event\
Terminal.Size func() (width, height int, err error)\
Terminal.Capabilities func() terminal.Capabilities\
Terminal.Exit func() error\
type Clock interface\
Clock.Now func() time.Time\
Clock.After func(d time.Duration) \<-chan time.Time

## run.go

type Timer = runtime.Timer\
type Runtime struct\
func New(opts ...Option) \*Runtime\
func (\*Runtime) After(d time.Duration, fn func()) \*Timer\
func (\*Runtime) Clicks() int\
func (\*Runtime) ContentBox(e \*Event) image.Rectangle\
func (\*Runtime) Copy(text string)\
func (\*Runtime) Dispatch(f func())\
func (\*Runtime) Focus(key string)\
func (\*Runtime) HideFocusRings()\
func (\*Runtime) Invalidate()\
func (\*Runtime) Now() time.Time\
func (\*Runtime) Quit()\
func (\*Runtime) Remember(key string, save func() string, restore func(string))\
func (\*Runtime) Run(app func() Node) error\
func (\*Runtime) ScrollIntoView(key string)\
func (\*Runtime) SetTheme(t theme.Theme)\
func (\*Runtime) Theme() theme.Theme\
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
func OnBlur(handler func(\*Event)) NodeOption\
func OnClick(handler func(\*Event)) NodeOption\
func OnFocus(handler func(\*Event)) NodeOption\
func OnFocusOutside(handler func(\*Event)) NodeOption\
func OnHotkey(handler func(\*Event)) NodeOption\
func OnKey(handler func(\*Event)) NodeOption\
func OnKeyDown(handler func(\*Event)) NodeOption\
func OnPaste(handler func(text string)) NodeOption\
func OnPointerDown(handler func(\*Event)) NodeOption\
func OnPointerDownOutside(handler func(\*Event)) NodeOption\
func OnPointerEnter(handler func(\*Event)) NodeOption\
func OnPointerLeave(handler func(\*Event)) NodeOption\
func OnPointerMove(handler func(\*Event)) NodeOption\
func OnPointerUp(handler func(\*Event)) NodeOption\
func OnScroll(handler func(offset image.Point)) NodeOption\
func OnWidth(handler func(contentWidth int)) NodeOption\
func Tag(element style.Element) NodeOption\
func TopLayer() NodeOption\
func RenderString(node Node, opts ...RenderOption) (string, error)\
func Render(w io.Writer, node Node, opts ...RenderOption) (err error)
