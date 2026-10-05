package ui

import (
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
)

func BenchmarkPointerFieldKeyToFrame(b *testing.B) {
	sheet, err := styles()
	if err != nil {
		b.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		name, bio := NewInput(rt), NewTextarea(rt)
		name.Insert("Peedro Alvares Cabral de Gouveia")
		bio.Insert("Born in Belmonte.\nSailed west in 1500.\nLanded in Porto Seguro.")
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 w-40"), name.Node(twi.AutoFocus()), bio.Node())
		}
	}, drive.Size(60, 14), drive.With(twi.Styles(sheet)))
	b.Cleanup(func() {
		if err := d.Close(); err != nil {
			b.Error(err)
		}
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		d.Press("a")
		d.Press("backspace")
	}
}

func BenchmarkTextareaKeyToFrame(b *testing.B) {
	sheet, err := styles()
	if err != nil {
		b.Fatal(err)
	}
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		bio := NewTextarea(rt)
		bio.Insert("Born in Belmonte.\nSailed west in 1500.\nLanded in Porto Seguro.\nThe fleet went on to Calicut.")
		return func() twi.Node {
			return twi.Element(twi.Class("flex flex-col gap-1 p-1 w-40"), bio.Node(twi.AutoFocus()))
		}
	}, drive.Size(60, 14), drive.With(twi.Styles(sheet)))
	b.Cleanup(func() {
		if err := d.Close(); err != nil {
			b.Error(err)
		}
	})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		d.Press("a")
		d.Press("backspace")
	}
}
