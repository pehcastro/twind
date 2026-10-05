package ui

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
)

func halo(s style.ComputedStyle, c, halo color.Color) bool {
	return len(s.Shadows) == 2 && len(s.InsetShadows) == 0 && s.Shadows[0].Spread == 1 && s.Shadows[0].Color == c && s.Shadows[1].Spread == 3 && s.Shadows[1].Blur == 0 && s.Shadows[1].Color == halo
}

func scaled(th theme.Theme, tok theme.Token, f float64) color.Color {
	return token(th, tok, uint8(math.Round(float64(th.Tokens[tok].RGBA.A)*f)))
}

func TestControlStates(t *testing.T) {
	light, dark := zinc(t, theme.Light), zinc(t, theme.Dark)
	rt := twi.New(twi.ColorProfile(color.None))
	lightFocus := func(s style.ComputedStyle) bool {
		return halo(s, light.Tokens[theme.Ring], scaled(light, theme.Ring, 0.5))
	}
	faded := func(s style.ComputedStyle) bool { return s.Opacity == 0.5 }
	inputRinged := func(s style.ComputedStyle) bool { return ring(s, light.Tokens[theme.Input]) }
	edged := func(s style.ComputedStyle, c color.Color) bool {
		one := cells(1)
		return s.BorderWidth == style.Edges{Top: one, Right: one, Bottom: one, Left: one} && s.BorderColor == c && len(s.Shadows) == 0
	}
	hairline := func(s style.ComputedStyle, c color.Color) bool {
		half := style.Length{Unit: style.Cells, Value: 0.5}
		return s.BorderWidth == style.Edges{Top: half, Right: half, Bottom: half, Left: half} && s.BorderColor == c && len(s.Shadows) == 0
	}
	checkbox := func(checked, focused, disabled, invalid bool) twi.Node {
		c := NewCheckbox(rt)
		c.Checked, c.focused, c.Disabled, c.Invalid = checked, focused, disabled, invalid
		return c.Node()
	}
	switcher := func(checked, disabled bool) twi.Node {
		s := NewSwitch(rt)
		s.Checked, s.Disabled = checked, disabled
		return s.Node()
	}
	radio := func(value string, focused, disabled bool) twi.Node {
		g := NewRadioGroup(rt)
		g.Value, g.focused, g.Disabled = value, focused, disabled
		a, b, c := g.Item("a", twi.Text("A")), g.Item("b", twi.Text("B")), g.Item("c", twi.Text("C"))
		return g.Node(a, b, c)
	}
	toggle := func(v Variant, pressed, disabled bool) twi.Node {
		tg := NewToggle(rt)
		tg.Variant, tg.Pressed, tg.Disabled = v, pressed, disabled
		return tg.Node(twi.Text("B"))
	}
	group := func(value []string, focused bool) twi.Node {
		g := NewToggleGroup(rt)
		g.Variant, g.Value, g.focused = Outline, value, focused
		x, y := g.Item("x", twi.Text("X")), g.Item("y", twi.Text("Y"))
		return g.Node(x, y)
	}
	slider := func(value, lo, hi int, focused, disabled bool) twi.Node {
		s := NewSlider(rt)
		s.Value, s.Min, s.Max, s.focused, s.Disabled = value, lo, hi, focused, disabled
		return s.Node()
	}
	otp := func(value string, focused bool) twi.Node {
		o := NewInputOTP(rt, 4)
		o.Value, o.focused = value, focused
		return o.Node(InputOTPGroup(o.Slot(0), o.Slot(1)), InputOTPSeparator(), InputOTPGroup(o.Slot(2), o.Slot(3)))
	}
	selector := func(disabled, invalid bool) twi.Node {
		s := NewNativeSelect(rt)
		s.Options, s.Value, s.Disabled, s.Invalid = []string{"one", "two"}, "two", disabled, invalid
		return s.Node()
	}
	field := func(disabled, invalid bool) twi.Node {
		in := NewInput(rt)
		in.Disabled, in.Invalid = disabled, invalid
		return in.Node()
	}
	grouped := func(invalid bool) twi.Node {
		in := NewInput(rt)
		in.Invalid = invalid
		return in.Group(InputGroupAddon(InlineStart, InputGroupText(twi.Text("https://"))), InputGroupAddon(InlineEnd, twi.Text(".com")))
	}
	checkParts(t, []partCase{
		{"checkbox unchecked: a three-cell box in the input ring, no fill", light, checkbox(false, false, false, false), nil, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input]) && s.Width == cells(3) && s.Height == cells(1) && s.Radius == style.RadiusSm && s.Background.Kind == color.Unset && s.Shrink == 0
		}},
		{"checkbox unchecked, dark: bg-input/30", dark, checkbox(false, false, false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == scaled(dark, theme.Input, 0.3) && ring(s, dark.Tokens[theme.Input])
		}},
		{"checkbox checked: bg-primary text-primary-foreground in a primary ring", light, checkbox(true, false, false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Color == light.Tokens[theme.PrimaryForeground] && ring(s, light.Tokens[theme.Primary])
		}},
		{"checkbox checked, dark: bg-primary, not input/30", dark, checkbox(true, false, false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == dark.Tokens[theme.Primary]
		}},
		{"checkbox the component thinks focused, without the runtime's focus: the idle ring", light, checkbox(false, true, false, false), nil, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input])
		}},
		{"checkbox disabled: opacity-50", light, checkbox(true, false, true, false), nil, faded},
		{"checkbox invalid: a destructive ring", light, checkbox(false, false, false, true), nil, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Destructive])
		}},
		{"switch unchecked: a bg-input pill, thumb at the start", light, switcher(false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Input] && s.Justify == style.JustifyStart && s.Width == cells(4) && s.Height == cells(1) && s.Radius == style.RadiusFull && len(s.Shadows) == 0
		}},
		{"switch unchecked, dark: bg-input/80", dark, switcher(false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == scaled(dark, theme.Input, 0.8)
		}},
		{"switch thumb unchecked: a bg-background circle", light, switcher(false, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Background] && s.Width == cells(2) && s.Radius == style.RadiusFull
		}},
		{"switch thumb unchecked, dark: bg-foreground", dark, switcher(false, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == dark.Tokens[theme.Foreground]
		}},
		{"switch checked: bg-primary, thumb at the end", light, switcher(true, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Justify == style.JustifyEnd
		}},
		{"switch thumb checked, dark: bg-primary-foreground", dark, switcher(true, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == dark.Tokens[theme.PrimaryForeground]
		}},
		{"switch disabled", light, switcher(false, true), nil, faded},
		{"radio group: a column, one row between items", light, radio("b", false, false), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column && s.RowGap == cells(1)
		}},
		{"radio item: circle and label in a row, gap-3 as shadcn's radio rows, room for the ring and the pill caps", light, radio("b", false, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.ColumnGap == cells(3)
		}},
		{"radio unchecked: a three-cell circle in the input ring", light, radio("b", false, false), []int{0, 0}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input]) && s.Radius == style.RadiusFull && s.Width == cells(3) && s.Background.Kind == color.Unset
		}},
		{"radio checked: text-primary for the dot", light, radio("b", false, false), []int{1, 0}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Primary] && ring(s, light.Tokens[theme.Input])
		}},
		{"radio unchecked, dark: bg-input/30", dark, radio("b", false, false), []int{2, 0}, func(s style.ComputedStyle) bool {
			return s.Background == scaled(dark, theme.Input, 0.3)
		}},
		{"radio focused, without the runtime's focus-visible: the checked item keeps its idle ring", light, radio("b", true, false), []int{1, 0}, inputRinged},
		{"radio focused: not on the others", light, radio("b", true, false), []int{0, 0}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input])
		}},
		{"radio focused with no value, without the runtime's focus-visible: the first item keeps its idle ring", light, radio("", true, false), []int{0, 0}, inputRinged},
		{"radio disabled", light, radio("b", false, true), nil, faded},
		{"toggle off: no fill, rounded-md", light, toggle(Default, false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background.Kind == color.Unset && s.Radius == style.RadiusMd && len(s.Shadows) == 0 && s.Height == cells(1)
		}},
		{"toggle on: bg-accent text-accent-foreground", light, toggle(Default, true, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent] && s.Color == light.Tokens[theme.AccentForeground]
		}},
		{"toggle outline: the input ring", light, toggle(Outline, false, false), nil, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input])
		}},
		{"toggle disabled", light, toggle(Default, false, true), nil, faded},
		{"toggle group: a row", light, group(nil, false), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter
		}},
		{"toggle group item on: bg-accent in the outline ring", light, group([]string{"y"}, false), []int{1}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent] && ring(s, light.Tokens[theme.Input])
		}},
		{"toggle group item off", light, group([]string{"y"}, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background.Kind == color.Unset && ring(s, light.Tokens[theme.Input])
		}},
		{"toggle group focused, without the runtime's focus-visible: the first item keeps its idle ring", light, group([]string{"y"}, true), []int{0}, inputRinged},
		{"toggle group focused: not on the second", light, group([]string{"y"}, true), []int{1}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input])
		}},
		{"slider: a full-width bg-muted track", light, slider(37, 0, 100, false, false), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.Width == percent(100) && s.AlignItems == style.AlignCenter && s.Background == light.Tokens[theme.Muted] && s.Radius == style.RadiusFull
		}},
		{"slider range at 37: bg-primary 37% wide", light, slider(37, 0, 100, false, false), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Width == percent(37) && s.Radius == style.RadiusFull
		}},
		{"slider thumb: white in a primary ring", light, slider(37, 0, 100, false, false), []int{1}, func(s style.ComputedStyle) bool {
			return s.Background == color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255}} && ring(s, light.Tokens[theme.Primary]) && s.Width == cells(2) && s.Shrink == 0
		}},
		{"slider focused, without the runtime's focus-visible: the thumb keeps its primary ring", light, slider(37, 0, 100, true, false), []int{1}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Primary])
		}},
		{"slider over max clamps to 100%", light, slider(140, 0, 100, false, false), []int{0}, func(s style.ComputedStyle) bool { return s.Width == percent(100) }},
		{"slider on a 10 to 20 scale: 15 is 50%", light, slider(15, 10, 20, false, false), []int{0}, func(s style.ComputedStyle) bool { return s.Width == percent(50) }},
		{"slider with min equal to max: 0%", light, slider(5, 5, 5, false, false), []int{0}, func(s style.ComputedStyle) bool { return s.Width == percent(0) }},
		{"slider disabled", light, slider(37, 0, 100, false, true), nil, faded},
		{"otp: slots and separator in a row", light, otp("12", false), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter
		}},
		{"otp slot: a three-cell box in the input ring", light, otp("12", false), []int{0, 0}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input]) && s.Width == cells(3) && s.Justify == style.JustifyCenter
		}},
		{"otp focused, without the runtime's focus-visible: the next empty slot keeps its idle ring", light, otp("12", true), []int{2, 0}, inputRinged},
		{"otp focused: not on a filled slot", light, otp("12", true), []int{0, 1}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input])
		}},
		{"otp full and focused, without the runtime's focus-visible: the last slot keeps its idle ring", light, otp("1234", true), []int{2, 1}, inputRinged},
		{"native select: a one-row box in the input ring", light, selector(false, false), []int{0}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Input]) && s.Height == cells(1) && s.Radius == style.RadiusMd && s.Direction == style.Row
		}},
		{"native select invalid", light, selector(false, true), []int{0}, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Destructive])
		}},
		{"native select disabled", light, selector(true, false), []int{0}, faded},
		{"native select chevron: muted at half opacity", light, selector(false, false), []int{0, 1}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground] && s.Opacity == 0.5
		}},
		{"input: a full-width box in a half-row input border around its text row, two rows with pixels and three without", light, field(false, false), nil, func(s style.ComputedStyle) bool {
			return hairline(s, light.Tokens[theme.Input]) && s.Height.Unit == style.Auto && s.Shrink == 0 && s.Padding.Left == cells(1) && s.Width == percent(100) && s.Radius == style.RadiusMd && s.OverflowX == style.OverflowHidden
		}},
		{"input, dark: bg-input/30", dark, field(false, false), nil, func(s style.ComputedStyle) bool {
			return s.Background == scaled(dark, theme.Input, 0.3)
		}},
		{"input invalid: a destructive border, as thin as the valid one", light, field(false, true), nil, func(s style.ComputedStyle) bool {
			return hairline(s, light.Tokens[theme.Destructive])
		}},
		{"input disabled", light, field(true, false), nil, faded},
		{"textarea: a bordered column, four text rows at least", light, NewTextarea(rt).Node(), nil, func(s style.ComputedStyle) bool {
			return edged(s, light.Tokens[theme.Input]) && s.Direction == style.Column && s.MinHeight == cells(6) && s.Width == percent(100)
		}},
		{"input group: the border on the group", light, grouped(false), nil, func(s style.ComputedStyle) bool {
			return edged(s, light.Tokens[theme.Input]) && s.Radius == style.RadiusMd
		}},
		{"input group invalid: a destructive border on the group", light, grouped(true), nil, func(s style.ComputedStyle) bool {
			return edged(s, light.Tokens[theme.Destructive])
		}},
		{"input group: the field inside has no border of its own", light, grouped(false), []int{0, 1}, func(s style.ComputedStyle) bool {
			return s.BorderWidth == style.Edges{} && s.Height == cells(1) && s.Grow == 1
		}},
		{"input group addon: muted", light, grouped(false), []int{0, 0}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground] && s.Padding.Left == cells(1)
		}},
	})
	root := []int{}
	checkFocused(t, []focusCase{
		{"checkbox focused: the ring and a 3px ring/50 halo", light, checkbox(false, false, false, false), root, nil, lightFocus},
		{"checkbox focused and checked: the focus ring over bg-primary", light, checkbox(true, false, false, false), root, nil, func(s style.ComputedStyle) bool {
			return lightFocus(s) && s.Background == light.Tokens[theme.Primary]
		}},
		{"checkbox invalid and focused: destructive with a destructive/20 halo", light, checkbox(false, false, false, true), root, nil, func(s style.ComputedStyle) bool {
			return halo(s, light.Tokens[theme.Destructive], scaled(light, theme.Destructive, 0.2))
		}},
		{"checkbox invalid and focused, dark: the dark destructive with a /40 halo", dark, checkbox(false, false, false, true), root, nil, func(s style.ComputedStyle) bool {
			return halo(s, dark.Tokens[theme.Destructive], scaled(dark, theme.Destructive, 0.4))
		}},
		{"switch focused", light, switcher(true, false), root, nil, lightFocus},
		{"toggle focused", light, toggle(Default, true, false), root, nil, lightFocus},
		{"native select focused", light, selector(false, false), []int{0}, []int{0}, lightFocus},
		{"input focused: the border turns ring, no halo", light, field(false, false), root, nil, func(s style.ComputedStyle) bool {
			return hairline(s, light.Tokens[theme.Ring])
		}},
		{"textarea focused: the border turns ring, no halo", light, NewTextarea(rt).Node(), root, nil, func(s style.ComputedStyle) bool {
			return edged(s, light.Tokens[theme.Ring])
		}},
		{"input group: the input has no ring of its own while focused", light, grouped(false), []int{0, 1}, []int{0, 1}, func(s style.ComputedStyle) bool {
			return len(s.Shadows) == 0 && s.Grow == 1
		}},
	})
}

