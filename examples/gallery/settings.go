package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

func newSettings(rt *twi.Runtime, start theme.Theme) func() twi.Node {
	palette, dark := ui.NewRadioGroup(rt), ui.NewSwitch(rt)
	palette.Value, dark.Checked = start.Name, start.Scheme == theme.Dark
	apply := func() {
		for _, t := range theme.Builtin() {
			if t.Name == palette.Value && (t.Scheme == theme.Dark) == dark.Checked {
				rt.SetTheme(t)
			}
		}
	}
	palette.OnChange = func(string) { apply() }
	dark.OnChange = func(bool) { apply() }
	apply()
	var names []string
	for _, t := range theme.Builtin() {
		if t.Scheme == theme.Light {
			names = append(names, t.Name)
		}
	}
	swatch := func(class, label string) twi.Node {
		return el("flex flex-col items-center gap-0", el("w-6 h-2 rounded-md shadow-[0_0_0_1px_var(--color-border)] "+class), txt("text-muted-foreground", label))
	}
	return func() twi.Node {
		var items []twi.NodeOption
		for _, n := range names {
			items = append(items, palette.Item(n, ui.Label(twi.Text(n))))
		}
		scheme := map[bool]string{false: "Light", true: "Dark"}[dark.Checked]
		return ui.Card(twi.Class("py-1"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Appearance")),
				ui.CardDescription(twi.Text("Every shadcn palette the runtime ships, light and dark. Applied as you choose.")),
			),
			ui.CardContent(el("flex flex-row gap-4",
				ui.FieldSet(twi.Class("w-20 shrink-0"), ui.FieldLegend(twi.Text("Palette")), palette.Node(items...)),
				el("flex flex-col grow gap-2",
					ui.Field(ui.Horizontal, dark.Node(), ui.FieldLabel(twi.Text("Dark mode")), txt("text-muted-foreground", scheme)),
					ui.Separator(ui.Horizontal),
					txt("font-medium", "Preview: "+palette.Value+" "+scheme),
					el("flex flex-row gap-2",
						swatch("bg-primary", "primary"), swatch("bg-secondary", "second"), swatch("bg-accent", "accent"),
						swatch("bg-muted", "muted"), swatch("bg-destructive", "danger"), swatch("bg-card", "card"),
					),
					el("flex flex-row items-center gap-2",
						ui.Button(ui.Default, ui.SizeDefault, twi.Text("Primary")),
						ui.Button(ui.Secondary, ui.SizeDefault, twi.Text("Secondary")),
						ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Outline")),
						ui.Badge(ui.Default, twi.Text("Badge")),
						ui.Badge(ui.Destructive, twi.Text("Error")),
					),
					ui.Alert(ui.Default, ui.AlertTitle(twi.Text("Themes switch at runtime")), ui.AlertDescription(twi.Text("The Style IR is compiled once; a theme only swaps the token values."))),
				),
			)),
		)
	}
}
