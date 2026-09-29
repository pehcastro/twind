package input

type Event interface{ event() }

type Key uint8

const (
	KeyRune Key = iota
	KeyEnter
	KeyEscape
	KeyTab
	KeyBackspace
	KeyDelete
	KeyInsert
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

type Modifiers uint8

const (
	ModShift Modifiers = 1 << iota
	ModAlt
	ModCtrl
	ModMeta
)

type KeyEvent struct {
	Key       Key
	Modifiers Modifiers
	Rune      rune
	Repeat    bool
	Release   bool
}

type MouseButton uint8

const (
	MouseNone MouseButton = iota
	MouseLeft
	MouseMiddle
	MouseRight
	MouseWheelUp
	MouseWheelDown
	MouseWheelLeft
	MouseWheelRight
)

type MouseAction uint8

const (
	MousePress MouseAction = iota
	MouseRelease
	MouseMove
	MouseScroll
)

type MouseEvent struct {
	X, Y      int
	Button    MouseButton
	Action    MouseAction
	Modifiers Modifiers
}

type PasteEvent struct{ Text string }

type ResizeEvent struct{ Width, Height int }

type FocusEvent struct{ Focused bool }

type ReplyKind uint8

const (
	ReplyPrimaryAttributes ReplyKind = iota
	ReplySecondaryAttributes
	ReplyMode
	ReplyCursorPosition
	ReplyKeyboardFlags
)

type ReplyEvent struct {
	Kind   ReplyKind
	Params []int
}

func (KeyEvent) event()    {}
func (MouseEvent) event()  {}
func (PasteEvent) event()  {}
func (ResizeEvent) event() {}
func (FocusEvent) event()  {}
func (ReplyEvent) event()  {}
