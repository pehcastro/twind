package docsapp

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	"github.com/twind-dev/twind/apps/documentation/components"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/highlight"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/ui"
)

const (
	titleBytes = 128
	tabCells   = "  "
	copiedFor  = 2 * time.Second
)

type preview struct {
	tabs   *ui.Tabs
	build  func(*twi.Runtime) func() twi.Node
	view   func() twi.Node
	source string
}

func (s *site) copy(code string) {
	if s.rt.Copy(code) != nil {
		return
	}
	if s.copying != nil {
		s.copying.Stop()
	}
	s.copied, s.copying = code, s.rt.After(copiedFor, func() {
		s.copied, s.copying = "", nil
		s.rt.Invalidate()
	})
	s.rt.Invalidate()
}

func (s *site) addPreview(name string) error {
	if _, done := s.previews[name]; done {
		return nil
	}
	i := slices.IndexFunc(s.catalogs, func(c components.Catalog) bool { return c.Demos[name] != nil })
	if i < 0 {
		return fmt.Errorf("no demo registered as %q", name)
	}
	src, err := fs.ReadFile(s.catalogs[i].Source, strings.ReplaceAll(name, "-", "_")+".go")
	if err != nil {
		return fmt.Errorf("demo %q: %w", name, err)
	}
	s.previews[name] = &preview{tabs: ui.NewTabs(s.rt), build: s.catalogs[i].Demos[name], source: string(src)}
	return nil
}

func (s *site) previewNode(name string) twi.Node {
	p := s.previews[name]
	if p.view == nil {
		p.view = p.build(s.rt)
	}
	t := p.tabs
	label := "Copy"
	if s.copied == p.source {
		label = "Copied"
	}
	return t.Node(twi.Key("preview-"+name),
		t.List(t.Trigger("preview", twi.Text("Preview")), t.Trigger("code", twi.Text("Code"))),
		t.Content("preview", el("flex flex-row min-h-9 min-w-0 items-center justify-center overflow-hidden rounded-lg border px-2 py-1", p.view())),
		t.Content("code", el("relative flex flex-col rounded-lg border bg-muted py-1",
			el("px-2 whitespace-pre overflow-x-auto", s.code("go", p.source)),
			ui.Button(ui.Ghost, ui.SizeXS, twi.Key("copy-"+name), twi.Class("absolute top-0 right-1"), twi.OnClick(func(*twi.Event) { s.copy(p.source) }), twi.Text(label)),
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
	g, compiled := s.grammars[language]
	if build, known := map[string]func() *highlight.Grammar{"go": highlight.Go, "bash": highlight.Bash}[language]; known && !compiled {
		g = build()
		s.grammars[language] = g
	}
	if g != nil {
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

func (s *site) propsTable(of string) twi.Node {
	rows := [][][]markdown.Inline{{{{Text: "Name"}}, {{Text: "Type"}}, {{Text: "Description"}}}}
	for _, p := range s.props[of] {
		rows = append(rows, [][]markdown.Inline{{{Style: markdown.CodeSpan, Text: p.name}}, {{Style: markdown.CodeSpan, Text: p.kind}}, {{Text: p.about}}})
	}
	return markdown.Render(markdown.Page{Blocks: []markdown.Block{{Kind: markdown.Table, Rows: rows, Align: make([]markdown.Align, 3)}}}, markdown.Options{})
}
