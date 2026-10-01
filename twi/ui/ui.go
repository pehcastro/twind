package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/style"
)

type Variant uint8

const (
	Default Variant = iota
	Destructive
	Outline
	Secondary
	Ghost
	Link
	Icon
	Muted
	Image
	Tinted
	Ruled
	Bordered
)

type Size uint8

const (
	SizeDefault Size = iota
	SizeXS
	SizeSM
	SizeLG
	SizeIcon
)

type Orientation uint8

const (
	Horizontal Orientation = iota
	Vertical
)

func pick[K Variant | Size | Orientation | Align | Side | Alignment | dialogKind | ringAt | Upload](component string, key K, classes map[K]string) string {
	c, ok := classes[key]
	if !ok {
		panic(fmt.Sprintf("ui: %s has no %T %d", component, key, key))
	}
	return c
}

func part(classes string, children []twi.NodeOption) twi.Node {
	return twi.Element(merged(classes, children)...)
}

func merged(classes string, options []twi.NodeOption) []twi.NodeOption {
	if caller, rest := twi.Classes(options); len(caller) > 0 {
		classes, options = Merge(classes, strings.Join(caller, " ")), rest
	}
	return append([]twi.NodeOption{twi.Class(classes)}, options...)
}

type ItemIcon struct{ twi.Node }

func slotted(out, children []twi.NodeOption, label ...twi.NodeOption) []twi.NodeOption {
	out = slices.Grow(out, len(children)+len(label))
	for _, c := range children {
		if _, ok := c.(ItemIcon); ok {
			out = append(out, c)
		}
	}
	out = append(out, label...)
	for _, c := range children {
		if _, ok := c.(ItemIcon); !ok {
			out = append(out, c)
		}
	}
	return out
}

func icon(glyph, classes string) twi.Node {
	return part(classes, []twi.NodeOption{twi.Tag(style.ElementSVG), twi.Text(glyph)})
}
