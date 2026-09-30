package main

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func shell(rt *twi.Runtime, open string) func() twi.Node {
	side, bar, card := ui.NewSidebar(rt), ui.NewMenubar(rt), ui.NewContextMenu(rt)
	file, edit, view, profiles := bar.Menu(), bar.Menu(), bar.Menu(), bar.Menu()
	share, find, tools := file.Sub(), edit.Sub(), card.Sub()
	sections := []*ui.Collapsible{ui.NewCollapsible(rt), ui.NewCollapsible(rt), ui.NewCollapsible(rt), ui.NewCollapsible(rt)}
	sections[0].Open = true
	bookmarksBar, fullURLs, showBookmarks, showURLs := false, true, true, false
	profile, person, chosen := "Benoit", "Pedro Duarte", "nothing yet"
	for _, m := range []*ui.DropdownMenu{&file.DropdownMenu, &edit.DropdownMenu, &view.DropdownMenu, &profiles.DropdownMenu, &card.DropdownMenu} {
		m.OnSelect = func(item string) { chosen = item }
	}
	switch open {
	case "menubar":
		file.Open = true
	case "context":
		card.Open = true
	case "collapsed":
		side.Open = false
	}
	shortcut := func(s string) twi.Node { return ui.DropdownMenuShortcut(label(s)) }
	inset := twi.Class("pl-4")
	nav := func(section *ui.Collapsible, glyph, title string, items ...string) twi.Node {
		chevron := "›"
		if section.Open {
			chevron = "⌄"
		}
		var subs []twi.NodeOption
		for _, s := range items {
			subs = append(subs, ui.SidebarMenuSubItem(ui.SidebarMenuSubButton(false, label(s))))
		}
		return section.Node(ui.SidebarMenuItem(
			ui.SidebarMenuButton(ui.SizeDefault, false, section.AsTrigger(), text("shrink-0", glyph), el("flex-1", label(title)), text("text-muted-foreground", chevron)),
			section.Content(ui.SidebarMenuSub(subs...)),
		))
	}
	project := func(glyph, name string, badge ...twi.NodeOption) twi.Node {
		return ui.SidebarMenuItem(append([]twi.NodeOption{ui.SidebarMenuButton(ui.SizeDefault, false, text("shrink-0", glyph), label(name))}, badge...)...)
	}
	titled := func(title, detail string) twi.Node {
		return el("flex flex-col flex-1 min-w-0", text("font-medium truncate", title), text("truncate text-muted-foreground", detail))
	}
	tags := []twi.NodeOption{twi.Class("w-26 rounded-md border"), text("px-2 font-medium", "Tags")}
	for i := 50; i > 0; i-- {
		tags = append(tags, text("px-2", "v1.2.0-beta."+strconv.Itoa(i)), ui.Separator(ui.Horizontal, twi.Class("mx-2 w-auto")))
	}
	return func() twi.Node {
		return el("flex flex-row flex-1 -mx-3 -my-1", side.Provider(
			side.Node(
				ui.SidebarHeader(ui.SidebarMenu(ui.SidebarMenuItem(ui.SidebarMenuButton(ui.SizeLG, false,
					text("flex h-2 w-3 shrink-0 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground", "▣"),
					titled("Acme Inc", "Enterprise"), label("⇅"),
				)))),
				ui.SidebarContent(
					ui.SidebarGroup(ui.SidebarGroupLabel(label("Platform")), ui.SidebarMenu(
						nav(sections[0], "▸", "Playground", "History", "Starred", "Settings"),
						nav(sections[1], "◉", "Models", "Genesis", "Explorer", "Quantum"),
						nav(sections[2], "▤", "Documentation", "Introduction", "Get Started", "Tutorials", "Changelog"),
						nav(sections[3], "✲", "Settings", "General", "Team", "Billing", "Limits"),
					)),
					ui.SidebarGroup(ui.SidebarGroupLabel(label("Projects")), ui.SidebarMenu(
						project("▢", "Design Engineering"),
						project("◔", "Sales & Marketing", ui.SidebarMenuBadge(label("12"))),
						project("◇", "Travel"),
						project("…", "More"),
					)),
				),
				ui.SidebarFooter(ui.SidebarMenu(ui.SidebarMenuItem(ui.SidebarMenuButton(ui.SizeLG, false,
					ui.Avatar(ui.SizeSM, twi.Class("h-2 w-3 rounded-lg"), ui.AvatarFallback(twi.Class("rounded-lg"), label("CN"))),
					titled("shadcn", "m@example.com"), label("⇅"),
				)))),
			),
			ui.SidebarInset(
				el("flex flex-row h-3 shrink-0 items-center gap-1 px-2",
					side.Trigger(twi.Class("-ml-1")),
					ui.Separator(ui.Vertical, twi.Class("mr-1 h-1 self-center")),
					ui.Breadcrumb(ui.BreadcrumbList(
						ui.BreadcrumbItem(twi.Class("hidden md:flex"), ui.BreadcrumbLink(label("Build Your Application"))), ui.BreadcrumbSeparator(twi.Class("hidden md:flex")),
						ui.BreadcrumbItem(ui.BreadcrumbPage(label("Data Fetching"))),
					)),
				),
				el("flex flex-col flex-1 min-h-0 gap-1 px-2 pb-1",
					bar.Node(twi.Class("self-start"),
						file.Node(file.Trigger(label("File")), file.Content(
							file.Item("New Tab", shortcut("⌘T")), file.Item("New Window", shortcut("⌘N")), file.Item("New Incognito Window"),
							ui.DropdownMenuSeparator(),
							share.Node(share.Trigger("Share"), share.Content(share.Item("Email link"), share.Item("Messages"), share.Item("Notes"))),
							ui.DropdownMenuSeparator(),
							file.Item("Print...", shortcut("⌘P")),
						)),
						edit.Node(edit.Trigger(label("Edit")), edit.Content(
							edit.Item("Undo", shortcut("⌘Z")), edit.Item("Redo", shortcut("⇧⌘Z")),
							ui.DropdownMenuSeparator(),
							find.Node(find.Trigger("Find"), find.Content(find.Item("Search the web"), ui.DropdownMenuSeparator(), find.Item("Find..."), find.Item("Find Next"), find.Item("Find Previous"))),
							ui.DropdownMenuSeparator(),
							edit.Item("Cut"), edit.Item("Copy"), edit.Item("Paste"),
						)),
						view.Node(view.Trigger(label("View")), view.Content(
							view.CheckboxItem("Always Show Bookmarks Bar", &bookmarksBar), view.CheckboxItem("Always Show Full URLs", &fullURLs),
							ui.DropdownMenuSeparator(),
							view.Item("Reload", inset, shortcut("⌘R")), view.Item("Force Reload", inset, shortcut("⇧⌘R")),
							ui.DropdownMenuSeparator(),
							view.Item("Toggle Fullscreen", inset),
							ui.DropdownMenuSeparator(),
							view.Item("Hide Sidebar", inset),
						)),
						profiles.Node(profiles.Trigger(label("Profiles")), profiles.Content(
							profiles.RadioItem("Andy", &profile), profiles.RadioItem("Benoit", &profile), profiles.RadioItem("Luis", &profile),
							ui.DropdownMenuSeparator(),
							profiles.Item("Edit...", inset),
							ui.DropdownMenuSeparator(),
							profiles.Item("Add Profile...", inset),
						)),
					),
					el("grid grid-cols-3 gap-2",
						card.Node(twi.Class("h-7"), card.Trigger(twi.Class("flex-1 items-center justify-center rounded-md border border-dashed"), label("Right click here")), card.Content(twi.Class("w-26"),
							card.Item("Back", inset, shortcut("⌘[")), card.Item("Forward", inset, shortcut("⌘]")), card.Item("Reload", inset, shortcut("⌘R")),
							tools.Node(tools.Trigger("More Tools", inset), tools.Content(twi.Class("w-22"),
								tools.Item("Save Page..."), tools.Item("Create Shortcut..."), tools.Item("Name Window..."),
								ui.DropdownMenuSeparator(), tools.Item("Developer Tools"), ui.DropdownMenuSeparator(), tools.Item("Delete", twi.Class("text-destructive")),
							)),
							ui.DropdownMenuSeparator(),
							card.CheckboxItem("Show Bookmarks", &showBookmarks), card.CheckboxItem("Show Full URLs", &showURLs),
							ui.DropdownMenuSeparator(),
							ui.DropdownMenuLabel(inset, label("People")),
							card.RadioItem("Pedro Duarte", &person), card.RadioItem("Colm Tuite", &person),
						)),
						el("h-7 rounded-xl bg-muted/50"),
						el("h-7 rounded-xl bg-muted/50"),
					),
					el("flex flex-row flex-1 min-h-0 gap-2",
						el("flex flex-col flex-1 gap-1 rounded-xl bg-muted/50 px-2 py-1",
							text("font-medium", "Data Fetching"),
							text("text-muted-foreground", "Ctrl+B folds the sidebar to its icons. A right click or Shift+F10 on the dashed card opens its menu. The menubar takes Left and Right between menus."),
							text("text-muted-foreground", "Chosen: "+chosen),
						),
						ui.ScrollArea(tags...),
					),
				),
			),
		))
	}
}
