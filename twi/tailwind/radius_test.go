package tailwind

import (
	"slices"
	"testing"

	"github.com/pehcastro/twind/twi/style"
)

func TestRadiusCorners(t *testing.T) {
	type corner struct {
		p style.Property
		r style.Radius
	}
	tl, tr, br, bl := style.PropRadiusTopLeft, style.PropRadiusTopRight, style.PropRadiusBottomRight, style.PropRadiusBottomLeft
	rules, warned := appRules(t), appWarnings(t, "")
	for class, want := range map[string]struct {
		decls  []corner
		warned bool
	}{
		"rounded-md":         {decls: []corner{{style.PropRadius, style.RadiusMd}}},
		"rounded-t-xl":       {decls: []corner{{tl, style.RadiusLg}, {tr, style.RadiusLg}}, warned: true},
		"rounded-b-none":     {decls: []corner{{br, style.RadiusNone}, {bl, style.RadiusNone}}},
		"rounded-tl-lg":      {decls: []corner{{tl, style.RadiusLg}}},
		"rounded-tr-none":    {decls: []corner{{tr, style.RadiusNone}}},
		"rounded-br-full":    {decls: []corner{{br, style.RadiusFull}}},
		"rounded-bl-sm":      {decls: []corner{{bl, style.RadiusSm}}},
		"rounded-s-md":       {decls: []corner{{tl, style.RadiusMd}, {bl, style.RadiusMd}}},
		"rounded-e-lg":       {decls: []corner{{tr, style.RadiusLg}, {br, style.RadiusLg}}},
		"rounded-ss-sm":      {decls: []corner{{tl, style.RadiusSm}}},
		"rounded-se-md":      {decls: []corner{{tr, style.RadiusMd}}},
		"rounded-es-lg":      {decls: []corner{{bl, style.RadiusLg}}},
		"rounded-ee-full":    {decls: []corner{{br, style.RadiusFull}}},
		"rounded-l-[3px]":    {decls: []corner{{tl, style.RadiusSm}, {bl, style.RadiusSm}}, warned: true},
		"rounded-l-full":     {decls: []corner{{tl, style.RadiusFull}, {bl, style.RadiusFull}}},
		"first:rounded-l-md": {decls: []corner{{tl, style.RadiusMd}, {bl, style.RadiusMd}}},
		"last:rounded-r-md":  {decls: []corner{{tr, style.RadiusMd}, {br, style.RadiusMd}}},
	} {
		if len(rules[class]) != 1 {
			t.Errorf("%s: %d rules, want one", class, len(rules[class]))
			continue
		}
		var got []corner
		for _, d := range rules[class][0].Decls {
			got = append(got, corner{d.Property, d.Radius})
		}
		if !slices.Equal(got, want.decls) {
			t.Errorf("%s: %v, want %v", class, got, want.decls)
		}
		if c, ok := warned[class]; ok != want.warned || ok && c != Approximated {
			t.Errorf("%s: warned %v (%s), want %v approximated", class, ok, c, want.warned)
		}
	}
}
