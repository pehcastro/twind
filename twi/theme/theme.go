package theme

import "github.com/twind-dev/twind/twi/color"

type Token uint8

const (
	Background Token = iota + 1
	Foreground
	Card
	CardForeground
	Popover
	PopoverForeground
	Primary
	PrimaryForeground
	Secondary
	SecondaryForeground
	Muted
	MutedForeground
	Accent
	AccentForeground
	Destructive
	Border
	Input
	Ring
	Sidebar
	SidebarForeground
	SidebarPrimary
	SidebarPrimaryForeground
	SidebarAccent
	SidebarAccentForeground
	SidebarBorder
	SidebarRing
	Chart1
	Chart2
	Chart3
	Chart4
	Chart5
	Selection
	SelectionForeground
	tokenEnd
)

func (t Token) String() string {
	return [tokenEnd]string{
		Background: "background", Foreground: "foreground", Card: "card", CardForeground: "card-foreground",
		Popover: "popover", PopoverForeground: "popover-foreground", Primary: "primary", PrimaryForeground: "primary-foreground",
		Secondary: "secondary", SecondaryForeground: "secondary-foreground", Muted: "muted", MutedForeground: "muted-foreground",
		Accent: "accent", AccentForeground: "accent-foreground", Destructive: "destructive", Border: "border", Input: "input", Ring: "ring",
		Sidebar: "sidebar", SidebarForeground: "sidebar-foreground", SidebarPrimary: "sidebar-primary", SidebarPrimaryForeground: "sidebar-primary-foreground",
		SidebarAccent: "sidebar-accent", SidebarAccentForeground: "sidebar-accent-foreground", SidebarBorder: "sidebar-border", SidebarRing: "sidebar-ring",
		Chart1: "chart-1", Chart2: "chart-2", Chart3: "chart-3", Chart4: "chart-4", Chart5: "chart-5",
		Selection: "selection", SelectionForeground: "selection-foreground",
	}[t]
}

func ParseToken(name string) (Token, bool) {
	for t := Background; t < tokenEnd; t++ {
		if t.String() == name {
			return t, true
		}
	}
	return 0, false
}

type Scheme uint8

const (
	Light Scheme = iota
	Dark
)

type Tokens [tokenEnd]color.Color

type Theme struct {
	Name   string
	Scheme Scheme
	Tokens Tokens
}
