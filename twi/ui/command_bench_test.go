package ui

import (
	"slices"
	"strconv"
	"testing"

	"github.com/pehcastro/twind/twi"
)

func TestCommandAllocsDoNotGrowWithItems(t *testing.T) {
	counts := map[int]float64{}
	for _, n := range []int{1000, 10000} {
		c := NewCommand(twi.New())
		items := make([]CommandItem, n)
		for i := range items {
			items[i] = c.Item("Command number " + strconv.Itoa(i))
		}
		groups := []CommandGroup{c.Group("Commands", items[:n/2]...), c.Group("More", items[n/2:]...)}
		c.Search("co")
		c.List(groups...)
		counts[n] = testing.AllocsPerRun(20, func() { c.List(groups...) })
	}
	if counts[1000] != counts[10000] {
		t.Errorf("List allocates %v times over 1,000 items and %v over 10,000", counts[1000], counts[10000])
	}
	c := NewCommand(twi.New())
	long := []CommandItem{c.Item("Alpha"), c.Item("Beta"), c.Item("Gamma"), c.Item("Delta")}
	c.List(c.Group("Long", long...))
	c.List(c.Group("Short", c.Item("Omega")))
	if !slices.Equal(c.visible, []string{"Omega"}) {
		t.Errorf("after a shorter list the visible items are %v, want [Omega]", c.visible)
	}
}

func BenchmarkCommandFilter(b *testing.B) {
	c := NewCommand(twi.New())
	var items []CommandItem
	for i := range 1000 {
		items = append(items, c.Item("Command number "+strconv.Itoa(i)))
	}
	group := c.Group("Commands", items...)
	searches := []string{"c", "co", "com", "comm", "comma", "command", "command ", "command n", "command 5", "command 55", "cmd5", "zz"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		c.Search(searches[i%len(searches)])
		c.List(group)
	}
}
