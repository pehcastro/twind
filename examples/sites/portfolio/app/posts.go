package app

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/markdown"
)

type post struct {
	file, title, summary string
	day                  time.Time
	minutes              int
	tags                 []string
	body                 markdown.Page
}

func load(files fs.FS) ([]post, error) {
	names, err := fs.Glob(files, "posts/*.md")
	if err != nil {
		return nil, err
	}
	meta := map[string]markdown.Component{"Meta": {Attrs: []string{"date", "tags"}, Build: func(map[string]string) twi.Node { return el("hidden") }}}
	var posts []post
	for _, name := range names {
		src, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, err
		}
		p := post{file: path.Base(name), minutes: max(1, len(strings.Fields(string(src)))/wordsPerMinute)}
		page, err := markdown.Parse(p.file, string(src), meta)
		if err != nil {
			return nil, err
		}
		b := page.Blocks
		if len(b) < 3 || b[0].Kind != markdown.Heading || b[0].Level != 1 || b[1].Kind != markdown.Tag || b[2].Kind != markdown.Paragraph {
			return nil, fmt.Errorf("%s: a post opens with a level 1 heading, a <Meta> line and a paragraph", p.file)
		}
		if p.day, err = time.Parse(dateLayout, b[1].Attrs["date"]); err != nil {
			return nil, fmt.Errorf("%s:%d: date: %w", p.file, b[1].Line, err)
		}
		p.title, p.summary, p.tags = plain(b[0].Inlines), plain(b[2].Inlines), strings.Split(strings.ReplaceAll(b[1].Attrs["tags"], " ", ""), ",")
		p.body = markdown.Page{File: p.file, Blocks: b[3:]}
		posts = append(posts, p)
	}
	slices.Reverse(posts)
	return posts, nil
}

func plain(inlines []markdown.Inline) string {
	var b strings.Builder
	for _, in := range inlines {
		switch in.Style {
		case markdown.SoftBreak, markdown.HardBreak:
			b.WriteByte(' ')
		default:
			b.WriteString(in.Text + plain(in.Children))
		}
	}
	return b.String()
}
