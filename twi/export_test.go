package twi

import (
	"image"
	"io"

	"github.com/pehcastro/twind/internal/terminal"
)

func RenderInline(out io.Writer, node Node, caps terminal.Capabilities, cursor image.Point, screenRows int, opts ...RenderOption) (bool, error) {
	var cfg renderConfig
	for _, o := range opts {
		o(&cfg)
	}
	return inline(out, node, cfg, caps, cursor, screenRows)
}
