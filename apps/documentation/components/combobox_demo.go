package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func ComboboxDemo(rt *twi.Runtime) func() twi.Node {
	framework := ui.NewCombobox(rt)
	framework.Placeholder, framework.Empty, framework.ShowClear = "Select framework...", "No framework found.", true
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col h-9 gap-1"),
			framework.Node(twi.Class("w-40"),
				framework.Input(),
				framework.Content(
					framework.Item("next", "Next.js"),
					framework.Item("svelte", "SvelteKit"),
					framework.Item("nuxt", "Nuxt.js"),
					framework.Item("remix", "Remix"),
					framework.Item("astro", "Astro"),
				),
			),
		)
	}
}
