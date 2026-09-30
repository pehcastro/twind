package runtime

import (
	"reflect"

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
}

func stateful(sheet style.Sheet, root *render.Node, path []int, from int, bit style.State) bool {
	if path == nil {
		return false
	}
	chain := []link{{root, sheet.Marks(root.Classes)}}
	for n := root; len(chain) <= len(path) && path[len(chain)-1] < len(n.Children); {
		n = &n.Children[path[len(chain)-1]]
		chain = append(chain, link{n, sheet.Marks(n.Classes)})
	}
	if from >= len(chain) {
		return false
	}
	var rules []int
	for k := from; k < len(chain); k++ {
		f := chain[k]
		own, with := stateOf(f.node), stateOf(f.node)
		with.States |= bit
		if !reflect.DeepEqual(sheet.ComputeState(style.ComputedStyle{}, f.node.Classes, own), sheet.ComputeState(style.ComputedStyle{}, f.node.Classes, with)) {
			return true
		}
		rules = sheet.Hands(f.node.Classes, everyState(f.node), rules[:0])
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
			rules = sheet.Hands(a.node.Classes, everyState(a.node), rules[:0])
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

func everyState(n *render.Node) style.NodeState {
	state := stateOf(n)
	state.States = ^style.State(0)
	return state
}

func accepts(m *style.Match, n *render.Node, marks style.Markers) bool {
	state := stateOf(n)
	state.States |= m.States
	return m.Accepts(n.Element, marks, state)
}

func touches(m *style.Match, l link, bit style.State) bool {
	return m.States&bit != 0 && accepts(m, l.node, l.marks)
}

func reaches(sheet style.Sheet, m *style.Match, children []render.Node) bool {
	for i := range children {
		c := &children[i]
		if accepts(m, c, sheet.Marks(c.Classes)) || m.Relation == style.RelationDescendant && reaches(sheet, m, c.Children) {
			return true
		}
	}
	return false
}
