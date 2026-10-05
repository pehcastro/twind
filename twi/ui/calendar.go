package ui

import (
	"strconv"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

type Calendar struct {
	control
	Month, Selected, Today time.Time
	OnSelect               func(time.Time)
	focus                  time.Time
}

func NewCalendar(rt *twi.Runtime) *Calendar { return &Calendar{control: control{rt: rt}} }

func date(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func monthOf(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC) }

func addMonths(t time.Time, n int) time.Time {
	first := monthOf(t).AddDate(0, n, 0)
	return first.AddDate(0, 0, min(t.Day(), first.AddDate(0, 1, -1).Day())-1)
}

func (c *Calendar) Node(options ...twi.NodeOption) twi.Node {
	if c.Month.IsZero() {
		c.Month = c.Selected
		if c.Month.IsZero() {
			c.Month = c.Today
		}
	}
	c.Month = monthOf(c.Month)
	if !monthOf(c.focus).Equal(c.Month) {
		c.focus = c.Month
		for _, d := range []time.Time{c.Today, c.Selected} {
			if !d.IsZero() && monthOf(d).Equal(c.Month) {
				c.focus = date(d)
			}
		}
	}
	turn := func(glyph string, months int) twi.Node {
		return Button(ButtonGhost, ButtonSizeIcon, twi.OnClick(func(*twi.Event) {
			c.focus = addMonths(c.focus, months)
			c.Month = monthOf(c.focus)
			c.rt.Invalidate()
		}), twi.Text(glyph))
	}
	weekdays := make([]twi.NodeOption, konst.DaysInWeek)
	for i := range weekdays {
		weekdays[i] = part("w-4 text-center text-muted-foreground select-none", []twi.NodeOption{twi.Text(time.Weekday(i).String()[:2])})
	}
	start := c.Month.AddDate(0, 0, -int(c.Month.Weekday()))
	weeks := []twi.NodeOption{part("flex flex-row gap-1", weekdays)}
	for w := range konst.WeeksShown {
		week := make([]twi.NodeOption, konst.DaysInWeek)
		for i := range week {
			week[i] = c.day(start.AddDate(0, 0, w*konst.DaysInWeek+i))
		}
		weeks = append(weeks, part("flex flex-row gap-1", week))
	}
	return part("flex flex-col w-fit gap-1 rounded-md bg-background p-1", append([]twi.NodeOption{
		part("relative flex flex-row items-center justify-between", []twi.NodeOption{turn("‹", -1), part("font-medium select-none", []twi.NodeOption{twi.Text(c.Month.Format("January 2006"))}), turn("›", 1)}),
		part(fade+"flex flex-col gap-1", append(c.behave(c.key), weeks...)),
	}, options...))
}

func (c *Calendar) day(d time.Time) twi.Node {
	classes := "w-4 rounded-md text-center select-none " + c.ring("", onItem)
	switch {
	case d.Equal(date(c.Selected)):
		classes += " bg-primary text-primary-foreground"
	case d.Equal(date(c.Today)):
		classes += " bg-accent text-accent-foreground"
	case !monthOf(d).Equal(c.Month):
		classes += " text-muted-foreground hover:bg-accent hover:text-accent-foreground"
	default:
		classes += " hover:bg-accent hover:text-accent-foreground"
	}
	return part(classes, []twi.NodeOption{c.dataActive(d.Equal(c.focus)), c.click(func() { c.pick(d) }), twi.Text(strconv.Itoa(d.Day()))})
}

func (c *Calendar) key(k input.KeyEvent) bool {
	step := map[input.Key]int{input.KeyArrowLeft: -1, input.KeyArrowRight: 1, input.KeyArrowUp: -konst.DaysInWeek, input.KeyArrowDown: konst.DaysInWeek}
	months := map[input.Key]int{input.KeyPageUp: -1, input.KeyPageDown: 1}
	span := 1
	if k.Modifiers == input.ModShift {
		span = konst.MonthsInYear
	}
	switch {
	case step[k.Key] != 0 && k.Modifiers == 0:
		c.focus = c.focus.AddDate(0, 0, step[k.Key])
	case months[k.Key] != 0 && k.Modifiers&^input.ModShift == 0:
		c.focus = addMonths(c.focus, months[k.Key]*span)
	case k.Key == input.KeyHome:
		c.focus = c.focus.AddDate(0, 0, -int(c.focus.Weekday()))
	case k.Key == input.KeyEnd:
		c.focus = c.focus.AddDate(0, 0, konst.DaysInWeek-1-int(c.focus.Weekday()))
	case press(k):
		c.pick(c.focus)
	default:
		return false
	}
	c.Month = monthOf(c.focus)
	return true
}

func (c *Calendar) pick(d time.Time) {
	c.focus, c.Month = d, monthOf(d)
	if !d.Equal(date(c.Selected)) {
		c.Selected = d
		notify(c.OnSelect, d)
	}
}
