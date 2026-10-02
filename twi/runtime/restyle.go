package runtime

import (
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/style"
)

func (r *Runtime) restyles(old, next []int, bit style.State) bool {
	from := 0
	if old != nil && next != nil {
		for from < min(len(old), len(next)) && old[from] == next[from] {
			from++
		}
		from++
	}
	sheet := r.cfg.Sheet.WithColumns(r.width)
	return stateful(sheet, &r.nodes, old, from, bit) || stateful(sheet, &r.nodes, next, from, bit)
}

type link struct {
	node  *render.Node
	marks style.Markers
	state style.NodeState
}

func stateful(sheet style.Sheet, root *render.Node, path []int, from int, bit style.State) bool {
	if path == nil {
		return false
	}
	chain := append(make([]link, 0, len(path)+1), link{node: root, state: stateOf(root)})
	for n := root; len(chain) <= len(path) && path[len(chain)-1] < len(n.Children); {
		i := path[len(chain)-1]
		state := placed(n.Children, i)
		n = &n.Children[i]
		chain = append(chain, link{node: n, state: state})
	}
	if from >= len(chain) {
		return false
	}
	for k := from; k < len(chain); k++ {
		chain[k].marks = sheet.Marks(chain[k].node.Classes)
	}
	var rules []int
	for k := from; k < len(chain); k++ {
		f := chain[k]
		own, with := f.state, f.state
		with.States |= bit
		if a, b := sheet.ComputeState(style.ComputedStyle{}, f.node.Classes, own), sheet.ComputeState(style.ComputedStyle{}, f.node.Classes, with); !a.Equal(&b) {
			return true
		}
		rules = sheet.Hands(f.node.Classes, everyState(f.state), rules[:0])
		for _, i := range rules {
			if rule := sheet.Rule(i); rule.When.States&bit != 0 && reaches(sheet, &rule.Target, f.node.Children) {
				return true
			}
		}
		for j, a := range chain[:k] {
			parent := j == k-1
			rules = sheet.Near(a.node.Classes, rules[:0])
			for _, i := range rules {
				if m := &sheet.Rule(i).Near; (m.Relation == style.RelationDescendant || m.Relation == style.RelationChild && parent) && touches(m, f, bit) {
					return true
				}
			}
			rules = sheet.Hands(a.node.Classes, everyState(a.state), rules[:0])
			for _, i := range rules {
				if m := &sheet.Rule(i).Target; (m.Relation == style.RelationDescendant || parent) && touches(m, f, bit) {
					return true
				}
			}
		}
		if k == 0 {
			continue
		}
		for _, sibling := range chain[k-1].node.Children[path[k-1]+1:] {
			rules = sheet.Near(sibling.Classes, rules[:0])
			for _, i := range rules {
				if m := &sheet.Rule(i).Near; m.Relation == style.RelationPrevious && touches(m, f, bit) {
					return true
				}
			}
		}
	}
	var walk func(n *render.Node, top int) bool
	walk = func(n *render.Node, top int) bool {
		for i := range n.Children {
			c := &n.Children[i]
			rules = sheet.Near(c.Classes, rules[:0])
			for _, j := range rules {
				m := &sheet.Rule(j).Near
				for a := from; a < top && m.Relation == style.RelationAncestor; a++ {
					if touches(m, chain[a], bit) {
						return true
					}
				}
			}
			next := top
			if top < len(chain) && chain[top].node == c {
				next++
			}
			if walk(c, next) {
				return true
			}
		}
		return false
	}
	return walk(chain[from].node, from+1)
}

func stateOf(n *render.Node) style.NodeState {
	if n.State == nil {
		return style.NodeState{}
	}
	return *n.State
}

func placed(siblings []render.Node, i int) style.NodeState {
	state := stateOf(&siblings[i])
	if bare(&siblings[i]) {
		return state
	}
	index, last := 0, -1
	for j := range siblings {
		if bare(&siblings[j]) {
			continue
		}
		if j < i {
			index++
		}
		last = j
	}
	count := index + 2
	if i == last {
		count = index + 1
	}
	state.Places = style.PlaceOf(index, count)
	return state
}

func bare(n *render.Node) bool {
	return n.Text != "" && n.Element == style.ElementAny && n.Classes == nil && n.State == nil && n.Children == nil
}

func everyState(state style.NodeState) style.NodeState {
	state.States = ^style.State(0)
	return state
}

func accepts(m *style.Match, l link) bool {
	state := l.state
	state.States |= m.States
	return m.Accepts(l.node.Element, l.marks, state)
}

func touches(m *style.Match, l link, bit style.State) bool {
	return m.States&bit != 0 && accepts(m, l)
}

func reaches(sheet style.Sheet, m *style.Match, children []render.Node) bool {
	for i := range children {
		c := &children[i]
		if accepts(m, link{c, sheet.Marks(c.Classes), placed(children, i)}) || m.Relation == style.RelationDescendant && reaches(sheet, m, c.Children) {
			return true
		}
	}
	return false
}
