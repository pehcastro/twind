# Motion

Twind animates with the same Tailwind classes as the web: transitions between states, and enter and exit animations. It draws frames only while something moves; at rest the runtime sleeps and uses no CPU.

## Transitions

`transition-colors` eases a colour change instead of jumping. `duration-*` sets how long it takes, `ease-*` and `delay-*` shape it. Here the first two cards ease into their hover colour and the third jumps:

<Preview name="motion-transition" />

```go
twi.Class("transition-colors duration-150 hover:bg-accent")
```

A transition runs when a class changes the value: a hover, a focus, a `data-*` attribute or a class you swap in the render function.

## Enter and exit

`animate-in` with `fade-in-0`, `zoom-in-95` or `slide-in-from-*` plays when the element appears. `animate-out` with `fade-out-0`, `zoom-out-95` or `slide-out-to-*` plays when it leaves. The overlays in twi/ui use them: a dialog fades and zooms in over 200 ms and plays the same motion back when it closes; a sheet slides in over 500 ms and out over 300 ms. The component keeps a closing element on screen until its exit has played.

<Preview name="motion-presence" />

## Loops

`animate-pulse` and `animate-spin` repeat until the element goes away. A skeleton that pulses while data loads costs frames only while it is shown; once the rows replace it, the frames stop.

## Frames and the clock

The runtime asks for a frame only when a transition or an animation is running, or when you call `rt.Invalidate()`. Frames are paced, and every motion reads a clock the runtime owns, so a headless run with the fake clock produces the same frames every time; see [Driving](driving.md).
