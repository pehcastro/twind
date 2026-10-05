package app

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func openable() []string {
	return []string{"dialog", "alert", "sheet", "menu", "menu-sub", "popover", "tooltip"}
}

func newOverlays(rt *twi.Runtime, open string) (view func() twi.Node, modal func() bool) {
	dialog, alert, sheet := ui.NewDialog(rt), ui.NewAlertDialog(rt), ui.NewSheet(rt, ui.SideRight)
	menu, pop, tip := ui.NewDropdownMenu(rt), ui.NewPopover(rt), ui.NewTooltip(rt)
	invite := menu.Sub()
	menu.Align, pop.Align = ui.AlignStart, ui.AlignStart
	chosen, statusBar, panel := "nothing yet", true, "Bottom"
	menu.OnSelect = func(item string) { chosen = item }
	name, username, width, height := ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	for in, v := range map[*ui.Input]string{name: "Pedro Duarte", username: "@peduarte", width: "100%", height: "25px"} {
		in.Insert(v)
	}
	switch open {
	case "dialog":
		dialog.Open = true
	case "alert":
		alert.Open = true
	case "sheet":
		sheet.Open = true
	case "menu":
		menu.Open = true
	case "menu-sub":
		menu.Open, invite.Open = true, true
	case "popover":
		pop.Open = true
	case "tooltip":
		tip.Open = true
	}
	outline, solid, size := ui.ButtonOutline, ui.ButtonDefault, ui.ButtonSizeDefault
	demo := func(title, about string, trigger twi.Node) twi.Node {
		return ui.Card(twi.Class("min-w-0 py-1"),
			ui.CardHeader(ui.CardTitle(twi.Text(title)), ui.CardDescription(twi.Text(about))),
			ui.CardContent(el("flex flex-row", trigger)),
		)
	}
	field := func(label string, in *ui.Input) twi.Node {
		return ui.Field(ui.Vertical, ui.FieldLabel(twi.Text(label)), in.Node())
	}
	view = func() twi.Node {
		return el("flex flex-col gap-2",
			el("grid grid-cols-3 gap-2",
				demo("Dialog", "A window over the page, focus trapped.", dialog.Trigger(outline, size, twi.Text("Edit Profile"))),
				demo("Dropdown menu", "Selected: "+chosen, menu.Node(menu.Trigger(outline, size, twi.Text("Open menu  ⌄")), menu.Content(
					ui.DropdownMenuLabel(twi.Text("My Account")),
					menu.Item("Profile", ui.DropdownMenuShortcut(twi.Text("⇧⌘P"))),
					menu.Item("Billing", ui.DropdownMenuShortcut(twi.Text("⌘B"))),
					invite.Node(invite.Trigger("Invite users"), invite.Content(invite.Item("Email"), invite.Item("Message"), ui.DropdownMenuSeparator(), invite.Item("More..."))),
					menu.Item("Settings", ui.DropdownMenuShortcut(twi.Text("⌘S"))),
					ui.DropdownMenuSeparator(),
					menu.CheckboxItem("Status Bar", &statusBar),
					ui.DropdownMenuSeparator(),
					ui.DropdownMenuLabel(twi.Text("Panel Position")),
					menu.RadioItem("Top", &panel), menu.RadioItem("Bottom", &panel),
					ui.DropdownMenuSeparator(),
					menu.Item("Log out", ui.DropdownMenuShortcut(twi.Text("⇧⌘Q"))),
				))),
				demo("Popover", "Rich content anchored to a trigger.", pop.Node(pop.Trigger(outline, size, twi.Text("Dimensions")), pop.Content(
					el("flex flex-col gap-1",
						el("flex flex-col", txt("font-medium", "Dimensions"), txt("text-muted-foreground", "Set the layer size.")),
						el("flex flex-row items-center gap-2", el("w-7", ui.Label(twi.Text("Width"))), width.Node()),
						el("flex flex-row items-center gap-2", el("w-7", ui.Label(twi.Text("Height"))), height.Node()),
					),
				))),
				demo("Sheet", "A panel that slides in from an edge.", sheet.Trigger(outline, size, twi.Text("Open sheet"))),
				demo("Tooltip", "A hint on focus, gone on blur.", tip.Node(tip.Trigger(outline, size, twi.Text("Focus me")), tip.Content(twi.Text("Add to library")))),
				demo("Alert dialog", "Asks before anything destructive.", alert.Trigger(ui.ButtonDestructive, size, twi.Text("Delete account"))),
			),
			ui.Alert(ui.AlertDefault, ui.AlertTitle(twi.Text("Overlays sit above the page")), ui.AlertDescription(twi.Text("Dialogs dim the page; menus and popovers float over it with soft shadows. Escape closes each one."))),
			dialog.Content(
				dialog.Header(dialog.Title(twi.Text("Edit profile")), dialog.Description(twi.Text("Make changes to your profile here. Click save when you're done."))),
				el("flex flex-col gap-1", field("Name", name), field("Username", username)),
				dialog.Footer(dialog.Close(outline, size, twi.Text("Cancel")), dialog.Close(solid, size, twi.Text("Save changes"))),
			),
			sheet.Content(
				sheet.Header(sheet.Title(twi.Text("Notifications")), sheet.Description(twi.Text("Choose what reaches you."))),
				el("flex flex-col gap-1 px-2", txt("font-medium", "Mentions"), txt("text-muted-foreground", "When someone @mentions you."), txt("font-medium", "Reviews"), txt("text-muted-foreground", "When a review is requested.")),
				sheet.Footer(sheet.Close(solid, size, twi.Text("Save")), sheet.Close(outline, size, twi.Text("Close"))),
			),
			alert.Content(
				alert.Header(alert.Title(twi.Text("Are you absolutely sure?")), alert.Description(twi.Text("This action cannot be undone. This will permanently delete your account."))),
				alert.Footer(alert.Close(outline, size, twi.Text("Cancel")), alert.Close(ui.ButtonDestructive, size, twi.Text("Delete"))),
			),
		)
	}
	return view, func() bool { return dialog.Open || alert.Open || sheet.Open }
}
