package tailwind

import (
	"cmp"
	"maps"
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

type compiler struct {
	layers     map[string]int
	theme      map[string][]css.Token
	registered map[string]bool
	rules      []ranked
	warnings   []Warning
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
	c := compiler{layers: map[string]int{}, theme: map[string][]css.Token{}, registered: map[string]bool{}}
	c.declarations(nodes)
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

func (c *compiler) declarations(nodes []css.Node) {
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
					c.declarations(n.Block)
				}
			case "property":
				c.registered[text(n.Prelude)] = true
			}
		case css.Rule:
			for _, d := range n.Block {
				if d, ok := d.(css.Declaration); ok && strings.HasPrefix(d.Property, "--") {
					c.theme[d.Property] = d.Value
				}
			}
		}
	}
}

func (c *compiler) walk(nodes []css.Node, sc scope) {
	for _, n := range nodes {
		switch n := n.(type) {
		case css.AtRule:
			switch n.Name {
			case "layer":
				name := text(n.Prelude)
				if n.HasBlock && name != "theme" && name != "properties" {
					c.walk(n.Block, scope{layer: name, when: sc.when, reject: sc.reject})
				}
			case "media":
				when, reject := media(sc.when, text(n.Prelude))
				c.walk(n.Block, scope{layer: sc.layer, when: when, reject: cmp.Or(sc.reject, reject)})
			case "supports":
				c.walk(n.Block, sc)
			}
		case css.Rule:
			c.rule(n, sc)
		}
	}
}

func (c *compiler) rule(r css.Rule, sc scope) {
	selectors := strings.Split(r.Selector, ",")
	if sc.layer == "base" {
		if slices.ContainsFunc(selectors, func(s string) bool { return strings.TrimSpace(s) == "*" }) {
			c.emit(sc.layer, style.Rule{}, c.block(r.Block, nil, func(string, problem) {}))
		}
		return
	}
	for _, sel := range selectors {
		out, reason := selector(strings.TrimSpace(sel), sc.when)
		if reason = cmp.Or(reason, sc.reject); reason != "" {
			c.warnings = append(c.warnings, Warning{Class: out.Class, Property: "selector", Category: Unsupported, Reason: reason})
			continue
		}
		warn := func(property string, p problem) {
			c.warnings = append(c.warnings, Warning{Class: out.Class, Property: property, Category: p.category, Reason: p.reason})
		}
		c.emit(sc.layer, out, c.block(r.Block, nil, warn))
	}
}

func (c *compiler) emit(layer string, r style.Rule, decls []style.Declaration) {
	if len(decls) == 0 {
		return
	}
	r.Decls = decls
	c.rules = append(c.rules, ranked{rank: c.rank(layer), seq: len(c.rules), rule: r})
}

func (c *compiler) block(nodes []css.Node, outer map[string][]css.Token, warn func(string, problem)) []style.Declaration {
	locals := map[string][]css.Token{}
	maps.Copy(locals, outer)
	for _, n := range nodes {
		if d, ok := n.(css.Declaration); ok && strings.HasPrefix(d.Property, "--") {
			locals[d.Property] = d.Value
		}
	}
	var out []style.Declaration
	for _, n := range nodes {
		switch n := n.(type) {
		case css.Declaration:
			if strings.HasPrefix(n.Property, "-") {
				continue
			}
			decls, p := c.declaration(strings.ToLower(n.Property), n.Value, locals)
			if p.reason != "" {
				warn(n.Property, p)
			}
			out = append(out, decls...)
		case css.AtRule:
			if n.Name != "supports" {
				warn("@"+n.Name, problem{Unsupported, "at-rule nested in a rule"})
				continue
			}
			out = append(out, c.block(n.Block, locals, warn)...)
		case css.Rule:
			warn(n.Selector, problem{Unsupported, "nested rule"})
		}
	}
	return out
}

func text(toks []css.Token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(t.Text)
	}
	return strings.TrimSpace(b.String())
}
