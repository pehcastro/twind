# Theming

Colours come from theme tokens, the same names shadcn/ui uses: `background`, `foreground`, `card`, `muted`, `border`, `primary`, `accent`, `destructive`, `ring` and the rest. A class such as `bg-primary` or `text-muted-foreground` follows the theme the runtime has.

## Built-in themes

Nine themes ship with Twind, each in a light and a dark scheme: the shadcn/ui base colours neutral, zinc, slate and stone, and the accent colours rose, blue, green, orange and violet. Press **t** in this app to try them.

## Change the theme at runtime

```go
for _, t := range theme.Builtin() {
	if t.Name == "zinc" && t.Scheme == theme.Dark {
		rt.SetTheme(t)
	}
}
```

The next frame repaints with the new colours. Nothing is rebuilt and no class changes.

## Light and dark variants

A `dark:` class applies only under a dark theme:

```go
twi.Class("bg-white text-zinc-900 dark:bg-zinc-900 dark:text-zinc-50")
```

## Colour depth

Colours are kept as full RGBA until the frame is written. On a terminal with 256 or 16 colours they are downgraded then, and `NO_COLOR` turns colour off.
