# Textarea

A text field of several lines.

<Preview name="textarea-demo" />

## Usage

```go
message := ui.NewTextarea(rt)
message.Placeholder = "Type your message here."
message.OnSubmit = func(text string) { send(text) }
```

```go
ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Your message")), message.Node())
```

It edits like a chat prompt. Shift+Enter or Ctrl+J breaks a line, and Enter sends: with `OnSubmit` set, Enter hands it the trimmed text, clears the field and keeps the text in the history; empty text sends nothing. Without `OnSubmit`, Enter goes on to the page. Long lines wrap at word boundaries to the field's width, and rewrap when the terminal resizes; the field grows with its rows from four to eight, then scrolls with the caret. `message.Value()` returns the text.

A paste keeps its line breaks, turns a tab into four spaces and is one undo step. A paste of 160 characters or more shows as a `[Text N characters]` chip; `OnSubmit` gets the pasted text in its place, and the history keeps it.

## Keys

| key | does |
|---|---|
| Enter | sends the text to `OnSubmit` |
| Shift+Enter, Ctrl+J | breaks the line |
| Left, Ctrl+B / Right, Ctrl+F | move by a character, across lines |
| Ctrl+Left, Ctrl+Right | move by a word; a word is a run of characters without spaces |
| Up, Down, Ctrl+P, Ctrl+N | move by a wrapped row, keeping the column; on the first or last row the caret stays |
| Up, Down on one line of text | walk the sent history; past the newest, the unsent draft comes back |
| Home, Ctrl+A / End, Ctrl+E | the start or end of the line |
| Ctrl+Home / Ctrl+End | the start or end of the text |
| Backspace, Ctrl+H / Delete, Ctrl+D | delete a character |
| Ctrl+W, Ctrl+Backspace / Ctrl+Delete | delete a word, within the line; at the line's start or end, join the neighbouring line |
| Ctrl+K | delete to the line end; at the end, join the line below |
| Ctrl+U | delete to the line start; at the start, join the line above |
| Ctrl+T | swap the two characters before the caret |
| Shift with any move | selects |
| Ctrl+G | selects everything |
| Ctrl+Shift+C | copies the selection |
| Ctrl+Z / Ctrl+Y, Ctrl+Shift+Z | undo / redo |

A page's hotkeys come first, as in tofu: with a [Sidebar](sidebar.md) on the page Ctrl+B toggles it, and with a [Command](command.md) palette Ctrl+K opens it.

## API reference

<Props of="Textarea" />
