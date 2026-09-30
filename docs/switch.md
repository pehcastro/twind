# Switch

A control that toggles between on and off.

<Preview name="switch-demo" />

## Usage

```go
airplane := ui.NewSwitch(rt)
airplane.OnChange = func(on bool) { setAirplane(on) }
```

```go
ui.Field(ui.Horizontal, airplane.Node(), ui.FieldLabel(twi.Text("Airplane mode")))
```

A click, Space or Enter flips it.

## API reference

<Props of="Switch" />
