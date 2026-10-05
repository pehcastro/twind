package twi_test

import (
	"testing"

	"github.com/pehcastro/twind/twi"
)

func TestOptionsBelongToTheirEntry(t *testing.T) {
	if _, ok := any(twi.Width(80)).(twi.Option); ok {
		t.Error("New accepts Width, which only Render reads")
	}
	if _, ok := any(twi.Backend(nil, nil)).(twi.RenderOption); ok {
		t.Error("Render accepts Backend, which only New reads")
	}
}
