package main

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/markdown"
	"github.com/twind-dev/twind/twi/ui"
)

type post struct {
	file, title, summary, date string
	tags                       []string
	page                       markdown.Page
}

func load(files fs.FS) ([]post, error) {
	names, err := fs.Glob(files, "posts/*.md")
	if err != nil {
		return nil, err
	}
	var posts []post
	for _, name := range names {
		src, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, err
		}
		p := post{file: path.Base(name)}
		minutes := max(1, len(strings.Fields(string(src)))/wordsPerMinute)
		meta := markdown.Component{Attrs: []string{"date", "tags"}, Build: func(map[string]string) twi.Node { return byline(p.date, minutes, p.tags) }}
		if p.page, err = markdown.Parse(p.file, string(src), map[string]markdown.Component{"Meta": meta}); err != nil {
			return nil, err
		}
		for _, b := range p.page.Blocks {
			switch {
			case b.Kind == markdown.Heading && b.Level == 1 && p.title == "":
				p.title = plain(b.Inlines)
			case b.Kind == markdown.Tag && p.date == "":
				day, err := time.Parse(dateLayout, b.Attrs["date"])
				if err != nil {
					return nil, fmt.Errorf("%s:%d: date: %w", p.file, b.Line, err)
				}
				p.date, p.tags = day.Format(shownDate), strings.Split(strings.ReplaceAll(b.Attrs["tags"], " ", ""), ",")
			case b.Kind == markdown.Paragraph && p.summary == "":
				p.summary = plain(b.Inlines)
			}
		}
		if p.title == "" || p.date == "" || p.summary == "" {
			return nil, fmt.Errorf("%s: a post needs a level 1 heading, a <Meta> line and a paragraph", p.file)
		}
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

func byline(date string, minutes int, tags []string) twi.Node {
	row := []twi.NodeOption{txt("", date), txt("", "·"), txt("", strconv.Itoa(minutes)+" min read")}
	for _, t := range tags {
		row = append(row, ui.Badge(ui.Secondary, twi.Text(t)))
	}
	return el("flex flex-row flex-wrap items-center gap-1 text-muted-foreground", row...)
}
