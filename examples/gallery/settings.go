package main

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/ui"
)

func newSettings(rt *twi.Runtime, start theme.Theme, toaster *ui.Toaster, dark *ui.Switch) func() twi.Node {
	palette := ui.NewRadioGroup(rt)
	palette.Value, dark.Checked = start.Name, start.Scheme == theme.Dark
	preview := func(name string) {
		for _, t := range theme.Builtin() {
			if t.Name == name && (t.Scheme == theme.Dark) == dark.Checked {
				rt.SetTheme(t)
			}
		}
	}
	apply := func() { preview(palette.Value) }
	palette.OnChange = func(string) { apply() }
	dark.OnChange = func(bool) { apply() }
	apply()
	opened := palette.Value
	restore := onKeys(rt, func(k input.KeyEvent) bool {
		if k.Key != input.KeyEscape || palette.Value == opened {
			return false
		}
		palette.Value = opened
		apply()
		return true
	})
	var names []string
	for _, t := range theme.Builtin() {
		if t.Scheme == theme.Light {
			names = append(names, t.Name)
		}
	}
	swatch := func(class, label string) twi.Node {
		return el("flex flex-col items-center gap-0", el("w-6 h-2 rounded-md shadow-[0_0_0_1px_var(--color-border)] "+class), txt("text-muted-foreground", label))
	}
	sample := func(v ui.Variant, label string) twi.Node {
		return ui.Button(v, ui.SizeDefault, clicked(rt, func() { toaster.Show(label+" pressed", "A sample of the "+label+" button.", ui.ToastAction{}) }), twi.Text(label))
	}
	return func() twi.Node {
		items := []twi.NodeOption{twi.Class("grid grid-cols-2 gap-x-1"), twi.OnFocus(func() { opened = palette.Value }), twi.OnPointerLeave(apply), restore}
		for _, n := range names {
			items = append(items, palette.Item(n, twi.OnPointerEnter(func() { preview(n) }), ui.Label(twi.Text(n))))
		}
		scheme := map[bool]string{false: "Light", true: "Dark"}[dark.Checked]
		return ui.Card(twi.Class("py-1"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Appearance")),
				ui.CardDescription(twi.Text("Every theme the runtime ships, light and dark. A hover previews, Escape goes back.")),
			),
			ui.CardContent(el("flex flex-row gap-4",
				ui.FieldSet(twi.Class("w-26 shrink-0"), ui.FieldLegend(twi.Text("Palette")), palette.Node(items...)),
				el("flex flex-col grow gap-2",
					ui.Field(ui.Horizontal, dark.Node(), ui.FieldLabel(twi.Text("Dark mode")), txt("text-muted-foreground", scheme)),
					ui.Separator(ui.Horizontal),
					txt("font-medium", "Preview: "+palette.Value+" "+scheme),
					el("flex flex-row gap-2",
						swatch("bg-primary", "primary"), swatch("bg-secondary", "second"), swatch("bg-accent", "accent"),
						swatch("bg-muted", "muted"), swatch("bg-destructive", "danger"), swatch("bg-card", "card"),
					),
					el("flex flex-row flex-wrap items-center gap-2",
						sample(ui.Default, "Primary"), sample(ui.Secondary, "Secondary"), sample(ui.Outline, "Outline"),
						ui.Badge(ui.Default, twi.Text("Badge")),
						ui.Badge(ui.Destructive, twi.Text("Error")),
					),
					ui.Alert(ui.Default, ui.AlertTitle(twi.Text("Themes switch at runtime")), ui.AlertDescription(twi.Text("The Style IR is compiled once; a theme only swaps the token values."))),
				),
			)),
		)
	}
}
