# Driving

The headless driver runs your app with no terminal: it sends keys, pointer events and time, and hands back the frame the terminal would show. Tests use it to check what a person would see, and the same frames come out on every OS.

## In a test

```go
func TestCounter(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	d := drive.New(App, drive.Size(80, 24), drive.With(twi.Styles(sheet)))
	defer d.Close()
	d.Press("tab")
	d.Press("enter")
	d.Type("hello")
	d.Advance(500 * time.Millisecond)
	if !strings.Contains(d.Frame().Text(), "count 1") {
		t.Errorf("after a press:\n%s", d.Frame().Text())
	}
}
```

`App` is the same function the program runs: `func App(rt *twi.Runtime) func() twi.Node`. The driver gives it a runtime on a fake clock, so timers, transitions and animations advance only when you call `Advance`. `drive.With` passes the options `twi.New` takes, such as `twi.Styles` and `twi.Theme`.

## The driver

| Method | What it does |
| --- | --- |
| `Press(name)` | a key: `enter`, `tab`, `escape`, `up`, `pagedown`, `f10`, a character, or a chord such as `ctrl+k` or `shift+tab` |
| `Type(text)` | each character of the text as a key |
| `Click(x, y)`, `Move(x, y)`, `Wheel(x, y, notches)` | the pointer, at a column and row from 0 |
| `ClickWith(button, x, y)`, `Down`, `Up` | other buttons, and presses held across moves |
| `Resize(w, h)` | a terminal resize |
| `Advance(d)` | moves the fake clock and runs every frame due |
| `Frame()` | the screen: `.Text()` for the characters, `.ANSI()` with colours, `.Cells()` for each cell |
| `Clipboard()` | the last text copied with OSC 52 |
| `Err()`, `Close()` | the first error the app returned, and shutting it down |

## Scripts

The same steps can live in a file and run with `twind drive`, which writes each `frame` as `NAME.txt` and `NAME.ansi`:

```bash
size 80x24
frame start
press tab
press enter
type hello
wait 500ms
frame typed
click 10 3
resize 60x20
frame small
```

| Verb | Argument |
| --- | --- |
| `size` | `WxH`, before anything else |
| `press`, `type` | a key name, or text |
| `click`, `down`, `up` | `X Y`, or `left`, `middle` or `right` then `X Y` |
| `move` | `X Y` |
| `wheel` | `up` or `down`, then `X Y` |
| `wait` | a duration such as `200ms` or `2s` |
| `resize` | `WxH` |
| `frame` | a file name for the frame |

## Why frames and not snapshots of state

A frame is what a person sees. A test that reads the frame breaks when the screen breaks, not when an internal field is renamed. Every page of this documentation is checked that way: a test opens each one through the palette and reads its breadcrumb from the frame.
