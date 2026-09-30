package twi

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"slices"
	"strings"

	termkonst "github.com/twind-dev/twind/internal/konst/terminal"
	konst "github.com/twind-dev/twind/internal/konst/twi"
	"github.com/twind-dev/twind/internal/present"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/runtime"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/theme"
)

type Node struct{ built *node }

type node struct {
	tree     render.Node
	handlers *handlers
}

type handlers struct {
	keys    []func(input.KeyEvent)
	ownKeys int
	events  runtime.Node
	behaves bool
}

type NodeOption interface{ apply(*node) }

func (n Node) node() *node {
	if n.built == nil {
		return &node{}
	}
	return n.built
}

func (n Node) runtimeTree() runtime.Tree {
	built := n.node()
	tree := runtime.Tree{Root: built.tree}
	if h := built.handlers; h != nil {
		tree.Keys, tree.Events = h.keys, h.events
	}
	return tree
}

func (n Node) apply(parent *node) {
	built := n.node()
	at := len(parent.tree.Children)
	parent.tree.Children = append(parent.tree.Children, built.tree)
	child := built.handlers
	if child == nil {
		return
	}
	own := parent.withHandlers()
	own.keys = append(own.keys, child.keys...)
	if child.behaves {
		events := child.events
		events.At = []int{at}
		own.events.Children = append(own.events.Children, events)
		return
	}
	for _, c := range child.events.Children {
		c.At = append([]int{at}, c.At...)
		own.events.Children = append(own.events.Children, c)
	}
}

func (n *node) withHandlers() *handlers {
	if n.handlers == nil {
		n.handlers = &handlers{}
	}
	return n.handlers
}

type onKey func(input.KeyEvent)

func (h onKey) apply(n *node) {
	own := n.withHandlers()
	own.keys = slices.Insert(own.keys, own.ownKeys, (func(input.KeyEvent))(h))
	own.ownKeys++
}

func OnKey(handler func(input.KeyEvent)) NodeOption { return onKey(handler) }

type classList struct {
	names  []string
	inline [inlineClasses]string
}

const inlineClasses = 8

func (c *classList) apply(n *node) {
	if n.tree.Classes == nil && len(c.names) > 0 {
		n.tree.Classes = c.names
		return
	}
	n.tree.Classes = append(n.tree.Classes, c.names...)
}

func (n *node) state() *style.NodeState {
	if n.tree.State == nil {
		n.tree.State = &style.NodeState{}
	}
	return n.tree.State
}

type attribute style.Attr

func (a attribute) apply(n *node) {
	state := n.state()
	state.Attrs = append(state.Attrs, style.Attr(a))
}

type tag style.Element

func (t tag) apply(n *node) { n.tree.Element = style.Element(t) }

func Tag(element style.Element) NodeOption { return tag(element) }

type at image.Point

func (p at) apply(n *node) {
	cell := image.Point(p)
	n.tree.At = &cell
}

func At(x, y int) NodeOption { return at{X: x, Y: y} }

func Data(name, value string) NodeOption { return attribute{Name: "data-" + name, Value: value} }

func Element(options ...NodeOption) Node {
	children := 0
	for _, o := range options {
		if _, ok := o.(Node); ok {
			children++
		}
	}
	n := &node{}
	if children > 0 {
		n.tree.Children = make([]render.Node, 0, children)
	}
	for _, o := range options {
		o.apply(n)
	}
	return Node{n}
}

func Text(s string) Node { return Node{&node{tree: render.Node{Text: s}}} }

func Class(classes ...string) NodeOption {
	list := &classList{}
	list.names = list.inline[:0]
	for _, c := range classes {
		for name := range strings.FieldsSeq(c) {
			list.names = append(list.names, name)
		}
	}
	list.names = slices.Clip(list.names)
	return list
}

func Classes(options []NodeOption) (classes []string, rest []NodeOption) {
	for _, o := range options {
		if list, ok := o.(*classList); ok {
			classes = append(classes, list.names...)
		} else {
			rest = append(rest, o)
		}
	}
	return classes, rest
}

type RenderOption func(*renderConfig)

type renderConfig struct {
	width       int
	profile     color.Profile
	profileSet  bool
	sheet       style.Sheet
	theme       *theme.Theme
	fullscreen  bool
	noClipboard bool
	graphics    *terminal.Graphics
	backend     runtime.Backend
	clock       runtime.Clock
}

