package components

import (
	"embed"

	"github.com/twind-dev/twind/twi"
)

//go:embed *.go
var Source embed.FS

type Demo struct {
	File string
	New  func(rt *twi.Runtime) func() twi.Node
}

func All() map[string]Demo {
	return map[string]Demo{
		"button-demo":     {"button_demo.go", ButtonDemo},
		"button-variants": {"button_variants.go", ButtonVariants},
		"button-sizes":    {"button_sizes.go", ButtonSizes},
		"dialog-demo":     {"dialog_demo.go", DialogDemo},
		"tabs-demo":       {"tabs_demo.go", TabsDemo},
	}
}
