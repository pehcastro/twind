package tailwind

import (
	"cmp"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/twind-dev/twind/twi/css"
	"github.com/twind-dev/twind/twi/style"
)

type Category uint8

const (
	Unsupported Category = iota
	Approximated
	Ignored
)

func (c Category) String() string {
	switch c {
	case Unsupported:
		return "unsupported"
	case Approximated:
		return "approximated"
	case Ignored:
		return "ignored"
	}
	panic("tailwind: unknown category " + strconv.Itoa(int(c)))
}

type Warning struct {
	Class    string
	Property string
	Category Category
	Reason   string
}

func (w Warning) String() string {
	return w.Class + ": " + w.Property + ": " + w.Category.String() + ": " + w.Reason
}

type problem struct {
	category Category
	reason   string
}

type ranked struct {
	rank, seq int
	rule      style.Rule
}

type tokens = map[string][]css.Token

type vars struct{ theme, local tokens }

type compiler struct {
	layers      map[string]int
	light, dark tokens
	initial     map[string]bool
	rules       []ranked
	warnings    []Warning
}

type scope struct {
	layer  string
	when   style.Condition
	reject string
}

const unlayered = ""

func Compile(src string) ([]style.Rule, []Warning, error) {
	nodes, err := css.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	c := compiler{layers: map[string]int{}, light: tokens{}, dark: tokens{}, initial: map[string]bool{}}
	c.declarations(nodes, c.light)
	if len(c.dark) == 0 {
		c.dark = nil
	} else {
		overrides := c.dark
		c.dark = maps.Clone(c.light)
		maps.Copy(c.dark, overrides)
	}
	c.walk(nodes, scope{layer: unlayered})
	slices.SortStableFunc(c.rules, func(a, b ranked) int {
		return cmp.Or(cmp.Compare(a.rank, b.rank), cmp.Compare(a.seq, b.seq))
	})
	rules := make([]style.Rule, len(c.rules))
	for i, r := range c.rules {
		rules[i] = r.rule
	}
	return rules, c.warnings, nil
}

func (c *compiler) rank(layer string) int {
	if layer == unlayered {
		return len(c.layers) + 1
	}
	if _, ok := c.layers[layer]; !ok {
		c.layers[layer] = len(c.layers)
	}
	return c.layers[layer]
}

func (c *compiler) declarations(nodes []css.Node, into tokens) {
	for _, n := range nodes {
		switch n := n.(type) {
		case css.AtRule:
			switch n.Name {
			case "layer":
				for _, t := range n.Prelude {
					if t.Kind == css.TokenIdent {
						c.rank(t.Text)
					}
				}
				if n.HasBlock && text(n.Prelude) == "theme" {
					c.declarations(n.Block, into)
				}
			case "media":
				if when, reject := media(style.Condition{}, text(n.Prelude)); reject == "" && when.Scheme == style.SchemeDark {
					c.declarations(n.Block, c.dark)
				}
			case "property":
				c.initial[text(n.Prelude)] = slices.ContainsFunc(n.Block, func(d css.Node) bool {
					decl, ok := d.(css.Declaration)
					return ok && decl.Property == "initial-value"
				})
			}
		case css.Rule:
			for _, d := range n.Block {
				if d, ok := d.(css.Declaration); ok && strings.HasPrefix(d.Property, "--") {
					into[d.Property] = d.Value
				}
			}
		}
	}
}

func siblings(nodes, out []css.Node) []css.Node {
	for _, n := range nodes {
		switch n := n.(type) {
		case css.AtRule:
			if n.Name == "supports" {
				out = siblings(n.Block, out)
				continue
			}
		case css.Rule:
			if last := len(out) - 1; last >= 0 {
				if prev, ok := out[last].(css.Rule); ok && prev.Selector == n.Selector {
					prev.Block = slices.Concat(prev.Block, n.Block)
					out[last] = prev
					continue
				}
			}
		}
		out = append(out, n)
	}
	return out
}

func (c *compiler) walk(nodes []css.Node, sc scope) {
	for _, n := range siblings(nodes, nil) {
		switch n := n.(type) {
		case css.AtRule:
			switch n.Name {
			case "layer":
				name := text(n.Prelude)
				if n.HasBlock && name != "theme" && name != "properties" {
					c.walk(n.Block, scope{layer: name, when: sc.when, reject: sc.reject})
				}
			case "media":
				if strings.ReplaceAll(text(n.Prelude), " ", "") == "not(hover:hover)" {
					continue
				}
				when, reject := media(sc.when, text(n.Prelude))
				c.walk(n.Block, scope{layer: sc.layer, when: when, reject: cmp.Or(sc.reject, reject)})
			}
		case css.Rule:
			c.rule(n, sc)
		}
	}
}

