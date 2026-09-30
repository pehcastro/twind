# Spinner

Shows that something is loading.

<Preview name="spinner-demo" />

## Usage

```go
spinner := ui.NewSpinner(rt)
```

```go
ui.Button(ui.Secondary, ui.SizeSM, twi.Disabled(), spinner.Node(), twi.Text("Please wait"))
```

Each spinner turns while it is drawn. Once it is no longer in the tree it stops asking for frames, so a hidden spinner costs nothing.

## API reference

<Props of="Spinner" />
