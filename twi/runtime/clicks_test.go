package runtime_test

import (
	"image"
	"slices"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/runtime/testdata/hover"
	"github.com/pehcastro/twind/twi/runtime/testdata/selection"
)

func TestPointerDownSeesTheClickCount(t *testing.T) {
	sheet, err := hover.Styles()
	if err != nil {
		t.Fatal(err)
	}
	var counts []int
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		down := twi.OnPointerDown(func(*twi.Event) { counts = append(counts, rt.Clicks()) })
		return func() twi.Node { return twi.Element(down, twi.Text("one two three")) }
	}, drive.Size(20, 2), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	for range 4 {
		d.Click(1, 0)
	}
	d.Advance(time.Second)
	d.Click(1, 0)
	d.Click(9, 0)
	if want := []int{1, 2, 3, 3, 1, 1}; !slices.Equal(counts, want) {
		t.Errorf("click counts %v, want %v", counts, want)
	}
}

func TestPointerContentBoxOfTheListener(t *testing.T) {
	sheet, err := selection.Styles()
	if err != nil {
		t.Fatal(err)
	}
	var boxes []image.Rectangle
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		down := twi.OnPointerDown(func(ev *twi.Event) { boxes = append(boxes, rt.ContentBox(ev.Current())) })
		return func() twi.Node {
			return twi.Element(twi.Class(selection.Card), down, twi.Element(twi.Focusable(), twi.Text("x")))
		}
	}, drive.Size(30, 6), drive.Styles(sheet))
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Error(err)
		}
	})
	d.Click(2, 1)
	d.Click(0, 0)
	want := image.Rect(2, 1, 22, 2)
	if !slices.Equal(boxes, []image.Rectangle{want, want}) {
		t.Errorf("content boxes %v, want %v twice: the text, then the border", boxes, want)
	}
}
