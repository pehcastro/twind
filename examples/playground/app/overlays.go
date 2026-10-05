package app

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func alertDialogPage(c controls) twi.Node {
	a := c.kit.alert
	return show("Alert dialog", "asks before anything destructive; only a button closes it",
		a.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Show dialog")),
		a.Content(
			a.Header(a.Title(twi.Text("Are you absolutely sure?")), a.Description(twi.Text("This cannot be undone. It deletes your account and its data."))),
			a.Footer(a.Close(ui.Outline, ui.SizeDefault, twi.Text("Cancel")), a.Close(ui.Default, ui.SizeDefault, twi.Text("Continue"))),
		),
	)
}

func commandDialogPage(c controls) twi.Node {
	return show("Command dialog", "the palette this playground navigates with",
		row(ui.KbdGroup(ui.Kbd(twi.Text("ctrl")), ui.Kbd(twi.Text("k"))), txt("text-muted-foreground", "anywhere, or"), c.kit.palette.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open the palette"))),
	)
}

func contextMenuPage(c controls) twi.Node {
	m := c.kit.context
	return show("Context menu", "a right click, or Shift+F10 on the focused area",
		m.Node(
			m.Trigger(twi.Class("h-5 w-50 items-center justify-center rounded-md border border-dashed"), twi.Text("Right click here")),
			m.Content(
				m.Item("Back", ui.DropdownMenuShortcut(twi.Text("⌘["))), m.Item("Forward", ui.DropdownMenuShortcut(twi.Text("⌘]"))), m.Item("Reload"),
				ui.DropdownMenuSeparator(),
				m.CheckboxItem("Show bookmarks", &c.kit.bookmarks),
				ui.DropdownMenuSeparator(),
				ui.DropdownMenuLabel(twi.Text("People")),
				m.RadioItem("Pedro Duarte", &c.kit.person), m.RadioItem("Colm Tuite", &c.kit.person),
			),
		),
		txt("text-muted-foreground", "chosen: "+c.kit.chosen),
	)
}

func dialogPage(c controls) twi.Node {
	return show("Dialog", "the motion page's dialog: zooms in over a fading backdrop, Escape or a click outside closes",
		c.kit.dialog.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open dialog")),
	)
}

func drawerPage(c controls) twi.Node {
	k := c.kit
	d := k.drawer
	step := func(label string, by int) twi.Node {
		return ui.Button(ui.Outline, ui.SizeIcon, c.clicked(func() { k.goal = min(max(k.goal+by, 200), 500) }), twi.Text(label))
	}
	return show("Drawer", "slides up from the bottom edge; drag its handle down to close it",
		d.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open drawer")),
		d.Content(el("flex flex-col w-full max-w-48 self-center",
			d.Header(d.Title(twi.Text("Move goal")), d.Description(twi.Text("Set your daily activity goal."))),
			el("flex flex-row items-center justify-center gap-4", step("-", -10), el("flex flex-col items-center", txt("font-bold", strconv.Itoa(k.goal)), txt("text-muted-foreground", "calories a day")), step("+", 10)),
			d.Footer(d.Close(ui.Default, ui.SizeDefault, twi.Class("py-1"), twi.Text("Submit")), d.Close(ui.Outline, ui.SizeDefault, twi.Class("border shadow-none"), twi.Text("Cancel"))),
		)),
	)
}

func dropdownMenuPage(c controls) twi.Node {
	k := c.kit
	m, invite := k.menu, k.invite
	return show("Dropdown menu", "items, a submenu, a checkbox and radio items; arrows, Enter, or the pointer",
		m.Node(m.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open menu  ⌄")), m.Content(
			ui.DropdownMenuLabel(twi.Text("My Account")),
			m.Item("Profile", ui.DropdownMenuShortcut(twi.Text("⇧⌘P"))),
			m.Item("Billing", ui.DropdownMenuShortcut(twi.Text("⌘B"))),
			invite.Node(invite.Trigger("Invite users"), invite.Content(invite.Item("Email"), invite.Item("Message"), ui.DropdownMenuSeparator(), invite.Item("More..."))),
			ui.DropdownMenuSeparator(),
			m.CheckboxItem("Status Bar", &k.statusBar),
			ui.DropdownMenuSeparator(),
			ui.DropdownMenuLabel(twi.Text("Panel Position")),
			m.RadioItem("Top", &k.panel), m.RadioItem("Bottom", &k.panel),
		)),
		txt("text-muted-foreground", "chosen: "+k.chosen),
	)
}

