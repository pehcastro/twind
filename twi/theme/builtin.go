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
		},
		Dark: {
			Background: "oklch(0.145 0 0)", Foreground: "oklch(0.985 0 0)", Card: "oklch(0.205 0 0)", CardForeground: "oklch(0.985 0 0)",
			Popover: "oklch(0.205 0 0)", PopoverForeground: "oklch(0.985 0 0)", Primary: "oklch(0.922 0 0)", PrimaryForeground: "oklch(0.205 0 0)",
			Secondary: "oklch(0.269 0 0)", SecondaryForeground: "oklch(0.985 0 0)", Muted: "oklch(0.269 0 0)", MutedForeground: "oklch(0.708 0 0)",
			Accent: "oklch(0.269 0 0)", AccentForeground: "oklch(0.985 0 0)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.556 0 0)",
		},
	}
	zinc := palette{
		Light: {
			Background: "oklch(1 0 0)", Foreground: "oklch(0.141 0.005 285.823)", Card: "oklch(1 0 0)", CardForeground: "oklch(0.141 0.005 285.823)",
			Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.141 0.005 285.823)", Primary: "oklch(0.21 0.006 285.885)", PrimaryForeground: "oklch(0.985 0 0)",
			Secondary: "oklch(0.967 0.001 286.375)", SecondaryForeground: "oklch(0.21 0.006 285.885)", Muted: "oklch(0.967 0.001 286.375)", MutedForeground: "oklch(0.552 0.016 285.938)",
			Accent: "oklch(0.967 0.001 286.375)", AccentForeground: "oklch(0.21 0.006 285.885)", Destructive: "oklch(0.577 0.245 27.325)",
			Border: "oklch(0.92 0.004 286.32)", Input: "oklch(0.92 0.004 286.32)", Ring: "oklch(0.705 0.015 286.067)",
		},
		Dark: {
			Background: "oklch(0.141 0.005 285.823)", Foreground: "oklch(0.985 0 0)", Card: "oklch(0.21 0.006 285.885)", CardForeground: "oklch(0.985 0 0)",
			Popover: "oklch(0.21 0.006 285.885)", PopoverForeground: "oklch(0.985 0 0)", Primary: "oklch(0.92 0.004 286.32)", PrimaryForeground: "oklch(0.21 0.006 285.885)",
			Secondary: "oklch(0.274 0.006 286.033)", SecondaryForeground: "oklch(0.985 0 0)", Muted: "oklch(0.274 0.006 286.033)", MutedForeground: "oklch(0.705 0.015 286.067)",
			Accent: "oklch(0.274 0.006 286.033)", AccentForeground: "oklch(0.985 0 0)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.552 0.016 285.938)",
		},
	}
	stone := palette{
		Light: {
			Background: "oklch(1 0 0)", Foreground: "oklch(0.147 0.004 49.25)", Card: "oklch(1 0 0)", CardForeground: "oklch(0.147 0.004 49.25)",
			Popover: "oklch(1 0 0)", PopoverForeground: "oklch(0.147 0.004 49.25)", Primary: "oklch(0.216 0.006 56.043)", PrimaryForeground: "oklch(0.985 0.001 106.423)",
			Secondary: "oklch(0.97 0.001 106.424)", SecondaryForeground: "oklch(0.216 0.006 56.043)", Muted: "oklch(0.97 0.001 106.424)", MutedForeground: "oklch(0.553 0.013 58.071)",
			Accent: "oklch(0.97 0.001 106.424)", AccentForeground: "oklch(0.216 0.006 56.043)", Destructive: "oklch(0.577 0.245 27.325)",
			Border: "oklch(0.923 0.003 48.717)", Input: "oklch(0.923 0.003 48.717)", Ring: "oklch(0.709 0.01 56.259)",
		},
		Dark: {
			Background: "oklch(0.147 0.004 49.25)", Foreground: "oklch(0.985 0.001 106.423)", Card: "oklch(0.216 0.006 56.043)", CardForeground: "oklch(0.985 0.001 106.423)",
			Popover: "oklch(0.216 0.006 56.043)", PopoverForeground: "oklch(0.985 0.001 106.423)", Primary: "oklch(0.923 0.003 48.717)", PrimaryForeground: "oklch(0.216 0.006 56.043)",
			Secondary: "oklch(0.268 0.007 34.298)", SecondaryForeground: "oklch(0.985 0.001 106.423)", Muted: "oklch(0.268 0.007 34.298)", MutedForeground: "oklch(0.709 0.01 56.259)",
			Accent: "oklch(0.268 0.007 34.298)", AccentForeground: "oklch(0.985 0.001 106.423)", Destructive: "oklch(0.704 0.191 22.216)",
			Border: "oklch(1 0 0 / 10%)", Input: "oklch(1 0 0 / 15%)", Ring: "oklch(0.553 0.013 58.071)",
		},
	}
	slate := palette{
		Light: {
			Background: "#ffffff", Foreground: "#020817", Card: "#ffffff", CardForeground: "#020817",
			Popover: "#ffffff", PopoverForeground: "#020817", Primary: "#0f172a", PrimaryForeground: "#f8fafc",
			Secondary: "#f1f5f9", SecondaryForeground: "#0f172a", Muted: "#f1f5f9", MutedForeground: "#64748b",
			Accent: "#f1f5f9", AccentForeground: "#0f172a", Destructive: "#ef4444",
			Border: "#e2e8f0", Input: "#e2e8f0", Ring: "#020817",
		},
		Dark: {
			Background: "#020817", Foreground: "#f8fafc", Card: "#020817", CardForeground: "#f8fafc",
			Popover: "#020817", PopoverForeground: "#f8fafc", Primary: "#f8fafc", PrimaryForeground: "#0f172a",
			Secondary: "#1e293b", SecondaryForeground: "#f8fafc", Muted: "#1e293b", MutedForeground: "#94a3b8",
			Accent: "#1e293b", AccentForeground: "#f8fafc", Destructive: "#7f1d1d",
			Border: "#1e293b", Input: "#1e293b", Ring: "#cbd5e1",
		},
	}
	accent := func(light, lightForeground, dark, darkForeground string) palette {
		return palette{
			Light: {Primary: light, PrimaryForeground: lightForeground, Secondary: "oklch(0.967 0.001 286.375)", SecondaryForeground: "oklch(0.21 0.006 285.885)"},
			Dark:  {Primary: dark, PrimaryForeground: darkForeground, Secondary: "oklch(0.274 0.006 286.033)", SecondaryForeground: "oklch(0.985 0 0)"},
		}
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
		{"rose", []palette{neutral, accent("oklch(0.514 0.222 16.935)", "oklch(0.969 0.015 12.422)", "oklch(0.455 0.188 13.697)", "oklch(0.969 0.015 12.422)")}},
		{"blue", []palette{neutral, accent("oklch(0.488 0.243 264.376)", "oklch(0.97 0.014 254.604)", "oklch(0.424 0.199 265.638)", "oklch(0.97 0.014 254.604)")}},
		{"green", []palette{neutral, accent("oklch(0.527 0.154 150.069)", "oklch(0.982 0.018 155.826)", "oklch(0.448 0.119 151.328)", "oklch(0.982 0.018 155.826)")}},
		{"orange", []palette{neutral, accent("oklch(0.553 0.195 38.402)", "oklch(0.98 0.016 73.684)", "oklch(0.47 0.157 37.304)", "oklch(0.98 0.016 73.684)")}},
		{"violet", []palette{neutral, accent("oklch(0.491 0.27 292.581)", "oklch(0.969 0.016 293.756)", "oklch(0.432 0.232 292.759)", "oklch(0.969 0.016 293.756)")}},
	} {
		for scheme := Light; scheme <= Dark; scheme++ {
			t := Theme{Name: p.name, Scheme: scheme}
			for _, layer := range p.layers {
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
