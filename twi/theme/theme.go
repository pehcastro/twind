package theme

//go:generate go run ./gen -pkg theme -out builtin_gen.go css

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
	SyntaxKeyword
	SyntaxString
	SyntaxNumber
	SyntaxComment
	SyntaxFunction
	SyntaxConstant
	SyntaxNamespace
	SyntaxParameter
	SyntaxPunctuation
	DestructiveForeground
	Primary50
	Primary100
	Primary200
	Primary300
	Primary400
	Primary500
	Primary600
	Primary700
	Primary800
	Primary900
	Primary950
	Accent50
	Accent100
	Accent200
	Accent300
	Accent400
	Accent500
	Accent600
	Accent700
	Accent800
	Accent900
	Accent950
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
		SyntaxKeyword: "syntax-keyword", SyntaxString: "syntax-string", SyntaxNumber: "syntax-number", SyntaxComment: "syntax-comment",
		SyntaxFunction: "syntax-function", SyntaxConstant: "syntax-constant", SyntaxNamespace: "syntax-namespace",
		SyntaxParameter: "syntax-parameter", SyntaxPunctuation: "syntax-punctuation", DestructiveForeground: "destructive-foreground",
		Primary50: "primary-50", Primary100: "primary-100", Primary200: "primary-200", Primary300: "primary-300", Primary400: "primary-400", Primary500: "primary-500",
		Primary600: "primary-600", Primary700: "primary-700", Primary800: "primary-800", Primary900: "primary-900", Primary950: "primary-950",
		Accent50: "accent-50", Accent100: "accent-100", Accent200: "accent-200", Accent300: "accent-300", Accent400: "accent-400", Accent500: "accent-500",
		Accent600: "accent-600", Accent700: "accent-700", Accent800: "accent-800", Accent900: "accent-900", Accent950: "accent-950",
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
	Name    string
	Scheme  Scheme
	Tokens  Tokens
	Schemes [Dark + 1]Tokens
}

func (t Theme) WithScheme(s Scheme) Theme {
	t.Scheme, t.Tokens = s, t.Schemes[s]
	return t
}

func Default() Theme { return twind() }

func Builtin() []Theme {
	var out []Theme
	for _, t := range []Theme{twind(), dream(), mono(), minimal(), dew(), cloud(), sukuna()} {
		out = append(out, t.WithScheme(Light), t.WithScheme(Dark))
	}
	return out
}
