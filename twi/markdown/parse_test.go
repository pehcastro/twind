package markdown

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/twind-dev/twind/twi"
)

type example struct {
	name, section, markdown, html string
}

func examples(t *testing.T) []example {
	t.Helper()
	raw, err := os.ReadFile("testdata/commonmark.txt")
	if err != nil {
		t.Fatal(err)
	}
	fence := strings.Repeat("`", 32)
	var out []example
	for chunk := range strings.SplitSeq(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n\n# ") {
		head, body, _ := strings.Cut(strings.TrimPrefix(chunk, "# "), "\n")
		fields := strings.SplitN(head, " ", 3)
		body = strings.TrimPrefix(body, fence+" example\n")
		body = body[:strings.LastIndex(body, fence)]
		md, html, ok := strings.Cut(body, "\n.\n")
		if !ok {
			md, html = "", strings.TrimPrefix(body, ".\n")
		} else {
			md += "\n"
		}
		unmark := strings.NewReplacer("→", "\t", "␣", " ")
		out = append(out, example{fields[0] + " " + fields[1], fields[2], unmark.Replace(md), unmark.Replace(html)})
	}
	return out
}

func TestParseCommonMark(t *testing.T) {
	all := examples(t)
	if len(all) < 190 {
		t.Fatalf("read %d examples from the fixture, want 195", len(all))
	}
	passed := map[string]int{}
	total := map[string]int{}
	for _, e := range all {
		total[e.section]++
		page, err := Parse("spec.md", e.markdown, nil)
		if err != nil {
			t.Errorf("%s (%s): %v", e.name, e.section, err)
			continue
		}
		if got := html(page.Blocks); got != e.html {
			t.Errorf("%s (%s)\nmarkdown %q\ngot  %q\nwant %q", e.name, e.section, e.markdown, got, e.html)
			continue
		}
		passed[e.section]++
	}
	for section, n := range total {
		t.Logf("%-30s %3d of %3d", section, passed[section], n)
	}
}

func TestParseAnchors(t *testing.T) {
	page, err := Parse("a.md", "# Getting started\n## Install\ntext\n## Install\n### What's `new`?\n> # Quoted *heading*\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Anchor{
		{1, "Getting started", "getting-started"},
		{2, "Install", "install"},
		{2, "Install", "install-1"},
		{3, "What's new?", "whats-new"},
		{1, "Quoted heading", "quoted-heading"},
	}
	if !reflect.DeepEqual(page.Anchors, want) {
		t.Errorf("anchors %+v\nwant %+v", page.Anchors, want)
	}
	if len(page.Blocks) != 6 || page.Blocks[3].ID != "install-1" || page.Blocks[4].ID != "whats-new" {
		t.Errorf("blocks %+v, want the fourth id install-1 and the fifth whats-new", page.Blocks)
	}
}

func callout() map[string]Component {
	return map[string]Component{"Callout": {Attrs: []string{"title", "tone"}, Build: func(a map[string]string) twi.Node {
		return twi.Element(twi.Class("flex flex-col rounded-md border px-1"), twi.Element(twi.Class("font-bold"), twi.Text(a["title"])), twi.Text("tone: "+a["tone"]))
	}}}
}

func TestComponentParse(t *testing.T) {
	page, err := Parse("page.md", "# Doc\n\nText.\n\n<Callout title=\"Heads up\" tone=\"info\" />\n\n- item\n\n  <Callout title=\"nested\"/>\n", callout())
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Blocks) != 4 {
		t.Fatalf("%d blocks, want heading, paragraph, tag, list: %s", len(page.Blocks), html(page.Blocks))
	}
	tag := page.Blocks[2]
	if tag.Kind != Tag || tag.Line != 5 || tag.Name != "Callout" || !reflect.DeepEqual(tag.Attrs, map[string]string{"title": "Heads up", "tone": "info"}) {
		t.Errorf("tag block %+v", tag)
	}
	if nested := page.Blocks[3].Items[0][1]; nested.Kind != Tag || nested.Line != 9 || nested.Attrs["title"] != "nested" {
		t.Errorf("tag in a list item %+v", nested)
	}
}

