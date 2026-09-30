# Text

Text is a node like any other: `twi.Text("hello")`. It takes its colour, weight and alignment from the classes of the element around it.

<Preview name="text-wrap" />

## Wrapping

Text wraps at word boundaries to the width of its box. A word longer than the line breaks only where you allow it: `break-words` breaks it when it cannot fit, `break-all` breaks anywhere.

| Class | What it does |
| --- | --- |
| (none) | wraps at spaces and collapses runs of spaces |
| `whitespace-nowrap` | keeps one line and lets it overflow |
| `whitespace-pre` | keeps every space and newline as written |
| `whitespace-pre-wrap` | keeps spaces and newlines, and still wraps |
| `truncate` | one line, cut with `…` at the edge of the box |

## Width is measured in cells

A letter takes one cell, and most CJK characters and emoji take two. Twind measures with a versioned width table rather than byte or rune counts, so a line of `漢字` or `🚀` wraps, aligns and selects whole, never half a glyph. Headless runs use the same table, so the frames are the same on every machine.

## Style

`font-bold`, `italic`, `underline` and `line-through` map to the terminal's own attributes. `text-left`, `text-center` and `text-right` align within the box. Colours come from classes such as `text-muted-foreground` or `text-red-500`, see [Theming](theming.md).

## Text is data

Every string is cleaned before it reaches the terminal. Escape sequences, control bytes and bidirectional overrides in a string are shown inert, never obeyed, so printing a file name or a log line cannot move the cursor, clear the screen or change the title.

```go
twi.Text(untrusted)
```

is always safe.
