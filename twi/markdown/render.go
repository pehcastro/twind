package markdown

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

type Options struct {
	Highlight func(language, code string) twi.Node
	Follow    func(target string)
}

func Render(p Page, o Options) twi.Node {
	return element("flex flex-col gap-1", o.blocks(p.Blocks, 0))
}

func element(classes string, children []twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(classes)}, children...)...)
}

func (o Options) blocks(blocks []Block, depth int) []twi.NodeOption {
	out := make([]twi.NodeOption, len(blocks))
	for i, b := range blocks {
		out[i] = o.block(b, depth)
	}
	return out
}

func (o Options) block(b Block, depth int) twi.Node {
	switch b.Kind {
	case Paragraph:
		return o.inline(b.Inlines, "")
	case Heading:
		return o.inline(b.Inlines, [...]string{"not-first:mt-1 font-bold border-b", "not-first:mt-1 font-bold", "font-semibold", "font-semibold text-muted-foreground", "font-semibold text-muted-foreground", "font-semibold text-muted-foreground"}[b.Level-1], twi.Data("anchor", b.ID))
	case List:
		width := len(strconv.Itoa(b.Start + len(b.Items) - 1))
		items := make([]twi.NodeOption, len(b.Items))
		bullet := [...]string{"•", "◦", "▪"}[depth%3]
		for i, it := range b.Items {
			mark := bullet
			if b.Ordered {
				mark = fmt.Sprintf("%*d.", width, b.Start+i)
			}
			items[i] = twi.Element(twi.Class("flex flex-row gap-1"), twi.Element(twi.Class("shrink-0 whitespace-pre text-muted-foreground"), twi.Text(mark)), element("flex flex-col gap-1 flex-1 min-w-0", o.blocks(it, depth+1)))
		}
		return element("flex flex-col gap-1", items)
	case Quote:
		return element("flex flex-col gap-1 border-l-2 pl-1 text-muted-foreground", o.blocks(b.Children, depth))
	case Code:
		body := twi.Text(strings.TrimSuffix(b.Code, "\n"))
		if o.Highlight != nil {
			body = o.Highlight(b.Language, b.Code)
		}
		code := []twi.NodeOption{twi.Element(twi.Class("px-1 whitespace-pre overflow-x-auto"), body)}
		if b.Language != "" {
			code = append([]twi.NodeOption{twi.Element(twi.Class("border-b px-1 text-muted-foreground"), twi.Text(b.Language))}, code...)
		}
		return element("flex flex-col rounded-md border bg-muted", code)
	case Table:
		rows := make([]twi.NodeOption, len(b.Rows))
		for i, row := range b.Rows {
			cells := make([]twi.NodeOption, len(row))
			for j, c := range row {
				align := twi.Class([...]string{AlignNone: "", AlignLeft: "text-left", AlignCenter: "text-center", AlignRight: "text-right"}[b.Align[j]])
				cell := ui.TableCell
				if i == 0 {
					cell = ui.TableHead
				}
				cells[j] = cell(align, o.inline(c, ""))
			}
			if i == len(b.Rows)-1 {
				cells = append(cells, twi.Class("border-b-0"))
			}
			rows[i] = ui.TableRow(cells...)
		}
		return twi.Element(twi.Class("rounded-md border"), ui.Table(ui.TableHeader(rows[0]), ui.TableBody(rows[1:]...)))
	case Break:
		return ui.Separator(ui.Horizontal)
	case Tag:
		return b.build(b.Attrs)
	}
	panic("markdown: unknown block kind " + strconv.Itoa(int(b.Kind)))
}

type segment struct{ text, classes, target string }

type flow struct {
	lines  [][][]segment
	joined bool
}

func (f *flow) walk(inlines []Inline, classes, target string) {
	for _, in := range inlines {
		switch in.Style {
		case Text:
			for i, word := range strings.Split(in.Text, " ") {
				if i > 0 {
					f.joined = false
				}
				if word != "" {
					f.add(segment{word, classes, target})
				}
			}
		case Emphasis:
			f.walk(in.Children, classes+" italic", target)
		case Strong:
			f.walk(in.Children, classes+" font-bold", target)
		case Link:
			f.walk(in.Children, classes+" text-primary underline", in.Target)
		case CodeSpan:
			f.add(segment{in.Text, classes + " bg-muted whitespace-pre", target})
		case SoftBreak:
			f.joined = false
		case HardBreak:
			f.lines, f.joined = append(f.lines, nil), false
		default:
			panic("markdown: unknown inline style " + strconv.Itoa(int(in.Style)))
		}
	}
}

func (f *flow) add(s segment) {
	line := &f.lines[len(f.lines)-1]
	if f.joined && len(*line) > 0 {
		(*line)[len(*line)-1] = append((*line)[len(*line)-1], s)
	} else {
		*line = append(*line, []segment{s})
	}
	f.joined = true
}

func (o Options) inline(inlines []Inline, classes string, extra ...twi.NodeOption) twi.Node {
	var plain strings.Builder
	styled := false
	for _, in := range inlines {
		switch in.Style {
		case Text:
			plain.WriteString(in.Text)
		case SoftBreak:
			plain.WriteByte(' ')
		default:
			styled = true
		}
	}
	if !styled {
		return twi.Element(append(extra, twi.Class(classes), twi.Text(plain.String()))...)
	}
	f := flow{lines: [][][]segment{nil}}
	f.walk(inlines, "", "")
	rows := make([]twi.NodeOption, len(f.lines))
	for i, line := range f.lines {
		words := make([]twi.NodeOption, len(line))
		for j, word := range line {
			segments := make([]twi.NodeOption, len(word))
			for k, s := range word {
				segments[k] = o.segment(s)
			}
			words[j] = segments[0]
			if len(segments) > 1 {
				words[j] = element("flex flex-row", segments)
			}
		}
		rows[i] = element("flex flex-row flex-wrap gap-x-1", words)
	}
	return twi.Element(append(append(extra, twi.Class("flex flex-col "+classes)), rows...)...)
}

func (o Options) segment(s segment) twi.Node {
	switch {
	case s.target != "" && o.Follow != nil:
		return twi.Element(twi.Class(s.classes), twi.Text(s.text), twi.OnClick(func(*twi.Event) { o.Follow(s.target) }))
	case s.classes != "":
		return twi.Element(twi.Class(s.classes), twi.Text(s.text))
	}
	return twi.Text(s.text)
}