func look(p color.Profile) paint.Look {
	if p <= color.Attributes {
		return paint.Plain
	}
	return paint.Composited
}

func Width(cells int) RenderOption { return func(c *renderConfig) { c.width = cells } }

func ColorProfile(p color.Profile) RenderOption {
	return func(c *renderConfig) { c.profile, c.profileSet = p, true }
}

func Styles(sheet style.Sheet) RenderOption { return func(c *renderConfig) { c.sheet = sheet } }

func Theme(t theme.Theme) RenderOption { return func(c *renderConfig) { c.theme = &t } }

func RenderString(node Node, opts ...RenderOption) string {
	var out strings.Builder
	if err := Render(&out, node, opts...); err != nil {
		panic(err)
	}
	return out.String()
}

func Render(w io.Writer, node Node, opts ...RenderOption) (err error) {
	var cfg renderConfig
	for _, o := range opts {
		o(&cfg)
	}
	termWidth, termHeight, sizeErr := terminal.Size(w)
	if cfg.width <= 0 {
		cfg.width = konst.DefaultWidth
		if sizeErr == nil {
			cfg.width = termWidth
		}
	}
	if !cfg.profileSet {
		cfg.profile = terminal.Profile(w, os.Getenv)
	}
	if cfg.theme != nil {
		cfg.sheet = cfg.sheet.WithTheme(cfg.theme)
	}
	if os.Getenv("TWIND_GRAPHICS") != "" {
		cfg.graphics = nil
	}
	f, ok := w.(*os.File)
	if ok && sizeErr == nil && cfg.profile != color.None {
		restore, vtErr := terminal.EnableVirtualTerminal(f)
		if vtErr != nil {
			return vtErr
		}
		defer func() { err = errors.Join(err, restore()) }()
	}
	var caps terminal.Capabilities
	if ok && sizeErr == nil && cfg.profile >= color.ANSI256 && cfg.width <= termWidth && (cfg.graphics == nil || *cfg.graphics != terminal.GraphicsNone) {
		var cursor image.Point
		if caps, cursor, err = terminal.Query(os.Stdin, f); err != nil {
			return err
		}
		if pixels, err := inline(f, node, cfg, caps, cursor, termHeight); pixels || err != nil {
			return err
		}
	}
	buf, err := render.Render(node.node().tree, render.Frame{Sheet: cfg.sheet, Width: cfg.width, Look: look(cfg.profile), Profile: cfg.profile, Cell: caps.CellPixels, Widths: caps.Widths, ReducedMotion: true})
	if err != nil {
		return err
	}
	return (&terminal.Writer{Out: w, Profile: cfg.profile}).Static(buf)
}

func inline(out io.Writer, node Node, cfg renderConfig, caps terminal.Capabilities, cursor image.Point, screenRows int) (bool, error) {
	if cfg.graphics != nil {
		caps.Graphics = *cfg.graphics
	}
	if caps.Graphics == terminal.GraphicsNone || caps.CellPixels.X <= 0 || caps.CellPixels.Y <= 0 {
		return false, nil
	}
	root, err := render.Scene(node.node().tree, render.Frame{Sheet: cfg.sheet, Width: cfg.width, Cell: caps.CellPixels, Widths: caps.Widths, ReducedMotion: true})
	rows := root.Bounds.H
	if err != nil || rows >= screenRows {
		return false, err
	}
	lead := rows
	if cursor.X > 0 {
		lead++
	}
	top := min(cursor.Y+lead, screenRows-1) - rows
	var buf bytes.Buffer
	if caps.Sync {
		buf.WriteString(termkonst.SyncBegin)
	}
	buf.WriteString(strings.Repeat("\n", lead))
	fmt.Fprintf(&buf, "%s%d;%dr%s", termkonst.CSI, top+1, top+rows, termkonst.OriginOn)
	screen := present.Screen{Out: &buf, Profile: cfg.profile, Graphics: caps.Graphics, Cell: caps.CellPixels, Widths: caps.Widths, Workers: 1}
	if err := screen.Frame(root, cfg.width, rows); err != nil {
		return false, err
	}
	fmt.Fprintf(&buf, "%s%s%s%d;1H", termkonst.OriginOff, termkonst.RegionReset, termkonst.CSI, top+rows+1)
	if caps.Sync {
		buf.WriteString(termkonst.SyncEnd)
	}
	_, err = out.Write(buf.Bytes())
	return true, err
}
