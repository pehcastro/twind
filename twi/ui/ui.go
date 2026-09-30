package ui

import (
	"fmt"

	"github.com/twind-dev/twind/twi"
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

func pick[K Variant | Size | Orientation | Align | Side | Alignment | dialogKind](component string, key K, classes map[K]string) string {
	c, ok := classes[key]
	if !ok {
		panic(fmt.Sprintf("ui: %s has no %T %d", component, key, key))
	}
	return c
}

func part(classes string, children []twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(classes)}, children...)...)
}