func TestComponentErrors(t *testing.T) {
	for _, c := range []struct {
		name, src string
		line      int
		problem   Problem
		what      string
	}{
		{"unknown tag", "a\n\n<Nope />\n", 3, UnknownTag, "Nope"},
		{"unknown attribute", "# T\n<Callout title=\"x\" colour=\"red\" />\n", 2, UnknownAttribute, "colour"},
		{"duplicate attribute", "<Callout title=\"a\" title=\"b\" />\n", 1, DuplicateAttribute, "title"},
		{"unquoted value", "\n\n\n<Callout title=x />\n", 4, MalformedTag, "Callout"},
		{"not self-closing", "<Callout title=\"x\">\n", 1, MalformedTag, "Callout"},
		{"text after the tag", "<Callout /> and more\n", 1, MalformedTag, "Callout"},
		{"unknown tag in a list item", "- item\n\n  <Nope />\n", 3, UnknownTag, "Nope"},
		{"unknown tag in a quote", "> quote\n>\n> <Nope a=\"b\" />\n", 3, UnknownTag, "Nope"},
		{"line count with CRLF", "a\r\n\r\n```\r\nx\r\n```\r\n<Nope />\r\n", 6, UnknownTag, "Nope"},
	} {
		_, err := Parse("page.md", c.src, callout())
		var e *Error
		if !errors.As(err, &e) {
			t.Errorf("%s: error %v, want *Error", c.name, err)
			continue
		}
		if e.File != "page.md" || e.Line != c.line || e.Problem != c.problem || e.Name != c.what {
			t.Errorf("%s: %+v, want line %d problem %d name %s", c.name, *e, c.line, c.problem, c.what)
		}
		if prefix := fmt.Sprintf("page.md:%d: ", c.line); !strings.HasPrefix(err.Error(), prefix) {
			t.Errorf("%s: message %q does not start with %q", c.name, err.Error(), prefix)
		}
	}
}

func TestComponentNotATag(t *testing.T) {
	page, err := Parse("page.md", "```\n<Nope />\n```\n\n<div>hi</div>\n\ntext <Nope /> inline\n", callout())
	if err != nil {
		t.Fatal(err)
	}
	if got := html(page.Blocks); got != "<pre><code>&lt;Nope /&gt;\n</code></pre>\n<p>&lt;div&gt;hi&lt;/div&gt;</p>\n<p>text &lt;Nope /&gt; inline</p>\n" {
		t.Errorf("html %q", got)
	}
}

var hostile = []string{
	"\x1b[2J", "\x1b[H", "\x1b]52;c;aGVsbG8=\x07", "\x1b]8;;https://evil.example\x1b\\fake\x1b]8;;\x1b\\",
	"\x1b]0;title\x07", "\x1b[6n", "\x1bP+q544e\x1b\\", "\u009b31m", "\u009d52;c;eA==\u009c", "\xe2\x80\xaeevil", "\x07\x08\x7f",
}

func hostilePage() string {
	h := strings.Join(hostile, "x")
	return "# Head " + h + "\n\npara *em " + h + "* `code " + h + "` [link " + h + "](https://ok.example/" + h + ")\n\n```go" + h + "\nfunc " + h + "\n```\n\n| a " + h + " | b |\n| - | - |\n| c | " + h + " |\n\n<Callout title=\"" + strings.ReplaceAll(h, "\"", "") + "\" />\n"
}

func clean(t *testing.T, where, s string) {
	t.Helper()
	for i, r := range s {
		if r == utf8.RuneError || (r < ' ' && r != '\n' && r != '\t') || r == 0x7f || (r >= 0x80 && r <= 0x9f) || (r >= 0x202a && r <= 0x202e) || (r >= 0x2066 && r <= 0x2069) {
			t.Errorf("%s: %U at byte %d of %q", where, r, i, s)
			return
		}
	}
}

func walk(t *testing.T, where string, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		clean(t, where, v.String())
	case reflect.Slice:
		for i := range v.Len() {
			walk(t, fmt.Sprintf("%s[%d]", where, i), v.Index(i))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			walk(t, where+" key", k)
			walk(t, where+"."+k.String(), v.MapIndex(k))
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				walk(t, where+"."+v.Type().Field(i).Name, v.Field(i))
			}
		}
	}
}

func TestParseHostile(t *testing.T) {
	page, err := Parse("evil.md", hostilePage(), callout())
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Blocks) != 5 {
		t.Fatalf("%d blocks, want heading, paragraph, code, table, tag: %s", len(page.Blocks), html(page.Blocks))
	}
	walk(t, "page", reflect.ValueOf(page))
	if got := page.Blocks[1].Inlines[len(page.Blocks[1].Inlines)-1].Target; !strings.HasPrefix(got, "https://ok.example/") {
		t.Errorf("link target %q", got)
	}
}

