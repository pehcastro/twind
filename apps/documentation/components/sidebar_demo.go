package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func SidebarDemo(rt *twi.Runtime) func() twi.Node {
	sidebar, opened := ui.NewSidebar(rt), "Home"
	sidebar.Hotkey = 'e'
	open := func(name string) twi.NodeOption {
		return twi.OnClick(func(*twi.Event) {
			opened = name
			rt.Invalidate()
		})
	}
	button := func(glyph, name string) twi.Node {
		return ui.SidebarMenuButton(ui.SizeDefault, opened == name, open(name), twi.Element(twi.Class("w-1"), twi.Text(glyph)), twi.Text(name))
	}
	sub := func(name string) twi.Node {
		return ui.SidebarMenuSubItem(ui.SidebarMenuSubButton(opened == name, open(name), twi.Text(name)))
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row h-14 w-60 overflow-hidden rounded-lg border"),
			sidebar.Provider(
				sidebar.Node(
					ui.SidebarHeader(twi.Element(twi.Class("px-1 font-semibold"), twi.Text("◆ Acme Inc."))),
					ui.SidebarContent(ui.SidebarGroup(
						ui.SidebarGroupLabel(twi.Text("Application")),
						ui.SidebarGroupContent(ui.SidebarMenu(
							ui.SidebarMenuItem(button("⌂", "Home")),
							ui.SidebarMenuItem(button("▤", "Inbox"), ui.SidebarMenuBadge(twi.Text("24"))),
							ui.SidebarMenuItem(button("◎", "Settings"), ui.SidebarMenuSub(sub("Profile"), sub("Billing"))),
						)),
					)),
					ui.SidebarSeparator(),
					ui.SidebarFooter(twi.Element(twi.Class("px-1"), twi.Text("shadcn"))),
				),
				ui.SidebarInset(
					twi.Element(twi.Class("flex flex-row items-center gap-1 border-b px-1"),
						sidebar.Trigger(),
						twi.Element(twi.Class("text-muted-foreground"), twi.Text("Ctrl+E")),
					),
					twi.Element(twi.Class("px-1"), twi.Text("Open: "+opened)),
				),
			),
		)
	}
}
