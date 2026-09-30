package style

const (
	IRMagic         = "TWIR"
	IRVersion       = 1
	TailwindVersion = "4.3.3"
	OpaquePercent   = 100
	NormalWeight    = 400
	BoldMinWeight   = 600
	BoldWeight      = 700
	MaxVarDepth     = 32
	GeneratedFile   = "twir_gen.go"
	FromPosition    = 0
	ViaPosition     = 0.5
	ToPosition      = 1
)

const PresetTheme = `@theme {
  --spacing: 1px;
  --breakpoint-sm: 80px;
  --breakpoint-md: 100px;
  --breakpoint-lg: 140px;
  --breakpoint-xl: 180px;
  --shadow-2xs: 0 1px 0 0 oklch(0% 0 0 / 0.03);
  --shadow-xs: 0 1px 0 0 oklch(0% 0 0 / 0.05);
  --shadow-sm: 1px 1px 0 0 oklch(0% 0 0 / 0.05);
  --shadow: 1px 1px 0 0 oklch(0% 0 0 / 0.05);
  --shadow-md: 1px 1px 0 0 oklch(0% 0 0 / 0.07);
  --shadow-lg: 1px 1px 1px 0 oklch(0% 0 0 / 0.1);
  --shadow-xl: 2px 1px 1px 0 oklch(0% 0 0 / 0.12);
  --shadow-2xl: 2px 1px 2px 0 oklch(0% 0 0 / 0.25);
  --inset-shadow-2xs: inset 0 1px 0 0 oklch(0% 0 0 / 0.03);
  --inset-shadow-xs: inset 0 1px 0 0 oklch(0% 0 0 / 0.05);
  --inset-shadow-sm: inset 0 1px 0 0 oklch(0% 0 0 / 0.07);
  --shadow-inner: inset 0 1px 0 0 oklch(0% 0 0 / 0.07);
  --color-background: var(--color-white);
  --color-foreground: var(--color-zinc-950);
  --color-card: var(--color-white);
  --color-card-foreground: var(--color-zinc-950);
  --color-popover: var(--color-white);
  --color-popover-foreground: var(--color-zinc-950);
  --color-primary: var(--color-zinc-900);
  --color-primary-foreground: var(--color-zinc-50);
  --color-secondary: var(--color-zinc-100);
  --color-secondary-foreground: var(--color-zinc-900);
  --color-muted: var(--color-zinc-100);
  --color-muted-foreground: var(--color-zinc-500);
  --color-accent: var(--color-zinc-100);
  --color-accent-foreground: var(--color-zinc-900);
  --color-destructive: var(--color-red-600);
  --color-border: var(--color-zinc-200);
  --color-input: var(--color-zinc-200);
  --color-ring: var(--color-zinc-400);
}
@layer theme {
  :root, :host {
    @variant dark {
      --color-background: var(--color-zinc-950);
      --color-foreground: var(--color-zinc-50);
      --color-card: var(--color-zinc-900);
      --color-card-foreground: var(--color-zinc-50);
      --color-popover: var(--color-zinc-900);
      --color-popover-foreground: var(--color-zinc-50);
      --color-primary: var(--color-zinc-200);
      --color-primary-foreground: var(--color-zinc-900);
      --color-secondary: var(--color-zinc-800);
      --color-secondary-foreground: var(--color-zinc-50);
      --color-muted: var(--color-zinc-800);
      --color-muted-foreground: var(--color-zinc-400);
      --color-accent: var(--color-zinc-800);
      --color-accent-foreground: var(--color-zinc-50);
      --color-destructive: var(--color-red-400);
      --color-border: oklch(100% 0 0 / 10%);
      --color-input: oklch(100% 0 0 / 15%);
      --color-ring: var(--color-zinc-500);
    }
  }
}
@layer base {
  *, ::after, ::before, ::backdrop, ::file-selector-button {
    border-color: var(--color-border);
  }
}
`
