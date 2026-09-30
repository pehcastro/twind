package blocks

import (
	"embed"

	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/twi"
)

//go:embed *.go
var source embed.FS

func All() components.Catalog {
	return components.Catalog{Source: source, Demos: map[string]func(rt *twi.Runtime) func() twi.Node{
		"login-form":     LoginForm,
		"dashboard-card": DashboardCard,
	}}
}
