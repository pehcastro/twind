package ui

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/theme"
)

func TestSpacingIconSlot(t *testing.T) {
	d := overlayDriver(t, 80, 30, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Light))
		m, s, p := NewDropdownMenu(rt), NewSelect(rt), NewCommand(rt)
		m.Open, s.Open, m.Align = true, true, Start
		mark := func(glyph string) ItemIcon { return ItemIcon{Node: icon(glyph, "")} }
		return func() twi.Node {
			column := func(n twi.Node) twi.Node { return twi.Element(twi.Class("flex w-25 shrink-0"), n) }
			return twi.Element(twi.Class("flex flex-row p-1 h-full bg-background text-foreground"),
				column(m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Menu")), m.Content(m.Item("Profile", DropdownMenuShortcut(twi.Text("⌘P")), mark("◉"))))),
				column(s.Node(s.Trigger(), s.Content(s.Item("apple", "Apple", mark("◆"))))),
				column(p.Node(p.List(p.Group("", p.Item("Calendar", mark("▦")), p.Item("Mail", twi.Text("Mail"), mark("✉")))))),
			)
		}
	})
	for _, want := range []string{"◉ Profile", "◆ Apple", "▦ Calendar", "✉ Mail"} {
		if !strings.Contains(d.Frame().Text(), want) {
			t.Errorf("no %q: an item's icon goes before its label, one cell apart:\n%s", want, d.Frame().Text())
		}
	}
}

func TestSpacing(t *testing.T) {
	type list struct {
		name                   string
		heading, first, second string
		after                  string
		app                    func(rt *twi.Runtime) func() twi.Node
	}
	page := func(children ...twi.NodeOption) twi.Node {
		return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-col p-1 h-full bg-background text-foreground")}, children...)...)
	}
	menu := func(l *menuLevel) []twi.NodeOption {
		return []twi.NodeOption{DropdownMenuLabel(twi.Text("Account")), l.Item("Profile"), l.Item("Billing"), DropdownMenuSeparator(), DropdownMenuLabel(twi.Text("Team")), l.Item("Invite")}
	}
	for _, c := range []list{
		{"command", "Suggestions", "Calendar", "Emoji", "Settings", func(rt *twi.Runtime) func() twi.Node {
			p := NewCommand(rt)
			return func() twi.Node {
				return page(p.Node(twi.Class("border"), p.Input("Search"), p.List(
					p.Group("Suggestions", p.Item("Calendar"), p.Item("Emoji")),
					p.Group("Settings", p.Item("Profile")),
				)))
			}
		}},
		{"dropdown menu", "Account", "Profile", "Billing", "Team", func(rt *twi.Runtime) func() twi.Node {
			m := NewDropdownMenu(rt)
			m.Open = true
			return func() twi.Node {
				return page(m.Node(m.Trigger(Outline, SizeDefault, twi.Text("Open")), m.Content(menu(&m.menuLevel)...)))
			}
		}},
		{"context menu", "Account", "Profile", "Billing", "Team", func(rt *twi.Runtime) func() twi.Node {
			m := NewContextMenu(rt)
			m.Open = true
			return func() twi.Node {
				return page(m.Node(m.Trigger(twi.Text("Right click here")), m.Content(menu(&m.menuLevel)...)))
			}
		}},
		{"menubar", "Account", "Profile", "Billing", "Team", func(rt *twi.Runtime) func() twi.Node {
			b := NewMenubar(rt)
			m := b.Menu()
			m.Open = true
			return func() twi.Node {
				return page(b.Node(m.Node(m.Trigger(twi.Text("File")), m.Content(menu(&m.menuLevel)...))))
			}
		}},
		{"select", "Fruits", "Apple", "Banana", "Vegetables", func(rt *twi.Runtime) func() twi.Node {
			s := NewSelect(rt)
			s.Open = true
			return func() twi.Node {
				return page(s.Node(s.Trigger(), s.Content(
					SelectLabel(twi.Text("Fruits")), s.Item("apple", "Apple"), s.Item("banana", "Banana"),
					SelectSeparator(), SelectLabel(twi.Text("Vegetables")), s.Item("leek", "Leek"),
				)))
			}
		}},
		{"combobox", "Fruits", "Apple", "Banana", "Vegetables", func(rt *twi.Runtime) func() twi.Node {
			c := NewCombobox(rt)
			c.Open = true
			return func() twi.Node {
				return page(c.Node(c.Input(), c.Content(
					SelectLabel(twi.Text("Fruits")), c.Item("apple", "Apple"), c.Item("banana", "Banana"),
					SelectSeparator(), SelectLabel(twi.Text("Vegetables")), c.Item("leek", "Leek"),
				)))
			}
		}},
		{"navigation menu", "", "Introduction", "Installation", "", func(rt *twi.Runtime) func() twi.Node {
			n := NewNavigationMenu(rt)
			i := n.Item("docs")
			n.Value = "docs"
			return func() twi.Node {
				return page(n.Node(n.List(i.Node(i.Trigger(twi.Text("Docs")), i.Content(
					i.Link("/intro", twi.Text("Introduction")), i.Link("/install", twi.Text("Installation")),
				)))))
			}
		}},
		{"sidebar", "Platform", "Home", "Inbox", "Projects", func(rt *twi.Runtime) func() twi.Node {
			s := NewSidebar(rt)
			button := func(text string) twi.Node {
				return SidebarMenuItem(SidebarMenuButton(SizeDefault, false, twi.Text(text)))
			}
			return func() twi.Node {
				return page(s.Provider(s.Node(SidebarContent(
					SidebarGroup(SidebarGroupLabel(twi.Text("Platform")), SidebarGroupContent(SidebarMenu(button("Home"), button("Inbox")))),
					SidebarGroup(SidebarGroupLabel(twi.Text("Projects")), SidebarGroupContent(SidebarMenu(button("Design")))),
				))))
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := overlayDriver(t, 80, 30, func(rt *twi.Runtime) func() twi.Node {
				rt.SetTheme(zinc(t, theme.Light))
				return c.app(rt)
			})
			row := func(s string) int {
				t.Helper()
				_, y, ok := at(d.Frame(), s)
				if !ok {
					t.Fatalf("%q is not on screen:\n%s", s, d.Frame().Text())
				}
				return y
			}
			first, second := row(c.first), row(c.second)
			if second-first != 1 {
				t.Errorf("%s and %s are on rows %d and %d, want adjacent rows:\n%s", c.first, c.second, first, second, d.Frame().Text())
			}
			if x, y, _ := at(d.Frame(), c.first); c.name != "sidebar" && !strings.HasSuffix(string([]rune(strings.Split(d.Frame().Text(), "\n")[y])[:x]), "│ ") {
				t.Errorf("%s at column %d is not one cell right of the panel border:\n%s", c.first, x, d.Frame().Text())
			}
			if c.heading == "" {
				return
			}
			if heading := row(c.heading); first-heading != 1 {
				t.Errorf("heading %s on row %d is not right above %s on row %d:\n%s", c.heading, heading, c.first, first, d.Frame().Text())
			}
			if after := row(c.after); after-second != 2 {
				t.Errorf("the next group's %s on row %d is not two rows below %s on row %d, one separator or blank row between groups:\n%s", c.after, after, c.second, second, d.Frame().Text())
			}
		})
	}
}
