# Carousel

Slides shown one at a time, with previous and next buttons.

<Preview name="carousel-demo" />

## Usage

```go
carousel := ui.NewCarousel(rt)
```

```go
carousel.Node(twi.Class("w-24 h-8"),
	carousel.Content(
		carousel.Item(twi.Text("1")),
		carousel.Item(twi.Text("2")),
	),
	carousel.Previous(),
	carousel.Next(),
)
```

With the carousel focused, Left and Right turn it, and it wraps at both ends. Each new slide slides in from the side it came from.

## API reference

<Props of="Carousel" />
