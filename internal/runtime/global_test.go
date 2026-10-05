package runtime_test

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/pehcastro/twind/internal/events"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func TestGlobalKeyEventMethodsAreDefined(t *testing.T) {
	var got []string
	read := func(hook string) func(*twi.Event) {
		return func(e *twi.Event) {
			e.StopPropagation()
			got = append(got, fmt.Sprint(hook, e.Offset(), e.Target() == nil, e.Current() == nil, e.Phase() == events.Phase(0), e.DefaultPrevented()))
		}
	}
	d := drive.New(func(*twi.Runtime) func() twi.Node {
		return func() twi.Node {
			return twi.Element(twi.OnHotkey(read("hotkey")), twi.OnKey(read("key")), twi.Text("x"))
		}
	}, drive.Size(10, 2))
	d.Press("a")
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	want := []string{fmt.Sprint("hotkey", image.Point{}, true, true, true, false), fmt.Sprint("key", image.Point{}, true, true, true, false)}
	if !slices.Equal(got, want) {
		t.Errorf("the global hooks read %q, want %q", got, want)
	}
}
