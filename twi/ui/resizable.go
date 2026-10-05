package ui

import (
	"slices"
	"strings"

	konst "github.com/pehcastro/twind/internal/konst/ui"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
)

const panelBases = `basis-[0%]
basis-[1%] basis-[2%] basis-[3%] basis-[4%] basis-[5%] basis-[6%] basis-[7%] basis-[8%] basis-[9%] basis-[10%]
basis-[11%] basis-[12%] basis-[13%] basis-[14%] basis-[15%] basis-[16%] basis-[17%] basis-[18%] basis-[19%] basis-[20%]
basis-[21%] basis-[22%] basis-[23%] basis-[24%] basis-[25%] basis-[26%] basis-[27%] basis-[28%] basis-[29%] basis-[30%]
basis-[31%] basis-[32%] basis-[33%] basis-[34%] basis-[35%] basis-[36%] basis-[37%] basis-[38%] basis-[39%] basis-[40%]
basis-[41%] basis-[42%] basis-[43%] basis-[44%] basis-[45%] basis-[46%] basis-[47%] basis-[48%] basis-[49%] basis-[50%]
basis-[51%] basis-[52%] basis-[53%] basis-[54%] basis-[55%] basis-[56%] basis-[57%] basis-[58%] basis-[59%] basis-[60%]
basis-[61%] basis-[62%] basis-[63%] basis-[64%] basis-[65%] basis-[66%] basis-[67%] basis-[68%] basis-[69%] basis-[70%]
basis-[71%] basis-[72%] basis-[73%] basis-[74%] basis-[75%] basis-[76%] basis-[77%] basis-[78%] basis-[79%] basis-[80%]
basis-[81%] basis-[82%] basis-[83%] basis-[84%] basis-[85%] basis-[86%] basis-[87%] basis-[88%] basis-[89%] basis-[90%]
basis-[91%] basis-[92%] basis-[93%] basis-[94%] basis-[95%] basis-[96%] basis-[97%] basis-[98%] basis-[99%] basis-[100%]`

type Resizable struct {
	control
	Orientation Orientation
	Sizes       []int
	OnResize    func([]int)
	panels      []*twi.Ref
	built       int
	dragging    bool
}

func NewResizable(rt *twi.Runtime) *Resizable { return &Resizable{control: control{rt: rt}} }

func (r *Resizable) Node(children ...twi.NodeOption) twi.Node {
	if n := r.built; n > 0 && len(r.Sizes) != n {
		r.Sizes = make([]int, n)
		for i := range r.Sizes {
			r.Sizes[i] = konst.PercentWhole / n
		}
		r.rt.Invalidate()
	}
	r.built = 0
	return part("flex h-full w-full "+pick("resizable", r.Orientation, map[Orientation]string{
		Horizontal: "flex-row",
		Vertical:   "flex-col",
	}), append([]twi.NodeOption{twi.Data("slot", "resizable-panel-group")}, children...))
}

func (r *Resizable) Panel(children ...twi.NodeOption) twi.Node {
	i := r.built
	r.built++
	if i == len(r.panels) {
		r.panels = append(r.panels, &twi.Ref{})
	}
	size := "flex-1"
	if i < len(r.Sizes)-1 {
		size = "shrink-0 grow-0 " + strings.Fields(panelBases)[r.Sizes[i]]
	}
	return part("relative flex flex-col min-w-0 min-h-0 overflow-hidden "+size, append([]twi.NodeOption{twi.Data("slot", "resizable-panel"), twi.Measure(r.panels[i])}, children...))
}

func (r *Resizable) Handle(withHandle bool, children ...twi.NodeOption) twi.Node {
	at := r.built - 1
	keys := r.behave(func(k input.KeyEvent) bool {
		less, more := input.KeyArrowLeft, input.KeyArrowRight
		if r.Orientation == Vertical {
			less, more = input.KeyArrowUp, input.KeyArrowDown
		}
		switch k.Key {
		case less:
			r.resize(at, r.Sizes[at]-konst.PanelStep)
		case more:
			r.resize(at, r.Sizes[at]+konst.PanelStep)
		case input.KeyHome:
			r.resize(at, 0)
		case input.KeyEnd:
			r.resize(at, konst.PercentWhole)
		default:
			return false
		}
		return true
	})
	if withHandle {
		children = append([]twi.NodeOption{part("absolute z-10 rounded-lg bg-border "+pick("resizable grip", r.Orientation, map[Orientation]string{
			Horizontal: "-left-1 top-1/2 h-2 w-2 -translate-y-1/2",
			Vertical:   "-top-2 left-1/2 h-2 w-4 -translate-x-1/2",
		}), nil)}, children...)
	}
	return part("relative flex shrink-0 self-stretch items-center justify-center border-border focus-visible:border-ring "+pick("resizable handle", r.Orientation, map[Orientation]string{
		Horizontal: "border-l",
		Vertical:   "border-t",
	}), slices.Concat(keys, []twi.NodeOption{
		twi.Data("slot", "resizable-handle"),
		twi.OnPointerDown(func(*twi.Event) { r.dragging = true }),
		twi.OnPointerMove(func(e *twi.Event) {
			if r.dragging {
				r.follow(at, e.Mouse)
			}
		}),
		twi.OnPointerUp(func(*twi.Event) { r.dragging = false }),
	}, children))
}

func (r *Resizable) follow(at int, m input.MouseEvent) {
	first, last := r.panels[0].Bounds(), r.panels[len(r.Sizes)-1].Bounds()
	start, length, pointer := r.panels[at].Bounds().Min.X, last.Max.X-first.Min.X, m.X
	if r.Orientation == Vertical {
		start, length, pointer = r.panels[at].Bounds().Min.Y, last.Max.Y-first.Min.Y, m.Y
	}
	if length > 0 {
		r.resize(at, ((pointer-start)*konst.PercentWhole+length-1)/length)
	}
}

func (r *Resizable) resize(at, size int) {
	pair := r.Sizes[at] + r.Sizes[at+1]
	if size = min(max(size, konst.PanelMin), pair-konst.PanelMin); size != r.Sizes[at] {
		r.Sizes[at], r.Sizes[at+1] = size, pair-size
		notify(r.OnResize, slices.Clone(r.Sizes))
		r.rt.Invalidate()
	}
}
