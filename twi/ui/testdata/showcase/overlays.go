package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func overlays(rt *twi.Runtime, open string) func() twi.Node {
	dialog, alert, sheet, drawer := ui.NewDialog(rt), ui.NewAlertDialog(rt), ui.NewSheet(rt, ui.Right), ui.NewDrawer(rt, ui.Bottom)
	menu, pop, tip, card := ui.NewDropdownMenu(rt), ui.NewPopover(rt), ui.NewTooltip(rt), ui.NewHoverCard(rt)
	invite := menu.Sub()
	menu.Align = ui.Start
	chosen, status, panel := "nothing yet", true, "Bottom"
	menu.OnSelect = func(item string) { chosen = item }
	name, username, sheetName, sheetUser, width, height := ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	for in, v := range map[*ui.Input]string{name: "Pedro Duarte", username: "@peduarte", sheetName: "Pedro Duarte", sheetUser: "@peduarte", width: "100%", height: "25px"} {
		in.Insert(v)
	}
	switch open {
	case "dialog":
		dialog.Open = true
	case "alert":
		alert.Open = true
	case "sheet":
		sheet.Open = true
	case "drawer":
		drawer.Open = true
	case "menu":
		menu.Open = true
	case "menu-sub":
		menu.Open, invite.Open = true, true
	case "popover":
		pop.Open = true
	case "tooltip":
		tip.Open = true
	case "hovercard":
		card.Open = true
	}
	outline, solid, size := ui.Outline, ui.Default, ui.SizeDefault
	field := func(name string, in *ui.Input) twi.Node {
		return ui.Field(ui.Vertical, ui.FieldLabel(label(name)), in.Node())
	}
	return func() twi.Node {
		return el("flex flex-row grow gap-8",
			el("flex flex-col gap-2 w-40",
				section("Dialog", row(dialog.Trigger(outline, size, label("Edit Profile")), dialog.Content(
					dialog.Header(dialog.Title(label("Edit profile")), dialog.Description(label("Make changes to your profile here. Click save when you're done."))),
					el("flex flex-col gap-1", field("Name", name), field("Username", username)),
					dialog.Footer(dialog.Close(outline, size, label("Cancel")), dialog.Close(solid, size, label("Save changes"))),
				))),
				section("Alert dialog", row(alert.Trigger(outline, size, label("Show Dialog")), alert.Content(
					alert.Header(alert.Title(label("Are you absolutely sure?")), alert.Description(label("This action cannot be undone. This will permanently delete your account and remove your data from our servers."))),
					alert.Footer(alert.Close(outline, size, label("Cancel")), alert.Close(solid, size, label("Continue"))),
				))),
				section("Sheet", row(sheet.Trigger(outline, size, label("Open")), sheet.Content(
					sheet.Header(sheet.Title(label("Edit profile")), sheet.Description(label("Make changes to your profile here. Click save when you're done."))),
					el("flex flex-col gap-1 px-2", field("Name", sheetName), field("Username", sheetUser)),
					sheet.Footer(sheet.Close(solid, size, label("Save changes")), sheet.Close(outline, size, label("Close"))),
				))),
				section("Drawer", row(drawer.Trigger(outline, size, label("Open Drawer")), drawer.Content(
					el("flex flex-col w-full max-w-48 self-center",
						drawer.Header(drawer.Title(label("Move Goal")), drawer.Description(label("Set your daily activity goal."))),
						el("flex flex-col items-center py-1", text("font-bold", "350"), text("text-muted-foreground", "CALORIES/DAY")),
						drawer.Footer(drawer.Close(solid, size, label("Submit")), drawer.Close(outline, size, label("Cancel"))),
					),
				))),
			),
			el("flex flex-col gap-2 w-40",
				section("Dropdown menu",
					menu.Node(menu.Trigger(outline, size, label("Open")), menu.Content(
						ui.DropdownMenuLabel(label("My Account")),
						menu.Item("Profile", ui.DropdownMenuShortcut(label("⇧⌘P"))),
						menu.Item("Billing", ui.DropdownMenuShortcut(label("⌘B"))),
						invite.Node(invite.Trigger("Invite users"), invite.Content(invite.Item("Email"), invite.Item("Message"), ui.DropdownMenuSeparator(), invite.Item("More..."))),
						menu.Item("Settings", ui.DropdownMenuShortcut(label("⌘S"))),
						ui.DropdownMenuSeparator(),
						menu.CheckboxItem("Status Bar", &status),
						ui.DropdownMenuSeparator(),
						ui.DropdownMenuLabel(label("Panel Position")),
						menu.RadioItem("Top", &panel), menu.RadioItem("Bottom", &panel),
						ui.DropdownMenuSeparator(),
						menu.Item("Log out", ui.DropdownMenuShortcut(label("⇧⌘Q"))),
					)),
					text("text-muted-foreground", "Selected: "+chosen),
				),
				section("Popover", pop.Node(pop.Trigger(outline, size, label("Open popover")), pop.Content(
					el("flex flex-col gap-1",
						el("flex flex-col", text("font-medium", "Dimensions"), text("text-muted-foreground", "Set the dimensions for the layer.")),
						el("flex flex-row items-center gap-2", el("w-8", ui.Label(label("Width"))), width.Node()),
						el("flex flex-row items-center gap-2", el("w-8", ui.Label(label("Height"))), height.Node()),
					),
				))),
				section("Tooltip", tip.Node(tip.Trigger(outline, size, label("Hover")), tip.Content(label("Add to library")))),
				section("Hover card", card.Node(card.Trigger(ui.Link, size, label("@nextjs")), card.Content(
					text("font-semibold", "@nextjs"),
					label("The React Framework, created and maintained by @vercel."),
					text("text-muted-foreground", "Joined December 2021"),
				))),
			),
			el("flex flex-col gap-2 grow",
				ui.Card(
					ui.CardHeader(ui.CardTitle(label("Team Members")), ui.CardDescription(label("Invite your team members to collaborate."))),
					ui.CardContent(el("flex flex-col gap-1",
						el("flex flex-row items-center gap-2", ui.Avatar(ui.SizeSM, ui.AvatarFallback(label("OM"))), el("flex flex-col grow", text("font-medium", "Sofia Davis"), text("text-muted-foreground", "m@example.com")), ui.Badge(ui.Secondary, label("Owner"))),
						el("flex flex-row items-center gap-2", ui.Avatar(ui.SizeSM, ui.AvatarFallback(label("JL"))), el("flex flex-col grow", text("font-medium", "Jackson Lee"), text("text-muted-foreground", "p@example.com")), ui.Badge(ui.Outline, label("Member"))),
					)),
				),
				ui.Alert(ui.Default, ui.AlertTitle(label("Overlays sit above the page")), ui.AlertDescription(label("Dialogs dim the page, menus and popovers float over it with soft shadows."))),
			),
		)
	}
}
