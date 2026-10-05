package twi_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/text"
)

func TestDir(t *testing.T) {
	page := twi.Element(twi.Dir(text.DirRTL),
		twi.Element(twi.Text("שלום world")),
		twi.Element(twi.Dir(text.DirLTR), twi.Text("שלום world")),
	)
	got := strings.Fields(strings.ReplaceAll(rendered(t, page, twi.Width(16), twi.ColorProfile(color.None)), " ", "_"))
	want := []string{"______world_םולש", "םולש_world"}
	if !slices.Equal(got, want) {
		t.Errorf("rtl page with an ltr child printed %q, want %q", got, want)
	}
}

func TestDirAutoPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("twi.Dir(text.DirAuto) did not panic")
		}
	}()
	twi.Dir(text.DirAuto)
}
