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
	RemPixels       = 16
	NominalCellX    = RemPixels / 2
	NominalCellY    = RemPixels
	EaseX1          = 0.25
	EaseY1          = 0.1
	EaseX2          = 0.25
	EaseY2          = 1
	RingOffsetWhite = 255
	ShadeEntries    = 1024
	ShadeWays       = 4
	MaxGridTracks   = 1000
	MaxMarkers      = 64
	ClassSlots      = 4
	KeyBytes        = 32
	MaxRules        = 1<<15 - 1
	HashA           = 0xa0761d6478bd642f
	HashB           = 0xe7037ed1a0b428db
	CornerBits      = 4
	CornerMask      = 1<<CornerBits - 1
	EveryCorner     = 0x1111
	RampSide        = 5
	RampLightest    = 0.97
	RampDarkest     = 0.28
	RampSpan        = 0.1
	RampEase        = 1.2
	RampLightFade   = 0.85
	RampDarkFade    = 0.45
)

const PresetTheme = `@theme {
  --spacing: 1px;
  --breakpoint-sm: 80px;
  --breakpoint-md: 100px;
  --breakpoint-lg: 140px;
  --breakpoint-xl: 180px;
  --breakpoint-2xl: 220px;
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
  --color-destructive-foreground: var(--color-zinc-50);
  --color-border: var(--color-zinc-200);
  --color-input: var(--color-zinc-200);
  --color-ring: var(--color-zinc-400);
  --color-sidebar: var(--color-zinc-50);
  --color-sidebar-foreground: var(--color-zinc-950);
  --color-sidebar-primary: var(--color-zinc-900);
  --color-sidebar-primary-foreground: var(--color-zinc-50);
  --color-sidebar-accent: var(--color-zinc-100);
  --color-sidebar-accent-foreground: var(--color-zinc-900);
  --color-sidebar-border: var(--color-zinc-200);
  --color-sidebar-ring: var(--color-zinc-400);
  --color-chart-1: var(--color-zinc-300);
  --color-chart-2: var(--color-zinc-500);
  --color-chart-3: var(--color-zinc-600);
  --color-chart-4: var(--color-zinc-700);
  --color-chart-5: var(--color-zinc-800);
  --color-selection: var(--color-black);
  --color-selection-foreground: var(--color-white);
  --color-syntax-keyword: oklch(0.55 0.19 22);
  --color-syntax-string: oklch(0.52 0.14 148);
  --color-syntax-number: oklch(0.5 0.18 256);
  --color-syntax-comment: oklch(0.53 0.035 200);
  --color-syntax-function: oklch(0.49 0.2 295);
  --color-syntax-constant: oklch(0.5 0.18 256);
  --color-syntax-namespace: oklch(0.49 0.2 295);
  --color-syntax-parameter: oklch(0.55 0.17 48);
  --color-syntax-punctuation: oklch(0.45 0 0);
  --color-primary-50: var(--color-zinc-50);
  --color-primary-100: var(--color-zinc-100);
  --color-primary-200: var(--color-zinc-200);
  --color-primary-300: var(--color-zinc-300);
  --color-primary-400: var(--color-zinc-400);
  --color-primary-500: var(--color-zinc-500);
  --color-primary-600: var(--color-zinc-600);
  --color-primary-700: var(--color-zinc-700);
  --color-primary-800: var(--color-zinc-800);
  --color-primary-900: var(--color-zinc-900);
  --color-primary-950: var(--color-zinc-950);
  --color-accent-50: var(--color-zinc-50);
  --color-accent-100: var(--color-zinc-100);
  --color-accent-200: var(--color-zinc-200);
  --color-accent-300: var(--color-zinc-300);
  --color-accent-400: var(--color-zinc-400);
  --color-accent-500: var(--color-zinc-500);
  --color-accent-600: var(--color-zinc-600);
  --color-accent-700: var(--color-zinc-700);
  --color-accent-800: var(--color-zinc-800);
  --color-accent-900: var(--color-zinc-900);
  --color-accent-950: var(--color-zinc-950);
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
      --color-sidebar: var(--color-zinc-900);
      --color-sidebar-foreground: var(--color-zinc-50);
      --color-sidebar-primary: var(--color-blue-700);
      --color-sidebar-primary-foreground: var(--color-zinc-50);
      --color-sidebar-accent: var(--color-zinc-800);
      --color-sidebar-accent-foreground: var(--color-zinc-50);
      --color-sidebar-border: oklch(100% 0 0 / 10%);
      --color-sidebar-ring: var(--color-zinc-500);
      --color-selection: var(--color-neutral-200);
      --color-selection-foreground: var(--color-neutral-900);
      --color-syntax-keyword: oklch(0.72 0.15 18);
      --color-syntax-string: oklch(0.8 0.14 150);
      --color-syntax-number: oklch(0.77 0.12 252);
      --color-syntax-comment: oklch(0.7 0.035 200);
      --color-syntax-function: oklch(0.74 0.14 300);
      --color-syntax-constant: oklch(0.77 0.12 252);
      --color-syntax-namespace: oklch(0.74 0.14 300);
      --color-syntax-parameter: oklch(0.81 0.11 58);
      --color-syntax-punctuation: oklch(0.72 0 0);
    }
  }
}
@layer base {
  *, ::after, ::before, ::backdrop, ::file-selector-button {
    border-color: var(--color-border);
  }
}
`
