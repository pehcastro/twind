package ui

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/text"
)

func Direction(dir text.Direction, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Dir(dir)}, children...)...)
}
