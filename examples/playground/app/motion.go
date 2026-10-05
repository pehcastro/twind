package app

import (
	"math"
	"slices"
	"time"

	runkonst "github.com/pehcastro/twind/internal/konst/runtime"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/motion"
	"github.com/pehcastro/twind/twi/ui"
)

type sprung struct {
	id     motion.ID
	live   bool
	target float64
}

type moves struct {
	rt          *twi.Runtime
	line        motion.Timeline
	now         time.Duration
	ticking     bool
	open        bool
	menu        sprung
	items, rows []string
	dots        []string
	slots       []sprung
	order       []int
}

func newMoves(rt *twi.Runtime) *moves {
	m := &moves{rt: rt, items: []string{"Profile", "Billing", "Settings", "Log out"}, rows: []string{"Inbox", "Draft", "Notes", "Queue"}, dots: []string{"text-chart-1", "text-chart-2", "text-chart-3", "text-chart-4"}}
	for i := range m.rows {
		m.slots = append(m.slots, sprung{target: float64(i)})
		m.order = append(m.order, i)
	}
	return m
}

func (m *moves) to(s *sprung, target float64) {
	if s.live {
		m.line.Retarget(s.id, m.now, motion.Float(target))
	} else {
		s.id, s.live = m.line.Start(m.now, motion.Float(s.target), motion.Float(target), motion.SpringDefault()), true
	}
	s.target = target
	if !m.ticking {
		m.ticking = true
		m.rt.After(runkonst.MotionInterval, m.tick)
	}
}

func (m *moves) tick() {
	m.now += runkonst.MotionInterval
	done, running := m.line.Step(m.now)
	m.menu.live = m.menu.live && !slices.Contains(done, m.menu.id)
	for i, s := range m.slots {
		m.slots[i].live = s.live && !slices.Contains(done, s.id)
	}
	m.rt.Invalidate()
	if m.ticking = running; running {
		m.rt.After(runkonst.MotionInterval, m.tick)
	}
}

func (m *moves) cell(s sprung) int {
	if s.live {
		return int(math.Round(m.line.Value(s.id).Float()))
	}
	return int(s.target)
}

func (m *moves) toggle() {
	m.open = !m.open
	target := 0.0
	if m.open {
		target = float64(len(m.items))
	}
	m.to(&m.menu, target)
}

func (m *moves) reorder() {
	last := len(m.order) - 1
	m.order = slices.Insert(m.order[:last], 0, m.order[last])
	for slot, row := range m.order {
		m.to(&m.slots[row], float64(slot))
	}
}

func (c controls) springMenu() twi.Node {
	m := c.kit.moves
	label := map[bool]string{false: "Open menu", true: "Close menu"}[m.open]
	trigger := []twi.Node{c.uiButton("open menu", ui.ButtonOutline, label, func(*state) { m.toggle() })}
	if shown := min(m.cell(m.menu), len(m.items)); shown > 0 {
		menu := []twi.NodeOption{twi.Class("absolute top-1 right-0 z-50 w-20 flex flex-col p-1 rounded-md border bg-popover text-popover-foreground shadow-md")}
		for _, item := range m.items[:shown] {
			menu = append(menu, twi.Element(twi.Class("px-1 rounded-sm hover:bg-accent hover:text-accent-foreground"), twi.Text(item), c.clicked(m.toggle)))
		}
		trigger = append(trigger, twi.Element(menu...))
	}
	return el("relative", trigger...)
}

func (c controls) flipList() twi.Node {
	m := c.kit.moves
	var rows []twi.Node
	for _, row := range slices.Backward(m.order) {
		rows = append(rows, twi.Element(twi.Key(m.rows[row]), twi.At(0, m.cell(m.slots[row])), twi.Class("flex flex-row gap-1 px-1"), txt(m.dots[row], "●"), twi.Text(m.rows[row])))
	}
	return el("relative h-4 w-12", rows...)
}
