# twi/markdown

Every exported name in `twi/markdown`, one line each, grouped by the file that declares it. A test reads them from the Go source and fails when this page drifts from it.

## highlight.go

type Highlighter struct\
func (\*Highlighter) Code(language, src string) twi.Node

## markdown.go

type Kind uint8\
const Paragraph Kind\
const Heading Kind\
const List Kind\
const Quote Kind\
const Code Kind\
const Table Kind\
const Break Kind\
const Tag Kind\
type Style uint8\
const Text Style\
const Emphasis Style\
const Strong Style\
const CodeSpan Style\
const Link Style\
const SoftBreak Style\
const HardBreak Style\
type Align uint8\
const AlignNone Align\
const AlignLeft Align\
const AlignCenter Align\
const AlignRight Align\
type Inline struct\
Inline.Style Style\
Inline.Text string\
Inline.Target string\
Inline.Children \[\]Inline\
type Block struct\
Block.Kind Kind\
Block.Line int\
Block.Level int\
Block.ID string\
Block.Inlines \[\]Inline\
Block.Ordered bool\
Block.Start int\
Block.Tight bool\
Block.Items \[\]\[\]Block\
Block.Children \[\]Block\
Block.Language string\
Block.Code string\
Block.Align \[\]Align\
Block.Rows \[\]\[\]\[\]Inline\
Block.Name string\
Block.Attrs map\[string\]string\
type Anchor struct\
Anchor.Level int\
Anchor.Text string\
Anchor.ID string\
type Page struct\
func Parse(file, src string, components map\[string\]Component) (Page, error)\
Page.File string\
Page.Blocks \[\]Block\
Page.Anchors \[\]Anchor\
type Component struct\
Component.Attrs \[\]string\
Component.Build func(attrs map\[string\]string) twi.Node\
type Problem uint8\
const UnknownTag Problem\
const UnknownAttribute Problem\
const DuplicateAttribute Problem\
const MalformedTag Problem\
type Error struct\
Error.File string\
Error.Line int\
Error.Problem Problem\
Error.Name string\
func (\*Error) Error() string

## Render

type Options struct\
Options.Highlight func(language, code string) twi.Node\
Options.Follow func(target string)\
Options.Copy func(code string)\
Options.Copied string\
func Render(p Page, o Options) twi.Node
