package ui

import (
	"strconv"
	"testing"

	"github.com/twind-dev/twind/twi"
)

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
