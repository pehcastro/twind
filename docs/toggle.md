# Toggle

A button that is either on or off.

<Preview name="toggle-demo" />

## Usage

```go
bold := ui.NewToggle(rt)
bold.OnChange = func(on bool) { setBold(on) }
```

```go
bold.Node(twi.Text("B"))
```

A click, Space or Enter flips it. `Variant` is `ui.ToggleDefault` or `ui.ToggleOutline`, `Size` is `ui.ToggleSizeSM`, `ui.ToggleSizeDefault` or `ui.ToggleSizeLG`.

## API reference

<Props of="Toggle" />
