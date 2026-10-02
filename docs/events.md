# Events and focus

Handlers are node options, next to the classes. Events bubble from the element under the pointer or in focus up to the root, as in the DOM.

<Preview name="events-demo" />

Tab into the boxes above, press **+** or click one.

## Handlers

| Option | Runs on |
| --- | --- |
| `twi.OnClick(func(*twi.Event))` | a click, or Enter or Space on the focused element |
| `twi.OnKeyDown(func(*twi.Event))` | a key while the element or a child has focus; the key is in `e.Key` |
| `twi.OnKey(func(input.KeyEvent))` | every key the focused element did not take, wherever the focus is |
| `twi.OnHotkey(func(input.KeyEvent) bool)` | every key before the focused element sees it; return true to take it, as the Sidebar's Ctrl+B and the Command palette's Ctrl+K do |
| `twi.OnPaste(func(string))` | a paste, on the focused element or the nearest parent with a handler |
| `twi.OnWidth(func(int))` | its content width after each layout, so text can wrap to it |
| `twi.OnFocus(func())`, `twi.OnBlur(func())` | the element gaining or losing focus |
| `twi.OnPointerEnter(func())`, `twi.OnPointerLeave(func())` | the pointer moving over it and away |
| `twi.OnPointerDown(func(*twi.Event))` | a button pressed over it; the button is in `e.Mouse` |
| `twi.OnPointerDownOutside(func())` | a press anywhere outside it, to close a popup |
| `twi.OnFocusOutside(func())` | focus moving somewhere outside it |

`e.StopPropagation()` stops the event from reaching the parents, and `e.PreventDefault()` stops what the runtime would do next, such as scrolling on an arrow key. `e.Target()` is the element the event started on and `e.Current()` the one whose handler is running.

## Redrawing

A handler changes your own state, then calls `rt.Invalidate()`. The runtime calls your render function again before the next frame. The components in twi/ui do this for you.

```go
count := 0
button := ui.Button(ui.Default, ui.SizeDefault, twi.OnClick(func(*twi.Event) {
	count++
	rt.Invalidate()
}), twi.Text("Add one"))
```

From another goroutine, use `rt.Dispatch(func() { ... })` to run the change on the runtime's goroutine, or keep the value in a `twi.NewSignal`, whose `Set` redraws.

## Focus

`twi.Focusable()` puts an element in the Tab order, which follows the tree. Tab and Shift+Tab move through it, and the focused element is scrolled into view. `twi.AutoFocus()` focuses an element when it first appears, and `twi.Disabled()` takes it out of the order and out of every event.

`focus-visible:` styles show only after a key, as in a browser: a click focuses without them, except on a text field (`input` or `textarea` tag), which shows them on any focus. Call `rt.HideFocusRings()` once to turn every focus-visible style off.

`twi.FocusScope()` traps Tab inside an element, as a dialog does, and gives focus back to where it was when the element goes away. `twi.Key("name")` gives an element a stable identity, so its focus and scroll offset survive when the list around it changes.

## Keys

`e.Key.Key` is the key, such as `input.KeyEnter`, `input.KeyArrowDown` or `input.KeyRune` for a character in `e.Key.Rune`. `e.Key.Modifiers` holds `input.ModCtrl`, `input.ModShift` and the rest. On Windows the same events come from the console's input records, so a program handles one set of keys everywhere.
