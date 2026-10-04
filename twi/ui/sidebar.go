package ui

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type Sidebar struct {
	control
	Open         bool
	OnOpenChange func(bool)
	Hotkey       rune
}

func NewSidebar(rt *twi.Runtime) *Sidebar {
	return &Sidebar{control: control{rt: rt}, Open: true, Hotkey: 'b'}
}

func (s *Sidebar) Toggle() {
	s.Open = !s.Open
	notify(s.OnOpenChange, s.Open)
	s.rt.Invalidate()
}

func (s *Sidebar) Provider(children ...twi.NodeOption) twi.Node {
	hotkey := twi.OnHotkey(func(k input.KeyEvent) bool {
		pressed := k.Key == input.KeyRune && k.Rune == s.Hotkey && k.Modifiers == input.ModCtrl
		if pressed {
			s.Toggle()
		}
		return pressed
	})
	return part("group/sidebar-wrapper flex flex-row w-full h-full", append([]twi.NodeOption{hotkey}, children...))
}

func (s *Sidebar) Node(children ...twi.NodeOption) twi.Node {
	state, collapsible, width := "expanded", "", "w-26 lg:w-30"
	if !s.Open {
		state, collapsible, width = "collapsed", "icon", "w-6"
	}
	return part("group peer flex flex-col h-full shrink-0 overflow-hidden border-r border-sidebar-border bg-sidebar text-sidebar-foreground "+width, append([]twi.NodeOption{
		twi.Data("state", state), twi.Data("collapsible", collapsible), twi.Data("variant", "sidebar"), twi.Data("side", "left"),
	}, children...))
}

func (s *Sidebar) Trigger(children ...twi.NodeOption) twi.Node {
	return Button(Ghost, SizeIcon, append([]twi.NodeOption{twi.OnClick(func(*twi.Event) { s.Toggle() }), icon("◧", "")}, children...)...)
}

func SidebarInset(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col flex-1 min-w-0 bg-background", children)
}

func SidebarHeader(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col px-1 py-1", children)
}

func SidebarFooter(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col px-1 py-1", children)
}

func SidebarSeparator() twi.Node {
	return part("mx-1 shrink-0 border-t border-sidebar-border", nil)
}

func SidebarContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col flex-1 min-h-0 gap-1 overflow-auto group-data-[collapsible=icon]:overflow-hidden", children)
}

func SidebarGroup(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col w-full min-w-0 px-1 ", children)
}

func SidebarGroupLabel(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row h-1 shrink-0 items-center rounded-md px-1 font-medium text-sidebar-foreground/70 group-data-[collapsible=icon]:-mt-1 group-data-[collapsible=icon]:opacity-0", children)
}

func SidebarGroupContent(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col w-full", children)
}

func SidebarMenu(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col w-full min-w-0", children)
}

func SidebarMenuItem(children ...twi.NodeOption) twi.Node {
	return part("group/menu-item relative flex flex-col", children)
}

func SidebarMenuButton(s Size, active bool, children ...twi.NodeOption) twi.Node {
	return part("peer/menu-button flex flex-row w-full items-center gap-1 overflow-hidden rounded-md px-1 text-left whitespace-nowrap select-none hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:w-3! [&_svg]:shrink-0 "+
		pick("sidebar menu button", s, map[Size]string{
			SizeDefault: "h-1 group-data-[collapsible=icon]:px-1!",
			SizeSM:      "h-1 group-data-[collapsible=icon]:px-1!",
			SizeLG:      "h-2 group-data-[collapsible=icon]:px-0!",
		}),
		append([]twi.NodeOption{twi.Focusable(), twi.Data("active", strconv.FormatBool(active))}, children...))
}

func SidebarMenuBadge(children ...twi.NodeOption) twi.Node {
	return part("absolute top-0 right-1 flex h-1 min-w-2 items-center justify-center rounded-md px-1 font-medium text-sidebar-foreground select-none pointer-events-none peer-data-[active=true]/menu-button:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden", children)
}

func SidebarMenuSub(children ...twi.NodeOption) twi.Node {
	return part("flex flex-col min-w-0 ml-2 border-l border-sidebar-border px-1 group-data-[collapsible=icon]:hidden", children)
}

func SidebarMenuSubItem(children ...twi.NodeOption) twi.Node {
	return part("group/menu-sub-item relative flex flex-col", children)
}

func SidebarMenuSubButton(active bool, children ...twi.NodeOption) twi.Node {
	return part("flex flex-row h-1 min-w-0 items-center gap-1 overflow-hidden rounded-md px-1 whitespace-nowrap text-sidebar-foreground select-none hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden",
		append([]twi.NodeOption{twi.Focusable(), twi.Data("active", strconv.FormatBool(active))}, children...))
}

func ScrollArea(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col overflow-y-auto pr-1 focus-visible:border-ring", append([]twi.NodeOption{twi.Focusable()}, children...))
}
