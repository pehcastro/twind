package theme

import "github.com/twind-dev/twind/twi/color"

type palette [Dark + 1][tokenEnd]string

func Builtin() []Theme {
	neutral := palette{
		Light: {
			Background: "oklch(1 0 0)", Foreground: "oklch(0.145 0 0)", Card: "oklch(1 0 0)", CardForeground: "oklch(0.145 0 0)",
			Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.145 0 0)", Primary: "oklch(0.205 0 0)", PrimaryForeground: "oklch(0.985 0 0)",
			Secondary: "oklch(0.97 0 0)", SecondaryForeground: "oklch(0.205 0 0)", Muted: "oklch(0.97 0 0)", MutedForeground: "oklch(0.556 0 0)",
			Accent: "oklch(0.97 0 0)", AccentForeground: "oklch(0.205 0 0)", Destructive: "oklch(0.577 0.245 27.325)",
			Border: "oklch(0.922 0 0)", Input: "oklch(0.922 0 0)", Ring: "oklch(0.708 0 0)",
			Chart1: "oklch(0.87 0 0)", Chart2: "oklch(0.556 0 0)", Chart3: "oklch(0.439 0 0)", Chart4: "oklch(0.371 0 0)", Chart5: "oklch(0.269 0 0)",
			Sidebar: "oklch(0.985 0 0)", SidebarForeground: "oklch(0.145 0 0)", SidebarPrimary: "oklch(0.205 0 0)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
			SidebarAccent: "oklch(0.97 0 0)", SidebarAccentForeground: "oklch(0.205 0 0)", SidebarBorder: "oklch(0.922 0 0)", SidebarRing: "oklch(0.708 0 0)",
		},
		Dark: {
			Background: "oklch(0.145 0 0)", Foreground: "oklch(0.985 0 0)", Card: "oklch(0.205 0 0)", CardForeground: "oklch(0.985 0 0)",
			Popover: "oklch(0.205 0 0)", PopoverForeground: "oklch(0.985 0 0)", Primary: "oklch(0.922 0 0)", PrimaryForeground: "oklch(0.205 0 0)",
			Secondary: "oklch(0.269 0 0)", SecondaryForeground: "oklch(0.985 0 0)", Muted: "oklch(0.269 0 0)", MutedForeground: "oklch(0.708 0 0)",
			Accent: "oklch(0.269 0 0)", AccentForeground: "oklch(0.985 0 0)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.556 0 0)",
			Chart1: "oklch(0.87 0 0)", Chart2: "oklch(0.556 0 0)", Chart3: "oklch(0.439 0 0)", Chart4: "oklch(0.371 0 0)", Chart5: "oklch(0.269 0 0)",
			Sidebar: "oklch(0.205 0 0)", SidebarForeground: "oklch(0.985 0 0)", SidebarPrimary: "oklch(0.488 0.243 264.376)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
			SidebarAccent: "oklch(0.269 0 0)", SidebarAccentForeground: "oklch(0.985 0 0)", SidebarBorder: "oklch(1 0 0 / 10%)", SidebarRing: "oklch(0.556 0 0)",
		},
	}
	zinc := palette{
		Light: {
			Background: "oklch(1 0 0)", Foreground: "oklch(0.141 0.005 285.823)", Card: "oklch(1 0 0)", CardForeground: "oklch(0.141 0.005 285.823)",
			Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.141 0.005 285.823)", Primary: "oklch(0.21 0.006 285.885)", PrimaryForeground: "oklch(0.985 0 0)",
			Secondary: "oklch(0.967 0.001 286.375)", SecondaryForeground: "oklch(0.21 0.006 285.885)", Muted: "oklch(0.967 0.001 286.375)", MutedForeground: "oklch(0.552 0.016 285.938)",
			Accent: "oklch(0.967 0.001 286.375)", AccentForeground: "oklch(0.21 0.006 285.885)", Destructive: "oklch(0.577 0.245 27.325)",
			Border: "oklch(0.92 0.004 286.32)", Input: "oklch(0.92 0.004 286.32)", Ring: "oklch(0.705 0.015 286.067)",
			Chart1: "oklch(0.871 0.006 286.286)", Chart2: "oklch(0.552 0.016 285.938)", Chart3: "oklch(0.442 0.017 285.786)", Chart4: "oklch(0.37 0.013 285.805)", Chart5: "oklch(0.274 0.006 286.033)",
			Sidebar: "oklch(0.985 0 0)", SidebarForeground: "oklch(0.141 0.005 285.823)", SidebarPrimary: "oklch(0.21 0.006 285.885)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
			SidebarAccent: "oklch(0.967 0.001 286.375)", SidebarAccentForeground: "oklch(0.21 0.006 285.885)", SidebarBorder: "oklch(0.92 0.004 286.32)", SidebarRing: "oklch(0.705 0.015 286.067)",
		},
		Dark: {
			Background: "oklch(0.141 0.005 285.823)", Foreground: "oklch(0.985 0 0)", Card: "oklch(0.21 0.006 285.885)", CardForeground: "oklch(0.985 0 0)",
			Popover: "oklch(0.21 0.006 285.885)", PopoverForeground: "oklch(0.985 0 0)", Primary: "oklch(0.92 0.004 286.32)", PrimaryForeground: "oklch(0.21 0.006 285.885)",
			Secondary: "oklch(0.274 0.006 286.033)", SecondaryForeground: "oklch(0.985 0 0)", Muted: "oklch(0.274 0.006 286.033)", MutedForeground: "oklch(0.705 0.015 286.067)",
			Accent: "oklch(0.274 0.006 286.033)", AccentForeground: "oklch(0.985 0 0)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.552 0.016 285.938)",
			Chart1: "oklch(0.871 0.006 286.286)", Chart2: "oklch(0.552 0.016 285.938)", Chart3: "oklch(0.442 0.017 285.786)", Chart4: "oklch(0.37 0.013 285.805)", Chart5: "oklch(0.274 0.006 286.033)",
			Sidebar: "oklch(0.21 0.006 285.885)", SidebarForeground: "oklch(0.985 0 0)", SidebarPrimary: "oklch(0.488 0.243 264.376)", SidebarPrimaryForeground: "oklch(0.985 0 0)",
			SidebarAccent: "oklch(0.274 0.006 286.033)", SidebarAccentForeground: "oklch(0.985 0 0)", SidebarBorder: "oklch(1 0 0 / 10%)", SidebarRing: "oklch(0.552 0.016 285.938)",
		},
	}
	stone := palette{
		Light: {
			Background: "oklch(1 0 0)", Foreground: "oklch(0.147 0.004 49.25)", Card: "oklch(1 0 0)", CardForeground: "oklch(0.147 0.004 49.25)",
			Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.147 0.004 49.25)", Primary: "oklch(0.216 0.006 56.043)", PrimaryForeground: "oklch(0.985 0.001 106.423)",
			Secondary: "oklch(0.97 0.001 106.424)", SecondaryForeground: "oklch(0.216 0.006 56.043)", Muted: "oklch(0.97 0.001 106.424)", MutedForeground: "oklch(0.553 0.013 58.071)",
			Accent: "oklch(0.97 0.001 106.424)", AccentForeground: "oklch(0.216 0.006 56.043)", Destructive: "oklch(0.577 0.245 27.325)",
			Border: "oklch(0.923 0.003 48.717)", Input: "oklch(0.923 0.003 48.717)", Ring: "oklch(0.709 0.01 56.259)",
			Chart1: "oklch(0.869 0.005 56.366)", Chart2: "oklch(0.553 0.013 58.071)", Chart3: "oklch(0.444 0.011 73.639)", Chart4: "oklch(0.374 0.01 67.558)", Chart5: "oklch(0.268 0.007 34.298)",
			Sidebar: "oklch(0.985 0.001 106.423)", SidebarForeground: "oklch(0.147 0.004 49.25)", SidebarPrimary: "oklch(0.216 0.006 56.043)", SidebarPrimaryForeground: "oklch(0.985 0.001 106.423)",
			SidebarAccent: "oklch(0.97 0.001 106.424)", SidebarAccentForeground: "oklch(0.216 0.006 56.043)", SidebarBorder: "oklch(0.923 0.003 48.717)", SidebarRing: "oklch(0.709 0.01 56.259)",
		},
		Dark: {
			Background: "oklch(0.147 0.004 49.25)", Foreground: "oklch(0.985 0.001 106.423)", Card: "oklch(0.216 0.006 56.043)", CardForeground: "oklch(0.985 0.001 106.423)",
			Popover: "oklch(0.216 0.006 56.043)", PopoverForeground: "oklch(0.985 0.001 106.423)", Primary: "oklch(0.923 0.003 48.717)", PrimaryForeground: "oklch(0.216 0.006 56.043)",
			Secondary: "oklch(0.268 0.007 34.298)", SecondaryForeground: "oklch(0.985 0.001 106.423)", Muted: "oklch(0.268 0.007 34.298)", MutedForeground: "oklch(0.709 0.01 56.259)",
			Accent: "oklch(0.268 0.007 34.298)", AccentForeground: "oklch(0.985 0.001 106.423)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.553 0.013 58.071)",
			Chart1: "oklch(0.869 0.005 56.366)", Chart2: "oklch(0.553 0.013 58.071)", Chart3: "oklch(0.444 0.011 73.639)", Chart4: "oklch(0.374 0.01 67.558)", Chart5: "oklch(0.268 0.007 34.298)",
			Sidebar: "oklch(0.216 0.006 56.043)", SidebarForeground: "oklch(0.985 0.001 106.423)", SidebarPrimary: "oklch(0.488 0.243 264.376)", SidebarPrimaryForeground: "oklch(0.985 0.001 106.423)",
			SidebarAccent: "oklch(0.268 0.007 34.298)", SidebarAccentForeground: "oklch(0.985 0.001 106.423)", SidebarBorder: "oklch(1 0 0 / 10%)", SidebarRing: "oklch(0.553 0.013 58.071)",
		},
	}
	slate := palette{
		Light: {
			Background: "#ffffff", Foreground: "#020817", Card: "#ffffff", CardForeground: "#020817",
			Popover: "#ffffff", PopoverForeground: "#020817", Primary: "#0f172a", PrimaryForeground: "#f8fafc",
			Secondary: "#f1f5f9", SecondaryForeground: "#0f172a", Muted: "#f1f5f9", MutedForeground: "#64748b",
			Accent: "#f1f5f9", AccentForeground: "#0f172a", Destructive: "#ef4444",
			Border: "#e2e8f0", Input: "#e2e8f0", Ring: "#020817",
			Chart1: "oklch(0.646 0.222 41.116)", Chart2: "oklch(0.6 0.118 184.704)", Chart3: "oklch(0.398 0.07 227.392)", Chart4: "oklch(0.828 0.189 84.429)", Chart5: "oklch(0.769 0.188 70.08)",
			Sidebar: "oklch(0.984 0.003 247.858)", SidebarForeground: "oklch(0.129 0.042 264.695)", SidebarPrimary: "oklch(0.208 0.042 265.755)", SidebarPrimaryForeground: "oklch(0.984 0.003 247.858)",
			SidebarAccent: "oklch(0.968 0.007 247.896)", SidebarAccentForeground: "oklch(0.208 0.042 265.755)", SidebarBorder: "oklch(0.929 0.013 255.508)", SidebarRing: "oklch(0.704 0.04 256.788)",
		},
		Dark: {
			Background: "#020817", Foreground: "#f8fafc", Card: "#020817", CardForeground: "#f8fafc",
			Popover: "#020817", PopoverForeground: "#f8fafc", Primary: "#f8fafc", PrimaryForeground: "#0f172a",
			Secondary: "#1e293b", SecondaryForeground: "#f8fafc", Muted: "#1e293b", MutedForeground: "#94a3b8",
			Accent: "#1e293b", AccentForeground: "#f8fafc", Destructive: "#7f1d1d",
			Border: "#1e293b", Input: "#1e293b", Ring: "#cbd5e1",
			Chart1: "oklch(0.488 0.243 264.376)", Chart2: "oklch(0.696 0.17 162.48)", Chart3: "oklch(0.769 0.188 70.08)", Chart4: "oklch(0.627 0.265 303.9)", Chart5: "oklch(0.645 0.246 16.439)",
			Sidebar: "oklch(0.208 0.042 265.755)", SidebarForeground: "oklch(0.984 0.003 247.858)", SidebarPrimary: "oklch(0.488 0.243 264.376)", SidebarPrimaryForeground: "oklch(0.984 0.003 247.858)",
			SidebarAccent: "oklch(0.279 0.041 260.031)", SidebarAccentForeground: "oklch(0.984 0.003 247.858)", SidebarBorder: "oklch(1 0 0 / 10%)", SidebarRing: "oklch(0.551 0.027 264.364)",
		},
	}
	selection := palette{
		Light: {Selection: "oklch(0% 0 0)", SelectionForeground: "oklch(1 0 0)"},
		Dark:  {Selection: "oklch(0.922 0 0)", SelectionForeground: "oklch(0.205 0 0)"},
	}
	syntax := palette{
		Light: {
			SyntaxKeyword: "oklch(0.55 0.19 22)", SyntaxString: "oklch(0.52 0.14 148)", SyntaxNumber: "oklch(0.5 0.18 256)",
			SyntaxComment: "oklch(0.53 0.035 200)", SyntaxFunction: "oklch(0.49 0.2 295)", SyntaxConstant: "oklch(0.5 0.18 256)",
			SyntaxNamespace: "oklch(0.49 0.2 295)", SyntaxParameter: "oklch(0.55 0.17 48)", SyntaxPunctuation: "oklch(0.45 0 0)",
		},
		Dark: {
			SyntaxKeyword: "oklch(0.72 0.15 18)", SyntaxString: "oklch(0.8 0.14 150)", SyntaxNumber: "oklch(0.77 0.12 252)",
			SyntaxComment: "oklch(0.7 0.035 200)", SyntaxFunction: "oklch(0.74 0.14 300)", SyntaxConstant: "oklch(0.77 0.12 252)",
			SyntaxNamespace: "oklch(0.74 0.14 300)", SyntaxParameter: "oklch(0.81 0.11 58)", SyntaxPunctuation: "oklch(0.72 0 0)",
		},
	}
	accent := func(hue string, primary, sidebarPrimary [Dark + 1]string, foreground string, chart [5]string) palette {
		p := palette{
			Light: {
				Secondary: "oklch(0.967 0.001 286.375)", SecondaryForeground: "oklch(0.21 0.006 285.885)",
				SyntaxComment: "oklch(0.53 0.04 " + hue + ")", SyntaxPunctuation: "oklch(0.45 0.02 " + hue + ")",
			},
			Dark: {
				Secondary: "oklch(0.274 0.006 286.033)", SecondaryForeground: "oklch(0.985 0 0)",
				SyntaxComment: "oklch(0.7 0.04 " + hue + ")", SyntaxPunctuation: "oklch(0.72 0.02 " + hue + ")",
			},
		}
		for scheme := Light; scheme <= Dark; scheme++ {
			p[scheme][Primary], p[scheme][PrimaryForeground] = primary[scheme], foreground
			p[scheme][SidebarPrimary], p[scheme][SidebarPrimaryForeground] = sidebarPrimary[scheme], foreground
			copy(p[scheme][Chart1:Chart5+1], chart[:])
		}
		return p
	}
	var out []Theme
	for _, p := range []struct {
		name   string
		layers []palette
	}{
		{"neutral", []palette{neutral}},
		{"zinc", []palette{zinc}},
		{"slate", []palette{slate}},
		{"stone", []palette{stone}},
		{"rose", []palette{neutral, accent("16.935",
			[Dark + 1]string{"oklch(0.514 0.222 16.935)", "oklch(0.455 0.188 13.697)"}, [Dark + 1]string{"oklch(0.586 0.253 17.585)", "oklch(0.645 0.246 16.439)"}, "oklch(0.969 0.015 12.422)",
			[5]string{"oklch(0.81 0.117 11.638)", "oklch(0.645 0.246 16.439)", "oklch(0.586 0.253 17.585)", "oklch(0.514 0.222 16.935)", "oklch(0.455 0.188 13.697)"},
		), {Dark: {Sidebar: "oklch(0.21 0.006 285.885)"}}}},
		{"blue", []palette{neutral, accent("264.376",
			[Dark + 1]string{"oklch(0.488 0.243 264.376)", "oklch(0.424 0.199 265.638)"}, [Dark + 1]string{"oklch(0.546 0.245 262.881)", "oklch(0.623 0.214 259.815)"}, "oklch(0.97 0.014 254.604)",
			[5]string{"oklch(0.809 0.105 251.813)", "oklch(0.623 0.214 259.815)", "oklch(0.546 0.245 262.881)", "oklch(0.488 0.243 264.376)", "oklch(0.424 0.199 265.638)"},
		)}},
		{"green", []palette{neutral, accent("150.069",
			[Dark + 1]string{"oklch(0.527 0.154 150.069)", "oklch(0.448 0.119 151.328)"}, [Dark + 1]string{"oklch(0.627 0.194 149.214)", "oklch(0.723 0.219 149.579)"}, "oklch(0.982 0.018 155.826)",
			[5]string{"oklch(0.871 0.15 154.449)", "oklch(0.723 0.219 149.579)", "oklch(0.627 0.194 149.214)", "oklch(0.527 0.154 150.069)", "oklch(0.448 0.119 151.328)"},
		)}},
		{"orange", []palette{neutral, accent("38.402",
			[Dark + 1]string{"oklch(0.553 0.195 38.402)", "oklch(0.47 0.157 37.304)"}, [Dark + 1]string{"oklch(0.646 0.222 41.116)", "oklch(0.705 0.213 47.604)"}, "oklch(0.98 0.016 73.684)",
			[5]string{"oklch(0.837 0.128 66.29)", "oklch(0.705 0.213 47.604)", "oklch(0.646 0.222 41.116)", "oklch(0.553 0.195 38.402)", "oklch(0.47 0.157 37.304)"},
		)}},
		{"violet", []palette{neutral, accent("292.581",
			[Dark + 1]string{"oklch(0.491 0.27 292.581)", "oklch(0.432 0.232 292.759)"}, [Dark + 1]string{"oklch(0.541 0.281 293.009)", "oklch(0.606 0.25 292.717)"}, "oklch(0.969 0.016 293.756)",
			[5]string{"oklch(0.811 0.111 293.571)", "oklch(0.606 0.25 292.717)", "oklch(0.541 0.281 293.009)", "oklch(0.491 0.27 292.581)", "oklch(0.432 0.232 292.759)"},
		)}},
	} {
		for scheme := Light; scheme <= Dark; scheme++ {
			t := Theme{Name: p.name, Scheme: scheme}
			for _, layer := range append([]palette{selection, syntax}, p.layers...) {
				for token, value := range layer[scheme] {
					if value == "" {
						continue
					}
					c, err := color.Parse(value)
					if err != nil {
						panic(err)
					}
					t.Tokens[token] = c
				}
			}
			out = append(out, t)
		}
	}
	return out
}
