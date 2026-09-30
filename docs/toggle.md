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

A click, Space or Enter flips it. `Variant` is `ui.Default` or `ui.Outline`, `Size` is small, default or large.

## API reference

<Props of="Toggle" />
