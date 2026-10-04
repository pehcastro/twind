# Native Select

A compact field that opens its options in a list, like a browser's select.

<Preview name="native-select-demo" />

## Usage

```go
status := ui.NewNativeSelect(rt)
status.Options, status.Value = []string{"Todo", "In Progress", "Done"}, "Todo"
```

```go
status.Node(twi.Class("w-30"))
```

A click anywhere on the field opens the list above the page, flipped above the field when there is no room below. Enter, Space, F4 and Alt+Down open it from the keyboard; in the list the arrows move, Enter or Space chooses and Escape closes. With the list closed, the arrows, Home, End and a typed letter change the value in place. For grouped items, labels or custom rows, use a [Select](select.md).

## API reference

<Props of="NativeSelect" />