func (c *compiler) rule(r css.Rule, sc scope) {
	selectors := selectorList(r.Selector)
	if sc.layer == "base" {
		if slices.Contains(selectors, "*") {
			c.themed(sc.layer, style.Rule{}, r.Block, func(string, problem) {})
		}
		return
	}
	for _, sel := range selectors {
		out, reason := selector(sel, sc.when)
		if reason = cmp.Or(reason, sc.reject); reason != "" {
			c.warnings = append(c.warnings, Warning{Class: out.Class, Property: "selector", Category: Unsupported, Reason: reason})
			continue
		}
		c.themed(sc.layer, out, r.Block, func(property string, p problem) {
			c.warnings = append(c.warnings, Warning{Class: out.Class, Property: property, Category: p.category, Reason: p.reason})
		})
	}
}

func (c *compiler) themed(layer string, out style.Rule, block []css.Node, warn func(string, problem)) {
	theme := c.light
	if out.When.Scheme == style.SchemeDark && c.dark != nil {
		theme = c.dark
	}
	decls := c.block(block, vars{theme: theme}, warn)
	c.emit(layer, out, decls)
	if out.When.Scheme != style.SchemeAny || c.dark == nil {
		return
	}
	if dark := c.block(block, vars{theme: c.dark}, func(string, problem) {}); !reflect.DeepEqual(dark, decls) {
		out.When.Scheme = style.SchemeDark
		c.emit(layer, out, dark)
	}
}

func selectorList(list string) []string {
	var out []string
	start, depth := 0, 0
	for i := 0; i < len(list); i++ {
		switch list[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(list[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(list[start:]))
}

func (c *compiler) emit(layer string, r style.Rule, decls []style.Declaration) {
	if len(decls) == 0 {
		return
	}
	r.Decls = decls
	c.rules = append(c.rules, ranked{rank: c.rank(layer), seq: len(c.rules), rule: r})
}

func (c *compiler) block(nodes []css.Node, outer vars, warn func(string, problem)) []style.Declaration {
	v := vars{theme: outer.theme, local: tokens{}}
	maps.Copy(v.local, outer.local)
	var collect func([]css.Node)
	collect = func(nodes []css.Node) {
		for _, n := range nodes {
			switch n := n.(type) {
			case css.Declaration:
				if strings.HasPrefix(n.Property, "--") {
					v.local[n.Property] = n.Value
				}
			case css.AtRule:
				if n.Name == "supports" {
					collect(n.Block)
				}
			}
		}
	}
	collect(nodes)
	var out []style.Declaration
	for _, n := range nodes {
		switch n := n.(type) {
		case css.Declaration:
			prop := strings.ToLower(n.Property)
			if strings.HasPrefix(prop, "-") && !tracked(prop) {
				continue
			}
			decls, p := c.declaration(prop, n.Value, v)
			if p.reason != "" {
				warn(n.Property, p)
			}
			out = append(out, decls...)
		case css.AtRule:
			if n.Name != "supports" {
				warn("@"+n.Name, problem{Unsupported, "at-rule nested in a rule"})
				continue
			}
			out = append(out, c.block(n.Block, v, warn)...)
		case css.Rule:
			warn(n.Selector, problem{Unsupported, "nested rule"})
		}
	}
	return out
}

func tracked(prop string) bool {
	switch prop {
	case "--tw-gradient-from", "--tw-gradient-via", "--tw-gradient-to", "--tw-gradient-from-position", "--tw-gradient-via-position", "--tw-gradient-to-position",
		"--tw-shadow-color", "--tw-inset-shadow-color", "--tw-ring-color", "--tw-ring-inset", "--tw-ring-offset-width", "--tw-ring-offset-color",
		"--tw-duration", "--tw-ease", "--tw-animation-duration", "--tw-animation-delay", "--tw-animation-iteration-count", "--tw-animation-fill-mode":
		return true
	}
	return strings.HasPrefix(prop, "--tw-enter-") || strings.HasPrefix(prop, "--tw-exit-")
}

func text(toks []css.Token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.Text)
	}
	return strings.TrimSpace(b.String())
}
