package twi

import (
	"errors"
	"io"
	"os"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/twi"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

type Node struct{ tree render.Node }

type NodeOption interface{ apply(*Node) }

func (n Node) apply(parent *Node) { parent.tree.Children = append(parent.tree.Children, n.tree) }

type classList []string

func (c classList) apply(n *Node) { n.tree.Classes = append(n.tree.Classes, c...) }

func Element(options ...NodeOption) Node {
	var n Node
	for _, o := range options {
		o.apply(&n)
	}
	return n
}

func Text(s string) Node { return Node{render.Node{Text: s}} }

func Class(classes ...string) NodeOption {
	var list classList
	for _, c := range classes {
		list = append(list, strings.Fields(c)...)
	}
	return list
}

type RenderOption func(*renderConfig)

type renderConfig struct {
	width      int
	profile    color.Profile
	profileSet bool
	sheet      style.Sheet
}

func Width(cells int) RenderOption { return func(c *renderConfig) { c.width = cells } }

func ColorProfile(p color.Profile) RenderOption {
	return func(c *renderConfig) { c.profile, c.profileSet = p, true }
}

func Styles(sheet style.Sheet) RenderOption { return func(c *renderConfig) { c.sheet = sheet } }

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
	termWidth, _, sizeErr := terminal.Size(w)
	if cfg.width <= 0 {
		cfg.width = konst.DefaultWidth
		if sizeErr == nil {
			cfg.width = termWidth
		}
	}
	if !cfg.profileSet {
		cfg.profile = terminal.Profile(w, os.Getenv)
	}
	buf, err := render.Render(node.tree, render.Frame{Sheet: cfg.sheet, Width: cfg.width})
	if err != nil {
		return err
	}
	if f, ok := w.(*os.File); ok && sizeErr == nil && cfg.profile != color.None {
		restore, vtErr := terminal.EnableVirtualTerminal(f)
		if vtErr != nil {
			return vtErr
		}
		defer func() { err = errors.Join(err, restore()) }()
	}
	return (&terminal.Writer{Out: w, Profile: cfg.profile}).Static(buf)
}
