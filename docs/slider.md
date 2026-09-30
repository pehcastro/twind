# Slider

Picks a value from a range.

<Preview name="slider-demo" />

## Usage

```go
volume := ui.NewSlider(rt)
volume.Value = 33
```

```go
volume.Node()
```

The arrows step by `Step`, PageUp and PageDown by ten steps, Home and End go to `Min` and `Max`.

## API reference

<Props of="Slider" />
