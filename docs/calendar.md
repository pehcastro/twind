# Calendar

A month of days to pick a date from, always six weeks tall so it keeps its size as the month turns.

<Preview name="calendar-demo" />

## Usage

```go
calendar := ui.NewCalendar(rt)
calendar.Today = time.Now()
calendar.OnSelect = func(day time.Time) { due = day }
```

```go
calendar.Node(twi.Class("rounded-lg border"))
```

The arrows move by a day and a week, PageUp and PageDown by a month, Shift with them by a year, and Home and End to the ends of the week. Enter or a click picks the day. The chevrons turn the month.

The demo fixes `Today` to a date so its frames are the same on every run; a program sets it from the clock.

## API reference

<Props of="Calendar" />
