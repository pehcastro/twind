package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/style"
)

type Orientation uint8

const (
	Horizontal Orientation = iota
	Vertical
)

type Side uint8

const (
	SideBottom Side = iota
	SideTop
	SideRight
	SideLeft
)

type Align uint8

const (
	AlignCenter Align = iota
	AlignStart
	AlignEnd
)

type active struct {
	twi.NodeOption
	on bool
}

func Active(on bool) twi.NodeOption {
	return active{twi.Data("active", strconv.FormatBool(on)), on}
}

func pick[K ~uint8](component string, key K, classes map[K]string) string {
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
