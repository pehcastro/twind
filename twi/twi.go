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

type Node struct {
	tree    render.Node
	keys    []func(input.KeyEvent)
	ownKeys int
	events  runtime.Node
	behaves bool
}

type NodeOption interface{ apply(*Node) }

func (n Node) apply(parent *Node) {
	at := len(parent.tree.Children)
	parent.tree.Children = append(parent.tree.Children, n.tree)
	parent.keys = append(parent.keys, n.keys...)
	if n.behaves {
		n.events.At = []int{at}
		parent.events.Children = append(parent.events.Children, n.events)
		return
	}
	for _, c := range n.events.Children {
		c.At = append([]int{at}, c.At...)
		parent.events.Children = append(parent.events.Children, c)
	}
}

type onKey func(input.KeyEvent)

func (h onKey) apply(n *Node) {
	n.keys = slices.Insert(n.keys, n.ownKeys, (func(input.KeyEvent))(h))
	n.ownKeys++
}

func OnKey(handler func(input.KeyEvent)) NodeOption { return onKey(handler) }

type classList []string

func (c classList) apply(n *Node) { n.tree.Classes = append(n.tree.Classes, c...) }

func (n *Node) state() *style.NodeState {
	if n.tree.State == nil {
		n.tree.State = &style.NodeState{}
	}
	return n.tree.State
}

type attribute style.Attr

func (a attribute) apply(n *Node) {
	state := n.state()
	state.Attrs = append(state.Attrs, style.Attr(a))
}

type tag style.Element

func (t tag) apply(n *Node) { n.tree.Element = style.Element(t) }

func Tag(element style.Element) NodeOption { return tag(element) }

func Data(name, value string) NodeOption { return attribute{Name: "data-" + name, Value: value} }

func Element(options ...NodeOption) Node {
	var n Node
	for _, o := range options {
		o.apply(&n)
	}
	return n
}

func Text(s string) Node { return Node{tree: render.Node{Text: s}} }

func Class(classes ...string) NodeOption {
	var list classList
	for _, c := range classes {
		list = append(list, strings.Fields(c)...)
	}
	return list
}

func Classes(options []NodeOption) (classes []string, rest []NodeOption) {
	for _, o := range options {
		if list, ok := o.(classList); ok {
			classes = append(classes, list...)
		} else {
			rest = append(rest, o)
		}
	}
	return classes, rest
}

type RenderOption func(*renderConfig)

type renderConfig struct {
	width      int
	profile    color.Profile
	profileSet bool
	sheet      style.Sheet
	theme      *theme.Theme
	fullscreen bool
	graphics   *terminal.Graphics
	backend    runtime.Backend
	clock      runtime.Clock
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
	if ok && sizeErr == nil && cfg.profile >= color.ANSI256 && cfg.width <= termWidth && (cfg.graphics == nil || *cfg.graphics != terminal.GraphicsNone) {
		caps, cursor, err := terminal.Query(os.Stdin, f)
		if err != nil {
			return err
		}
		if pixels, err := inline(f, node, cfg, caps, cursor, termHeight); pixels || err != nil {
			return err
		}
	}
	buf, err := render.Render(node.tree, render.Frame{Sheet: cfg.sheet, Width: cfg.width, Look: look(cfg.profile)})
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
	root, err := render.Scene(node.tree, render.Frame{Sheet: cfg.sheet, Width: cfg.width})
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
	screen := present.Screen{Out: &buf, Profile: cfg.profile, Graphics: caps.Graphics, Cell: caps.CellPixels}
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
