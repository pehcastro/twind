package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func tabsPage(rt *twi.Runtime, open string) func() twi.Node {
	tabs, fruit, zone := ui.NewTabs(rt), ui.NewSelect(rt), ui.NewSelect(rt)
	name, username, current, next := ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	name.Insert("Pedro Duarte")
	username.Insert("@peduarte")
	current.Placeholder, next.Placeholder = "Current password", "New password"
	fruit.Placeholder, fruit.Open = "Select a fruit", open == "select"
	zone.Value = "utc"
	saved := "nothing saved yet"
	save := func(what string) twi.NodeOption {
		return ui.OnPress(func() {
			saved = what
			rt.Invalidate()
		})
	}
	field := func(label string, in *ui.Input) twi.Node {
		return ui.Field(ui.Vertical, ui.FieldLabel(twi.Text(label)), in.Node())
	}
	card := func(title, description, button string, fields ...twi.NodeOption) twi.Node {
		return ui.Card(twi.Class("py-1"),
			ui.CardHeader(ui.CardTitle(twi.Text(title)), ui.CardDescription(twi.Text(description))),
			ui.CardContent(el("flex flex-col gap-1", fields...)),
			ui.CardFooter(ui.Button(ui.Default, ui.SizeDefault, save(button), twi.Text(button))),
		)
	}
	return func() twi.Node {
		return el("flex flex-row grow gap-6",
			el("flex flex-col gap-1 w-48 shrink-0",
				section("Tabs", tabs.Node(
					tabs.List(tabs.Trigger("account", twi.Text("Account")), tabs.Trigger("password", twi.Text("Password"))),
					tabs.Content("account", card("Account", "Make changes to your account here. Click save when you're done.", "Save changes", field("Name", name), field("Username", username))),
					tabs.Content("password", card("Password", "Change your password here. After saving, you'll be logged out.", "Save password", field("Current password", current), field("New password", next))),
				)),
				text("text-muted-foreground", "Saved: "+saved),
			),
			el("flex flex-col gap-2 flex-1",
				section("Select", fruit.Node(fruit.Trigger(twi.Class("w-22")), fruit.Content(
					ui.SelectLabel(twi.Text("Fruits")),
					fruit.Item("apple", "Apple"), fruit.Item("banana", "Banana"), fruit.Item("blueberry", "Blueberry"),
					fruit.Item("grapes", "Grapes"), fruit.Item("pineapple", "Pineapple"),
				))),
				text("text-muted-foreground", "Fruit: "+fruit.Value),
				section("Time zone", zone.Node(zone.Trigger(twi.Class("w-30")), zone.Content(
					ui.SelectLabel(twi.Text("Europe & Africa")),
					zone.Item("gmt", "Greenwich Mean Time"), zone.Item("cet", "Central European Time"),
					ui.SelectSeparator(),
					ui.SelectLabel(twi.Text("Other")),
					zone.Item("utc", "Coordinated Universal Time"), zone.Item("ist", "India Standard Time"),
				))),
				ui.Alert(ui.Default, ui.AlertTitle(twi.Text("Overlays sit above the page")), ui.AlertDescription(twi.Text("An open select draws over the text and cards under it."))),
			),
		)
	}
}
