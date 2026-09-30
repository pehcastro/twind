# Sidebar

A column of navigation beside the main area, collapsible to icons.

<Preview name="sidebar-demo" />

## Usage

```go
sidebar := ui.NewSidebar(rt)
```

```go
sidebar.Provider(
	sidebar.Node(ui.SidebarContent(ui.SidebarGroup(
		ui.SidebarGroupLabel(twi.Text("Application")),
		ui.SidebarGroupContent(ui.SidebarMenu(
			ui.SidebarMenuItem(ui.SidebarMenuButton(ui.SizeDefault, true, twi.Text("Home"))),
		)),
	))),
	ui.SidebarInset(page),
)
```

Ctrl+B or the trigger collapses it. The sidebar of this site is one; the demo uses Ctrl+E so the two do not share a key. Its content scrolls when it is taller than the screen.

## API reference

<Props of="Sidebar" />
