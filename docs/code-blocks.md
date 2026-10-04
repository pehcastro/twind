# Code blocks

A fenced code block is coloured by the language named after the opening fence. The colours come from the theme's syntax tokens (`--color-syntax-keyword`, `--color-syntax-string` and the rest), so a block follows the theme and the light or dark scheme like any other text.

| Fence | Language |
| --- | --- |
| `go` | Go |
| `bash`, `sh`, `shell` | shell scripts |
| `console` | shell sessions: prompt, command and output |
| `json` | JSON |
| `toml` | TOML |
| `css` | CSS |
| `yaml` | YAML |
| `md`, `markdown` | Markdown |
| `ts`, `js` | TypeScript and JavaScript |
| `tsx` | TSX |

A fence with any other language, or none, is shown as plain text. Any page renders its fences this way through `markdown.Highlighter`:

```go
var code markdown.Highlighter
page := markdown.Render(doc, markdown.Options{Highlight: code.Code})
```

## Go

```go
func main() {
	if err := twi.Run(app); err != nil {
		log.Fatal(err)
	}
}
```

## Shell

```bash
for f in *.md; do
  echo "page: $f"
done
```

```console
$ twind build ./app
wrote app/twir_gen.go
```

## Data

```json
{ "name": "twind", "port": 8080, "debug": true }
```

```toml
[server]
port = 8080
```

```yaml
name: twind
debug: false
tags: [ui, terminal]
```

## CSS

```css
@theme {
  --color-primary: oklch(0.6 0.2 260);
}
.card:hover { padding: calc(var(--spacing) * 2); }
```

## Markdown

```md
# Title

Some **bold**, some *italic*,
some ~~struck~~ and some `code`.
```

## TypeScript

```ts
const count: number = 3
function greet(name: string): string {
  return `hi ${name}`
}
```

```tsx
export const Card = ({ title }: Props) => (
  <div className="card">{title}</div>
)
```
