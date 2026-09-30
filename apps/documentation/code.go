package docsapp

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/ui"
)

const (
	pickerRows = 9
	tabCells   = "  "
	copiedFor  = 2 * time.Second
)

type preview struct {
	tabs   *ui.Tabs
	view   func() twi.Node
	source string
	copied *twi.Timer
}

func (s *site) addPreview(name string) error {
	if _, done := s.previews[name]; done {
		return nil
	}
	demo, ok := s.demos[name]
	if !ok {
		return fmt.Errorf("no component demo registered as %q", name)
	}
	src, err := fs.ReadFile(s.source, demo.File)
	if err != nil {
		return fmt.Errorf("component demo %q: %w", name, err)
	}
	s.previews[name] = &preview{tabs: ui.NewTabs(s.rt), view: demo.New(s.rt), source: string(src)}
	return nil
}

func (s *site) previewNode(name string) twi.Node {
	p := s.previews[name]
	t := p.tabs
	label := "Copy"
	if p.copied != nil {
		label = "Copied"
	}
	copyCode := func(*twi.Event) {
		if s.rt.Copy(p.source) != nil {
			return
		}
		if p.copied != nil {
			p.copied.Stop()
		}
		p.copied = s.rt.After(copiedFor, func() {
			p.copied = nil
			s.rt.Invalidate()
		})
		s.rt.Invalidate()
	}
	return t.Node(twi.Key("preview-"+name),
		t.List(t.Trigger("preview", twi.Text("Preview")), t.Trigger("code", twi.Text("Code"))),
		t.Content("preview", el("flex flex-row min-h-9 items-center justify-center rounded-lg border px-2 py-1", p.view())),
		t.Content("code", el("relative flex flex-col rounded-lg border bg-muted py-1",
			el("px-2 whitespace-pre overflow-x-auto", s.code("go", p.source)),
			ui.Button(ui.Ghost, ui.SizeXS, twi.Key("copy-"+name), twi.Class("absolute top-0 right-1"), twi.OnClick(copyCode), twi.Text(label)),
		)),
	)
}

func (s *site) code(language, src string) twi.Node {
	src = strings.TrimSuffix(src, "\n")
	lines := [][]twi.NodeOption{nil}
	emit := func(kind highlight.Kind, text string) {
		for i, piece := range strings.Split(text, "\n") {
			if i > 0 {
				lines = append(lines, nil)
			}
			if piece != "" {
				lines[len(lines)-1] = append(lines[len(lines)-1], txt(s.colours[kind], strings.ReplaceAll(piece, "\t", tabCells)))
			}
		}
	}
	if g, ok := s.grammars[language]; ok {
		for span := range highlight.Tokens(src, g) {
			emit(span.Kind, src[span.Start:span.End])
		}
	} else {
		emit(highlight.Text, src)
	}
	rows := make([]twi.NodeOption, len(lines))
	for i, line := range lines {
		rows[i] = el("flex flex-row h-1 shrink-0", line...)
	}
	return el("flex flex-col", rows...)
}

type prop struct{ name, kind, about string }

func props() map[string][]prop {
	return map[string][]prop{
		"Button": {
			{"v", "ui.Variant", "Default, Secondary, Destructive, Outline, Ghost or Link"},
			{"s", "ui.Size", "SizeDefault, SizeXS, SizeSM, SizeLG or SizeIcon"},
			{"children", "...twi.NodeOption", "text, handlers and classes; a class merges over the defaults"},
		},
		"Dialog": {
			{"Open", "bool", "whether it is showing"},
			{"OnOpenChange", "func(bool)", "called on every open and close"},
			{"Trigger(v, s, children...)", "twi.Node", "a button that opens it"},
			{"Content(children...)", "twi.Node", "the panel, over a dimmed page"},
			{"Header, Title, Description, Footer", "twi.Node", "the panel's parts"},
			{"Close(v, s, children...)", "twi.Node", "a button that closes it"},
		},
		"Tabs": {
			{"Value", "string", "the selected tab, the first trigger by default"},
			{"OnChange", "func(string)", "called when the selection changes"},
			{"Orientation", "ui.Orientation", "Horizontal or Vertical"},
			{"List(children...)", "twi.Node", "the row of triggers"},
			{"Trigger(value, children...)", "twi.Node", "selects its panel"},
			{"Content(value, children...)", "twi.Node", "shown while its value is selected"},
		},
	}
}

func propsTable(of string) twi.Node {
	rows := [][][]markdown.Inline{{{{Text: "Name"}}, {{Text: "Type"}}, {{Text: "Description"}}}}
	for _, p := range props()[of] {
		rows = append(rows, [][]markdown.Inline{{{Style: markdown.CodeSpan, Text: p.name}}, {{Style: markdown.CodeSpan, Text: p.kind}}, {{Text: p.about}}})
	}
	return markdown.Render(markdown.Page{Blocks: []markdown.Block{{Kind: markdown.Table, Rows: rows, Align: make([]markdown.Align, 3)}}}, markdown.Options{})
}
