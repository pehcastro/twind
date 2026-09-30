# Radio Group

A set of choices where only one can be on.

<Preview name="radio-group-demo" />

## Usage

```go
density := ui.NewRadioGroup(rt)
density.Value = "comfortable"
```

```go
density.Node(
	density.Item("default", ui.Label(twi.Text("Default"))),
	density.Item("comfortable", ui.Label(twi.Text("Comfortable"))),
)
```

The group is one stop in the Tab order. The arrows move the choice, and a click picks an item.

## API reference

<Props of="RadioGroup" />
