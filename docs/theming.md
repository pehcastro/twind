# Theming

Colours come from theme tokens, the same names shadcn/ui uses: `background`, `foreground`, `card`, `muted`, `border`, `primary`, `accent`, `destructive`, `ring` and the rest. A class such as `bg-primary` or `text-muted-foreground` follows the theme the runtime has.

<Preview name="theming-tokens" />

## Tokens

| Token | Used for |
| --- | --- |
| `background`, `foreground` | the page and its text |
| `card`, `popover` | raised surfaces, each with a `-foreground` |
| `primary`, `secondary`, `accent`, `muted` | buttons, hovers and quiet text, each with a `-foreground` |
| `destructive` | errors and dangerous actions |
| `border`, `input`, `ring` | hairlines, fields and the focus ring |
| `sidebar-*` | the sidebar's own set |
| `chart-1` to `chart-5` | series in charts |

## Built-in themes

Nine themes ship with Twind, each in a light and a dark scheme: the shadcn/ui base colours neutral, zinc, slate and stone, and the accent colours rose, blue, green, orange and violet. Press **t** in this app to try them: the arrows preview each one live, Enter keeps it and Escape puts back the one you had.

## Change the theme at runtime

```go
for _, t := range theme.Builtin() {
	if t.Name == "zinc" && t.Scheme == theme.Dark {
		rt.SetTheme(t)
	}
}
```

The next frame repaints with the new colours. Nothing is rebuilt and no class changes.

## A custom theme

A theme is a name, a scheme and an array of colours, so a custom one starts from a built-in and changes what it needs:

```go
brand := theme.Builtin()[0]
brand.Name = "brand"
brand.Tokens[theme.Primary], _ = color.Parse("oklch(0.55 0.2 260)")
brand.Tokens[theme.Ring] = brand.Tokens[theme.Primary]
rt.SetTheme(brand)
```

`color.Parse` reads the forms Tailwind writes: hex, `rgb()` and `oklch()`.

## Light and dark variants

A `dark:` class applies only under a dark theme:

```go
twi.Class("bg-white text-zinc-900 dark:bg-zinc-900 dark:text-zinc-50")
```

## Colour depth

Colours are kept as full RGBA until the frame is written. On a terminal with 256 or 16 colours they are downgraded then, and `NO_COLOR` turns colour off.