func html(blocks []Block) string {
	var b strings.Builder
	for _, blk := range blocks {
		writeBlock(&b, blk, false)
	}
	return b.String()
}

func cr(b *strings.Builder) {
	if s := b.String(); s != "" && !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
}

var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func writeBlock(b *strings.Builder, blk Block, tight bool) {
	switch blk.Kind {
	case Paragraph:
		if tight {
			writeInlines(b, blk.Inlines)
			return
		}
		b.WriteString("<p>")
		writeInlines(b, blk.Inlines)
		b.WriteString("</p>\n")
	case Heading:
		fmt.Fprintf(b, "<h%d>", blk.Level)
		writeInlines(b, blk.Inlines)
		fmt.Fprintf(b, "</h%d>\n", blk.Level)
	case Break:
		b.WriteString("<hr />\n")
	case Code:
		b.WriteString("<pre><code")
		if blk.Language != "" {
			b.WriteString(` class="language-` + escape.Replace(blk.Language) + `"`)
		}
		b.WriteString(">" + escape.Replace(blk.Code) + "</code></pre>\n")
	case Quote:
		b.WriteString("<blockquote>\n")
		for _, c := range blk.Children {
			writeBlock(b, c, false)
		}
		b.WriteString("</blockquote>\n")
	case List:
		tag := "ul"
		switch {
		case blk.Ordered && blk.Start != 1:
			tag = "ol"
			fmt.Fprintf(b, "<ol start=\"%d\">\n", blk.Start)
		case blk.Ordered:
			tag = "ol"
			b.WriteString("<ol>\n")
		default:
			b.WriteString("<ul>\n")
		}
		for _, item := range blk.Items {
			b.WriteString("<li>")
			for _, c := range item {
				if !blk.Tight || c.Kind != Paragraph {
					cr(b)
				}
				writeBlock(b, c, blk.Tight)
			}
			b.WriteString("</li>\n")
		}
		b.WriteString("</" + tag + ">\n")
	case Table:
		b.WriteString("<table>\n<thead>\n")
		for i, row := range blk.Rows {
			if i == 1 {
				b.WriteString("<tbody>\n")
			}
			b.WriteString("<tr>\n")
			for j, cell := range row {
				tag := map[bool]string{true: "th", false: "td"}[i == 0]
				b.WriteString("<" + tag + map[Align]string{AlignCenter: ` align="center"`, AlignRight: ` align="right"`, AlignLeft: ` align="left"`}[blk.Align[j]] + ">")
				writeInlines(b, cell)
				b.WriteString("</" + tag + ">\n")
			}
			b.WriteString("</tr>\n")
			if i == 0 {
				b.WriteString("</thead>\n")
			}
		}
		if len(blk.Rows) > 1 {
			b.WriteString("</tbody>\n")
		}
		b.WriteString("</table>\n")
	case Tag:
		b.WriteString("<" + blk.Name + " />\n")
	default:
		panic(fmt.Sprintf("html: kind %d", blk.Kind))
	}
}

func writeInlines(b *strings.Builder, inlines []Inline) {
	for _, in := range inlines {
		switch in.Style {
		case Text:
			b.WriteString(escape.Replace(in.Text))
		case Emphasis, Strong:
			tag := map[Style]string{Emphasis: "em", Strong: "strong"}[in.Style]
			b.WriteString("<" + tag + ">")
			writeInlines(b, in.Children)
			b.WriteString("</" + tag + ">")
		case CodeSpan:
			b.WriteString("<code>" + escape.Replace(in.Text) + "</code>")
		case Link:
			var href strings.Builder
			for i := range len(in.Target) {
				if c := in.Target[i]; c < 0x80 && (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte("-_.!~*'();/?:@&=+$,#%", c) >= 0) {
					href.WriteByte(c)
				} else {
					fmt.Fprintf(&href, "%%%02X", c)
				}
			}
			b.WriteString(`<a href="` + escape.Replace(href.String()) + `">`)
			writeInlines(b, in.Children)
			b.WriteString("</a>")
		case SoftBreak:
			b.WriteString("\n")
		case HardBreak:
			b.WriteString("<br />\n")
		default:
			panic(fmt.Sprintf("html: style %d", in.Style))
		}
	}
}
