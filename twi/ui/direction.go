package ui

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/text"
)

func Direction(dir text.Direction, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Dir(dir)}, children...)...)
}
