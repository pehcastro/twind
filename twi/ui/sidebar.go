package ui

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
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
	hotkey := twi.OnHotkey(func(e *twi.Event) {
		if e.Key.Key == input.KeyRune && e.Key.Rune == s.Hotkey && e.Key.Modifiers == input.ModCtrl {
			e.PreventDefault()
			s.Toggle()
		}
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
	return Button(ButtonGhost, ButtonSizeIcon, append([]twi.NodeOption{twi.OnClick(func(*twi.Event) { s.Toggle() }), icon("◧", "")}, children...)...)
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

func SidebarSeparator(children ...twi.NodeOption) twi.Node {
	return part("mx-1 shrink-0 border-t border-sidebar-border", children)
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

type SidebarMenuButtonSize uint8

const (
	SidebarMenuButtonSizeDefault SidebarMenuButtonSize = iota
	SidebarMenuButtonSizeSM
	SidebarMenuButtonSizeLG
)

func SidebarMenuButton(s SidebarMenuButtonSize, children ...twi.NodeOption) twi.Node {
	return part("peer/menu-button flex flex-row w-full items-center gap-1 overflow-hidden rounded-md px-1 text-left whitespace-nowrap select-none hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:w-3! [&_svg]:shrink-0 "+
		pick("sidebar menu button", s, map[SidebarMenuButtonSize]string{
			SidebarMenuButtonSizeDefault: "h-1 group-data-[collapsible=icon]:px-1!",
			SidebarMenuButtonSizeSM:      "h-1 group-data-[collapsible=icon]:px-1!",
			SidebarMenuButtonSizeLG:      "h-2 group-data-[collapsible=icon]:px-0!",
		}),
		append([]twi.NodeOption{twi.Focusable()}, children...))
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

func SidebarMenuSubButton(children ...twi.NodeOption) twi.Node {
	return part("flex flex-row h-1 min-w-0 items-center gap-1 overflow-hidden rounded-md px-1 whitespace-nowrap text-sidebar-foreground select-none hover:bg-sidebar-accent/50 focus-visible:bg-sidebar-accent focus-visible:text-sidebar-accent-foreground active:bg-sidebar-accent active:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:hidden",
		append([]twi.NodeOption{twi.Focusable()}, children...))
}

func ScrollArea(children ...twi.NodeOption) twi.Node {
	return part("relative flex flex-col overflow-y-auto pr-1 focus-visible:border-ring", append([]twi.NodeOption{twi.Focusable()}, children...))
}
