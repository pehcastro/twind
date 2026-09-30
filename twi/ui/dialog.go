package ui

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

type dialogKind uint8

const (
	modal dialogKind = iota
	alert
	palette
	sheet
	drawer
)

type Dialog struct {
	overlay
	kind          dialogKind
	side          Side
	closes, focus int
}

func newDialog(rt *twi.Runtime, kind dialogKind, side Side) *Dialog {
	return &Dialog{overlay: overlay{control: control{rt: rt}}, kind: kind, side: side}
}

func NewDialog(rt *twi.Runtime) *Dialog { return newDialog(rt, modal, Bottom) }

func NewAlertDialog(rt *twi.Runtime) *Dialog { return newDialog(rt, alert, Bottom) }

func NewSheet(rt *twi.Runtime, side Side) *Dialog { return newDialog(rt, sheet, side) }

func NewDrawer(rt *twi.Runtime, side Side) *Dialog { return newDialog(rt, drawer, side) }

func (d *Dialog) Content(children ...twi.NodeOption) twi.Node { return d.content(nil, children) }

func (d *Dialog) content(always, children []twi.NodeOption) twi.Node {
	switch d.kind {
	case modal, sheet:
		children = append(children, d.close("absolute top-1 right-2 rounded-xs opacity-70", "", []twi.NodeOption{icon("✕", "")}))
	case palette:
		children = append(children, d.close("absolute top-0 right-1 rounded-xs opacity-70", "", []twi.NodeOption{icon("✕", "")}))
	case alert, drawer:
	default:
		panic("ui: unknown dialog kind")
	}
	if d.kind != alert {
		children = append(children, twi.OnPointerDownOutside(func() { d.set(false) }))
	}
	if d.kind == drawer && d.side == Bottom {
		children = append([]twi.NodeOption{part("self-center mt-1 h-1 w-12 shrink-0 rounded-full bg-muted", nil)}, children...)
	}
	d.closes = 0
	if !d.Open {
		return part("hidden", append([]twi.NodeOption{twi.Key("closed")}, always...))
	}
	edge := pick("sheet", d.side, map[Side]string{
		Right:  "h-full w-3/4 max-w-48 border-l",
		Left:   "h-full w-3/4 max-w-48 border-r",
		Top:    "w-full border-b",
		Bottom: "w-full border-t",
	})
	backdrop := "flex-col items-center justify-center"
	if d.kind == sheet || d.kind == drawer {
		backdrop = pick("sheet", d.side, map[Side]string{
			Right:  "flex-row justify-end",
			Left:   "flex-row",
			Top:    "flex-col",
			Bottom: "flex-col justify-end",
		})
	}
	boxed := "w-full max-w-64 gap-1 rounded-lg border px-3 py-1 shadow-lg"
	panel := pick("dialog", d.kind, map[dialogKind]string{
		modal:   boxed,
		alert:   boxed,
		palette: "w-full max-w-64 overflow-hidden rounded-lg border shadow-lg",
		sheet:   "gap-1 shadow-lg " + edge,
		drawer:  edge + pick("drawer", d.side, map[Side]string{Bottom: " max-h-[80%]", Top: " max-h-[80%]", Right: "", Left: ""}),
	})
	return part("fixed inset-0 z-50 flex bg-black/50 "+backdrop, append([]twi.NodeOption{
		d.dismissable("relative flex flex-col bg-background text-foreground "+panel, children),
	}, always...))
}

func (d *Dialog) Close(v Variant, s Size, children ...twi.NodeOption) twi.Node {
	return d.close(button(v, s, ""), idleRing(v), children)
}

func (d *Dialog) close(classes, ring string, children []twi.NodeOption) twi.Node {
	d.closes++
	here := d.closes
	focus := func(on bool) func() {
		return func() {
			switch {
			case on:
				d.focus = here
			case d.focus == here:
				d.focus = 0
			}
		}
	}
	return part(classes+" "+ring+" "+focusRing, append([]twi.NodeOption{
		twi.Focusable(), twi.OnFocus(focus(true)), twi.OnBlur(focus(false)), d.click(func() { d.set(false) }),
		keyDown(d.rt, func(k input.KeyEvent) bool {
			if press(k) {
				d.set(false)
			}
			return press(k)
		}),
	}, children...))
}

func (d *Dialog) Header(children ...twi.NodeOption) twi.Node {
	return part(pick("dialog", d.kind, map[dialogKind]string{
		modal:   "flex flex-col text-center sm:text-left",
		alert:   "flex flex-col text-center sm:text-left",
		palette: "flex flex-col text-center sm:text-left",
		sheet:   "flex flex-col px-2 py-1",
		drawer:  "flex flex-col px-2 py-1 text-center md:text-left",
	}), children)
}

func (d *Dialog) Footer(children ...twi.NodeOption) twi.Node {
	return part(pick("dialog", d.kind, map[dialogKind]string{
		modal:   "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
		alert:   "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
		palette: "flex flex-col-reverse gap-2 sm:flex-row sm:justify-end",
		sheet:   "flex flex-col grow justify-end gap-1 px-2 py-1",
		drawer:  "flex flex-col grow justify-end gap-1 px-2 py-1",
	}), children)
}

func (d *Dialog) Title(children ...twi.NodeOption) twi.Node {
	return part("font-semibold", children)
}

func (d *Dialog) Description(children ...twi.NodeOption) twi.Node {
	return part("text-muted-foreground", children)
}
