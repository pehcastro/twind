# Native Select

A compact field that steps through its options in place, with no list.

<Preview name="native-select-demo" />

## Usage

```go
status := ui.NewNativeSelect(rt)
status.Options, status.Value = []string{"Todo", "In Progress", "Done"}, "Todo"
```

```go
status.Node(twi.Class("w-30"))
```

With it focused, the arrows step through the options, Home and End jump to the ends. When the choices need to be seen at once, use a [Select](select.md).

## API reference

<Props of="NativeSelect" />