func TestControlKeys(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	var (
		in     *Input
		ta     *Textarea
		cb     *Checkbox
		sw     *Switch
		off    *Toggle
		rg     *RadioGroup
		tg     *ToggleGroup
		sl     *Slider
		otp    *InputOTP
		sel    *NativeSelect
		heard  []rune
		change []string
	)
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		in, ta, cb, sw, off = NewInput(rt), NewTextarea(rt), NewCheckbox(rt), NewSwitch(rt), NewToggle(rt)
		rg, tg, sl, otp, sel = NewRadioGroup(rt), NewToggleGroup(rt), NewSlider(rt), NewInputOTP(rt, 4), NewNativeSelect(rt)
		off.Disabled = true
		cb.OnChange = func(v bool) { change = append(change, "checkbox "+map[bool]string{true: "on", false: "off"}[v]) }
		rg.OnChange = func(v string) { change = append(change, "radio "+v) }
		sl.Min, sl.Max, sl.Step, sl.Value = 0, 10, 3, 5
		sel.Options, sel.Value = []string{"one", "two", "three"}, "two"
		return func() twi.Node {
			a, b, c := rg.Item("a", twi.Text("A")), rg.Item("b", twi.Text("B")), rg.Item("c", twi.Text("C"))
			x, y, z := tg.Item("x", twi.Text("X")), tg.Item("y", twi.Text("Y")), tg.Item("z", twi.Text("Z"))
			return twi.Element(twi.Class("flex flex-col gap-1 p-1"),
				twi.OnKey(func(k input.KeyEvent) {
					if !k.Release && k.Key == input.KeyRune {
						heard = append(heard, k.Rune)
					}
				}),
				in.Node(), ta.Node(), cb.Node(), sw.Node(), off.Node(twi.Text("off")), rg.Node(a, b, c), tg.Node(x, y, z), sl.Node(),
				otp.Node(otp.Slot(0), otp.Slot(1), otp.Slot(2), otp.Slot(3)), sel.Node(),
			)
		}
	}, drive.Size(40, 40), drive.Styles(sheet))
	defer func() {
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	focus := func(want *control) {
		t.Helper()
		before := d.Frame().ANSI()
		d.Press("tab")
		var on []*control
		for _, c := range []*control{&in.control, &ta.control, &cb.control, &sw.control, &off.control, &rg.control, &tg.control, &sl.control, &otp.control, &sel.control} {
			if c.focused {
				on = append(on, c)
			}
		}
		if len(on) != 1 || on[0] != want {
			t.Fatalf("after tab, %d controls hold focus, want exactly the next one", len(on))
		}
		if d.Frame().ANSI() == before {
			t.Error("tab moved focus but no new frame shows it")
		}
	}
	step := func(keys string, got func() any, want any) {
		t.Helper()
		for k := range strings.FieldsSeq(keys) {
			if typed, ok := strings.CutPrefix(k, "type:"); ok {
				d.Type(typed)
				continue
			}
			d.Press(k)
		}
		if g := got(); !equal(g, want) {
			t.Errorf("after %s: %v, want %v", keys, g, want)
		}
	}
	focus(&in.control)
	step("type:hi", func() any { return in.Value() }, "hi")
	focus(&ta.control)
	step("type:a shift+enter type:b", func() any { return ta.Value() }, "a\nb")
	if frame := d.Frame().Text(); !regexp.MustCompile(`[│▐] a +[│▌]\n *[│▐] b[ \x{a0}]+[│▌]`).MatchString(frame) {
		t.Errorf("the textarea does not break the line:\n%s", frame)
	}
	focus(&cb.control)
	step("enter", func() any { return cb.Checked }, false)
	step("space", func() any { return cb.Checked }, true)
	if !strings.Contains(d.Frame().Text(), "✓") {
		t.Errorf("a checked checkbox shows no mark:\n%s", d.Frame().Text())
	}
	step("space", func() any { return cb.Checked }, false)
	focus(&sw.control)
	step("space", func() any { return sw.Checked }, true)
	step("enter", func() any { return sw.Checked }, false)
	focus(&rg.control)
	step("space", func() any { return rg.Value }, "a")
	step("down", func() any { return rg.Value }, "b")
	step("right down", func() any { return rg.Value }, "a")
	step("up", func() any { return rg.Value }, "c")
	step("left", func() any { return rg.Value }, "b")
	focus(&tg.control)
	step("space", func() any { return tg.Value }, []string{"x"})
	step("right enter", func() any { return tg.Value }, []string{"y"})
	step("space", func() any { return tg.Value }, []string{})
	step("right right", func() any { return tg.Value }, []string{})
	step("space", func() any { return tg.Value }, []string{"x"})
	tg.Multiple = true
	step("end space", func() any { return tg.Value }, []string{"x", "z"})
	focus(&sl.control)
	step("right", func() any { return sl.Value }, 8)
	step("up", func() any { return sl.Value }, 10)
	step("home", func() any { return sl.Value }, 0)
	step("left", func() any { return sl.Value }, 0)
	step("end", func() any { return sl.Value }, 10)
	step("pagedown", func() any { return sl.Value }, 0)
	focus(&otp.control)
	step("type:1 space type:2a4", func() any { return otp.Value }, "12a4")
	step("type:9", func() any { return otp.Value }, "12a4")
	step("backspace backspace backspace backspace backspace", func() any { return otp.Value }, "")
	step("type:12", func() any { return otp.Value }, "12")
	focus(&sel.control)
	step("down", func() any { return sel.Value }, "three")
	step("down", func() any { return sel.Value }, "three")
	step("up", func() any { return sel.Value }, "two")
	step("home", func() any { return sel.Value }, "one")
	step("end", func() any { return sel.Value }, "three")
	focus(&in.control)
	if len(rg.items) != 3 || len(tg.items) != 3 {
		t.Errorf("the groups remember %d and %d items after many frames, want 3 each", len(rg.items), len(tg.items))
	}
	if len(heard) != 0 {
		t.Errorf("the page heard keys a control took: %q", string(heard))
	}
	if want := []string{"checkbox on", "checkbox off", "radio a", "radio b", "radio c", "radio a", "radio c", "radio b"}; !slices.Equal(change, want) {
		t.Errorf("OnChange calls %q, want %q", change, want)
	}
	if err := d.Err(); err != nil {
		t.Fatal(err)
	}
}

func equal(a, b any) bool {
	if as, ok := a.([]string); ok {
		bs, _ := b.([]string)
		return slices.Equal(as, bs)
	}
	return a == b
}
