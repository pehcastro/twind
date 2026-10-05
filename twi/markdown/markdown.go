package markdown

import (
	"fmt"

	"github.com/pehcastro/twind/twi"
)

type Kind uint8

const (
	Paragraph Kind = iota + 1
	Heading
	List
	Quote
	Code
	Table
	Break
	Tag
)

type Style uint8

const (
	Text Style = iota
	Emphasis
	Strong
	CodeSpan
	Link
	SoftBreak
	HardBreak
)

type Align uint8

const (
	AlignNone Align = iota
	AlignLeft
	AlignCenter
	AlignRight
)

type Inline struct {
	Style    Style
	Text     string
	Target   string
	Children []Inline
}

type Block struct {
	Kind     Kind
	Line     int
	Level    int
	ID       string
	Inlines  []Inline
	Ordered  bool
	Start    int
	Tight    bool
	Items    [][]Block
	Children []Block
	Language string
	Code     string
	Align    []Align
	Rows     [][][]Inline
	Name     string
	Attrs    map[string]string
	build    func(map[string]string) twi.Node
}

type Anchor struct {
	Level int
	Text  string
	ID    string
}

type Page struct {
	File    string
	Blocks  []Block
	Anchors []Anchor
}

type Component struct {
	Attrs []string
	Build func(attrs map[string]string) twi.Node
}

type Problem uint8

const (
	UnknownTag Problem = iota + 1
	UnknownAttribute
	DuplicateAttribute
	MalformedTag
)

type Error struct {
	File    string
	Line    int
	Problem Problem
	Name    string
}

func (e *Error) Error() string {
	what, ok := map[Problem]string{
		UnknownTag:         "unknown component <%s>",
		UnknownAttribute:   "unknown attribute %s",
		DuplicateAttribute: "attribute %s given twice",
		MalformedTag:       "component <%s> is not a self-closing tag with quoted attributes",
	}[e.Problem]
	if !ok {
		panic(fmt.Sprintf("markdown: unknown problem %d", e.Problem))
	}
	return fmt.Sprintf("%s:%d: "+what, e.File, e.Line, e.Name)
}
