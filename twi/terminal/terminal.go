package terminal

import "github.com/pehcastro/twind/internal/terminal"

type Graphics = terminal.Graphics

const (
	GraphicsNone   = terminal.GraphicsNone
	GraphicsSixel  = terminal.GraphicsSixel
	GraphicsITerm2 = terminal.GraphicsITerm2
	GraphicsKitty  = terminal.GraphicsKitty
	GraphicsGDI    = terminal.GraphicsGDI
)

type Capabilities = terminal.Capabilities