func hoverCardPage(c controls) twi.Node {
	h := c.kit.hover
	return show("Hover card", "shown while the pointer or the focus is on the link",
		h.Node(h.Trigger(ui.Link, ui.SizeDefault, twi.Text("@nextjs")), h.Content(
			txt("font-semibold", "@nextjs"),
			twi.Text("The React Framework, created and maintained by @vercel."),
			txt("text-muted-foreground", "Joined December 2021"),
		)),
	)
}

func menubarPage(c controls) twi.Node {
	k := c.kit
	menu := func(m *ui.MenubarMenu, name string, items ...twi.NodeOption) twi.Node {
		return m.Node(m.Trigger(twi.Text(name)), m.Content(items...))
	}
	return show("Menubar", "arrows move across and down, the pointer follows an open menu",
		k.bar.Node(
			menu(k.file, "File", k.file.Item("New Tab", ui.DropdownMenuShortcut(twi.Text("⌘T"))), k.file.Item("New Window", ui.DropdownMenuShortcut(twi.Text("⌘N"))), ui.DropdownMenuSeparator(), k.file.Item("Print...")),
			menu(k.edit, "Edit", k.edit.Item("Undo", ui.DropdownMenuShortcut(twi.Text("⌘Z"))), k.edit.Item("Redo"), ui.DropdownMenuSeparator(), k.edit.Item("Cut"), k.edit.Item("Copy"), k.edit.Item("Paste")),
			menu(k.view, "View", k.view.CheckboxItem("Always show bookmarks", &k.bookmarks), ui.DropdownMenuSeparator(), k.view.Item("Reload"), k.view.Item("Full screen")),
		),
		txt("text-muted-foreground", "chosen: "+k.chosen),
	)
}

func popoverPage(c controls) twi.Node {
	k := c.kit
	p := k.popover
	size := func(label string, in *ui.Input) twi.Node {
		return el("flex flex-row items-center gap-2", el("w-7", ui.Label(twi.Text(label))), in.Node())
	}
	return show("Popover", "rich content anchored to its trigger; Escape or a click outside closes",
		p.Node(p.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Dimensions")), p.Content(
			el("flex flex-col gap-1",
				el("flex flex-col", txt("font-medium", "Dimensions"), txt("text-muted-foreground", "Set the layer size.")),
				size("Width", k.width), size("Height", k.height),
			),
		)),
	)
}

func sheetPage(c controls) twi.Node {
	return show("Sheet", "the motion page's sheet: slides in from the right edge",
		c.kit.sheet.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Open sheet")),
	)
}

func toasterPage(c controls) twi.Node {
	t := c.kit.toaster
	toast := func(label string, show func(title, description string, action ui.ToastAction)) twi.Node {
		return ui.Button(ui.Outline, ui.SizeDefault, c.clicked(func() {
			show("Event has been created", "Sunday, December 03 at 9:00", ui.ToastAction{Label: "Undo"})
		}), twi.Text(label))
	}
	return show("Toaster", "a stack in the corner; a hover holds it, Escape dismisses the newest",
		row(toast("Show", t.Show), toast("Success", t.Success), toast("Error", t.Error)),
	)
}

func tooltipPage(c controls) twi.Node {
	h := c.kit.hint
	return show("Tooltip", "on hover or focus, gone on leave, blur or Escape",
		h.Node(h.Trigger(ui.Outline, ui.SizeDefault, twi.Text("Hover")), h.Content(twi.Text("Add to library"))),
	)
}
