package markdown

import (
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/theme"
)

const tabCells = "  "

type Highlighter struct{ grammars map[string]*highlight.Grammar }

func (h *Highlighter) grammar(language string) *highlight.Grammar {
	var name string
	var build func() *highlight.Grammar
	switch language {
	case "go":
		name, build = "go", highlight.Go
	case "bash", "sh", "shell":
		name, build = "bash", highlight.Bash
	case "console":
		name, build = "console", highlight.Console
	case "json":
		name, build = "json", highlight.JSON
	case "toml":
		name, build = "toml", highlight.TOML
	case "css":
		name, build = "css", highlight.CSS
	case "yaml":
		name, build = "yaml", highlight.YAML
	case "md", "markdown":
		name, build = "md", highlight.Markdown
	case "ts", "js":
		name, build = "ts", highlight.TypeScript
	case "tsx":
		name, build = "tsx", highlight.TSX
	default:
		return nil
	}
	if h.grammars == nil {
		h.grammars = map[string]*highlight.Grammar{}
	}
	g := h.grammars[name]
	if g == nil {
		g = build()
		h.grammars[name] = g
	}
	return g
}

func kindClass(kind highlight.Kind) string {
	switch token := highlight.DefaultPalette()[kind]; token {
	case theme.Foreground:
		return "text-foreground"
	case theme.SyntaxKeyword:
		return "text-syntax-keyword"
	case theme.SyntaxString:
		return "text-syntax-string"
	case theme.SyntaxNumber:
		return "text-syntax-number"
	case theme.SyntaxComment:
		return "text-syntax-comment"
	case theme.SyntaxFunction:
		return "text-syntax-function"
	case theme.SyntaxConstant:
		return "text-syntax-constant"
	case theme.SyntaxNamespace:
		return "text-syntax-namespace"
	case theme.SyntaxParameter:
		return "text-syntax-parameter"
	case theme.SyntaxPunctuation:
		return "text-syntax-punctuation"
	default:
		panic("markdown: no class for the syntax token " + token.String())
	}
}

func (h *Highlighter) Code(language, src string) twi.Node {
	src = strings.TrimSuffix(src, "\n")
	lines := [][]twi.NodeOption{nil}
	emit := func(span highlight.Span) {
		class := kindClass(span.Kind)
		if span.Style&highlight.StyleBold != 0 {
			class += " font-bold"
		}
		if span.Style&highlight.StyleItalic != 0 {
			class += " italic"
		}
		if span.Style&highlight.StyleStrike != 0 {
			class += " line-through"
		}
		for i, piece := range strings.Split(src[span.Start:span.End], "\n") {
			if i > 0 {
				lines = append(lines, nil)
			}
			if piece != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], twi.Element(twi.Class(class), twi.Text(strings.ReplaceAll(piece, "\t", tabCells))))
			}
		}
	}
	if g := h.grammar(language); g != nil {
		for span := range highlight.Tokens(src, g) {
			emit(span)
		}
	} else {
		emit(highlight.Span{End: len(src)})
	}
	rows := make([]twi.NodeOption, len(lines))
	for i, line := range lines {
		rows[i] = element("flex flex-row h-1 shrink-0", line)
	}
	return element("flex flex-col", rows)
}
