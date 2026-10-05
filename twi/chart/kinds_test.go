package chart

import (
	"image"
	"math"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/chart"
	paintkonst "github.com/twind-dev/twind/internal/konst/paint"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/text"
	"github.com/twind-dev/twind/twi/theme"
)

func clock(c point, radius, angle float32) point {
	x := float64(c.x) + float64(radius)*math.Sin(float64(angle))
	y := float64(c.y) - float64(radius)*math.Cos(float64(angle))
	return point{float32(x), float32(y)}
}

func browsers(kind Kind, values ...float64) *Chart {
	c := &Chart{Kind: kind, Labels: []string{"Visitors"}}
	for s, v := range values {
		c.Series = append(c.Series, Series{Label: []string{"Chrome", "Safari", "Firefox", "Edge", "Other"}[s], Color: theme.Chart1 + theme.Token(s), Values: []float64{v}})
	}
	return c
}

func TestPieSlicesSumToAFullTurn(t *testing.T) {
	for _, values := range [][]float64{{275, 200, 187}, {1, 0, 2}, {5}, {0.1, 0.2, 0.3}} {
		edges := browsers(Pie, values...).model().slices()
		if edges[0] != 0 || edges[len(edges)-1] != turn {
			t.Errorf("%v: slices run from %v to %v, want 0 to a full turn %v", values, edges[0], edges[len(edges)-1], float32(turn))
		}
		total := 0.0
		for _, v := range values {
			total += v
		}
		for s, v := range values {
			if got, want := float64(edges[s+1]-edges[s]), turn*v/total; math.Abs(got-want) > 1e-5 {
				t.Errorf("%v: slice %d spans %.6f, want %.6f", values, s, got, want)
			}
		}
	}
}

func TestPieSlicesRunClockwiseFromTwelveWithoutSeams(t *testing.T) {
	c := browsers(Pie, 1, 1, 2)
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw(c, dst, image.Point{10, 20})
	g := c.model().wheel(200, 200, point{10, 20}, 0)
	at := func(radius, angle float32) [4]uint8 {
		p := clock(g.centre, radius, angle)
		o := dst.RGBAAt(int(p.x), int(p.y))
		return [4]uint8{o.R, o.G, o.B, o.A}
	}
	for _, k := range []struct {
		angle float32
		want  [4]uint8
		name  string
	}{{turn / 8, [4]uint8{255, 0, 0, 255}, "chart-1 between 12 and 3 o'clock"}, {turn * 3 / 8, [4]uint8{0, 0, 255, 255}, "chart-2 between 3 and 6"}, {turn * 3 / 4, [4]uint8{0, 255, 0, 255}, "chart-3 over the left half"}} {
		if got := at(g.radius/2, k.angle); got != k.want {
			t.Errorf("pixel at %.2f rad is %v, want %s %v", k.angle, got, k.name, k.want)
		}
	}
	edges := browsers(Pie, 1, 2, 3).model().slices()
	c.Series[1].Values[0], c.Series[2].Values[0] = 2, 3
	clear(dst.Pix)
	draw(c, dst, image.Point{10, 20})
	for _, edge := range edges[1:3] {
		for r := g.radius * 0.2; r < g.radius*0.9; r += 3 {
			if got := at(r, edge); got[3] != math.MaxUint8 {
				t.Fatalf("pixel on the edge at %.2f rad, radius %.0f has alpha %d: a seam between slices", edge, r, got[3])
			}
		}
	}
	if got := at(g.radius*1.05, turn/8); got[3] != 0 {
		t.Errorf("pixel outside the pie has alpha %d", got[3])
	}
	c = browsers(Pie, 1, 1, 2)
	c.hover = 1
	clear(dst.Pix)
	draw(c, dst, image.Point{10, 20})
	if got := at(g.radius+g.grow/2, turn/8); got != [4]uint8{255, 0, 0, 255} {
		t.Errorf("hovered slice does not grow: pixel past the radius is %v", got)
	}
	if got := at(g.radius+g.grow/2, turn*3/8); got[3] != 0 {
		t.Errorf("a slice that is not hovered grows: alpha %d past the radius", got[3])
	}
}

func TestDonutLeavesItsHoleEmpty(t *testing.T) {
	c := browsers(Donut, 3, 2)
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw(c, dst, image.Point{10, 20})
	g := c.model().wheel(200, 200, point{10, 20}, 0)
	if o := dst.RGBAAt(100, 100); o.A != 0 {
		t.Errorf("donut centre has alpha %d, want the hole", o.A)
	}
	p := clock(g.centre, (g.radius+g.hole)/2, turn/4)
	if o := dst.RGBAAt(int(p.x), int(p.y)); o.R != 255 || o.A != 255 {
		t.Errorf("donut ring at 3 o'clock is %v, want chart-1", o)
	}
}

func TestRadialRingsTurnByTheirShareOfTheLargest(t *testing.T) {
	c := browsers(Radial, 200, 100)
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw(c, dst, image.Point{10, 20})
	g := c.model().wheel(200, 200, point{10, 20}, 0)
	ring := func(s int, angle float32) [3]uint8 {
		p := clock(g.centre, g.radius-g.band*(float32(s)+0.5), angle)
		o := dst.RGBAAt(int(p.x), int(p.y))
		return [3]uint8{o.R, o.G, o.B}
	}
	for _, a := range []float32{turn / 8, turn / 2, turn * 7 / 8} {
		if got := ring(0, a); got != [3]uint8{255, 0, 0} {
			t.Errorf("the largest value is not a full turn: outer ring is %v at %.2f rad", got, a)
		}
	}
	muted := [3]uint8{128, 128, 128}
	if ring(1, turn/4) != [3]uint8{0, 0, 255} || ring(1, turn*3/4) != muted {
		t.Errorf("half the largest value: ring at 3 o'clock %v (want chart-2), at 9 o'clock %v (want the muted track)", ring(1, turn/4), ring(1, turn*3/4))
	}
}

func TestRadarPointsSitOnTheirAxes(t *testing.T) {
	c := &Chart{Kind: Radar, Labels: []string{"a", "b", "c", "d", "e", "f"}, Series: []Series{{Color: theme.Chart1, Values: []float64{100, 80, 100, 80, 100, 80}}}}
	dst := image.NewRGBA(image.Rect(0, 0, 300, 300))
	draw(c, dst, image.Point{10, 20})
	p := c.model()
	g := p.wheel(300, 300, point{10, 20}, 0)
	for i, v := range c.Series[0].Values {
		a := turn * float32(i) / 6
		in, out := clock(g.centre, g.reach(p, v)*0.93, a), clock(g.centre, g.reach(p, v)*1.07, a)
		if a := dst.RGBAAt(int(in.x), int(in.y)).A; a == 0 {
			t.Errorf("axis %d: nothing just inside its vertex at %v", i, in)
		}
		if a := dst.RGBAAt(int(out.x), int(out.y)).R; a != 0 {
			t.Errorf("axis %d: fill just past its vertex at %v", i, out)
		}
	}
	if top := clock(g.centre, g.reach(p, 100)*0.9, 0); dst.RGBAAt(int(top.x), int(top.y)).A == 0 {
		t.Errorf("the first axis does not point to 12 o'clock")
	}
}

func TestStepChangesHalfwayBetweenPoints(t *testing.T) {
	got := steps(nil, []point{{0, 4}, {10, 8}, {20, 2}})
	want := []point{{0, 4}, {5, 4}, {5, 8}, {10, 8}, {15, 8}, {15, 2}, {20, 2}}
	if len(got) != len(want) {
		t.Fatalf("step curve %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step point %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestHorizontalBarsRoundOnlyTheirFarEnd(t *testing.T) {
	c := &Chart{Kind: Bar, Horizontal: true, Labels: []string{"a", "b"}, Series: []Series{{Color: theme.Chart1, Negative: theme.Chart2, Values: []float64{4, -4}}}}
	p := c.model()
	dst := image.NewRGBA(image.Rect(0, 0, 200, 100))
	c.canvas = reuse(nil, 200, 100)
	m := pixelMetrics(image.Point{10, 20})
	m.top = 200
	c.canvas.dst, c.canvas.token, c.canvas.dot = dst, palette, 1
	c.trace(c.canvas, p, m, 0)
	zero := int(p.y(0, m))
	lo, hi := p.bar(0, 0, 100, m)
	end := int(p.y(4, m)) - 1
	if o := dst.RGBAAt(zero, int(lo+hi)/2); o.A != math.MaxUint8 || o.R != 255 {
		t.Errorf("positive bar at the zero line %d,%d is %v: want chart-1 from the zero line", zero, int(lo+hi)/2, o)
	}
	if o := dst.RGBAAt(end, int(lo)); o.A == math.MaxUint8 {
		t.Errorf("positive bar far corner %d,%d is square", end, int(lo))
	}
	lo, hi = p.bar(1, 0, 100, m)
	far := int(p.y(-4, m))
	if o := dst.RGBAAt(zero-1, int(lo+hi)/2); o.A != math.MaxUint8 || o.B != 255 {
		t.Errorf("negative bar at the zero line is %v: want chart-2 up to the zero line", o)
	}
	if o := dst.RGBAAt(far, int(lo)); o.A == math.MaxUint8 {
		t.Errorf("negative bar far corner %d,%d is square", far, int(lo))
	}
	if o := dst.RGBAAt(far+1, int(lo+hi)/2); o.B != 255 || o.R != 0 {
		t.Errorf("negative bar does not reach left to its value: %v at %d", o, far+1)
	}
}

func TestDitherIsOrderedDots(t *testing.T) {
	seen := map[float32]bool{}
	for y := range 4 {
		for x := range 4 {
			seen[bayer(x, y)] = true
			if bayer(x, y) != bayer(x+4, y+8) {
				t.Errorf("threshold at %d,%d does not repeat every 4", x, y)
			}
		}
	}
	if len(seen) != 16 {
		t.Errorf("a 4x4 ordered dither has 16 thresholds, got %d", len(seen))
	}
	c := &Chart{Kind: Bar, Dither: true, Labels: []string{"a"}, Series: []Series{{Color: theme.Chart1, Values: []float64{4}}}}
	dst := image.NewRGBA(image.Rect(0, 0, 100, 200))
	draw(c, dst, image.Point{10, 20})
	lit := [2]int{}
	for y := range 200 {
		for x := range 100 {
			switch a := dst.RGBAAt(x, y).A; a {
			case 0:
			case math.MaxUint8:
				lit[y*2/200]++
			default:
				t.Fatalf("dithered pixel %d,%d has alpha %d: dots are on or off", x, y, a)
			}
		}
	}
	if lit[0] <= lit[1] || lit[1] == 0 {
		t.Errorf("dithered bar lights %d dots in its top half and %d in its bottom: want fewer going down, never none", lit[0], lit[1])
	}
}

func TestXLabelsThinOutInsteadOfOverlapping(t *testing.T) {
	var names []string
	for i := range 30 {
		names = append(names, "Jun "+string(rune('a'+i%26)))
	}
	line := xAxis(text.Widths{}, names, 40)
	if strings.Contains(line, "Jun aJun") || len(strings.Fields(line)) < 4 {
		t.Errorf("x axis %q: labels touch or too few shown", line)
	}
	for _, word := range regexp.MustCompile(`Jun \w`).FindAllString(line, -1) {
		if len(word) != len("Jun a") {
			t.Errorf("label %q cut", word)
		}
	}
}

func TestMapFillsRegionsByValue(t *testing.T) {
	c := &Chart{Kind: Map, Labels: []string{"BRA", "USA", "AUS"}, Series: []Series{{Label: "Visitors", Color: theme.Chart1, Values: []float64{300, 100, 200}}}}
	const w, h = 600, 300
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw(c, dst, image.Point{10, 20})
	a := c.atlas()
	inside := func(code string) image.Point {
		for i, k := range a.countries {
			if k.code != code {
				continue
			}
			best, most := image.Point{}, 0
			for y := 0; y < h; y += 2 {
				for x := 0; x < w; x += 2 {
					if a.locate(w, h, point{float32(x) + 0.5, float32(y) + 0.5}) != i+1 {
						continue
					}
					n := 0
					for _, d := range []image.Point{{-3, 0}, {3, 0}, {0, -3}, {0, 3}} {
						if a.locate(w, h, point{float32(x+d.X) + 0.5, float32(y+d.Y) + 0.5}) == i+1 {
							n++
						}
					}
					if n > most {
						best, most = image.Pt(x, y), n
					}
				}
			}
			if most == 4 {
				return best
			}
		}
		t.Fatalf("no pixel well inside %s", code)
		return image.Point{}
	}
	bra, usa, aus, can := dst.RGBAAt(inside("BRA").X, inside("BRA").Y), dst.RGBAAt(inside("USA").X, inside("USA").Y), dst.RGBAAt(inside("AUS").X, inside("AUS").Y), dst.RGBAAt(inside("CAN").X, inside("CAN").Y)
	if bra.R != 255 || bra.A != 255 {
		t.Errorf("Brazil, the largest value, is %v: want chart-1 at full strength", bra)
	}
	if floor := konst.MapFloor * math.MaxUint8; usa.G != 0 || math.Abs(float64(usa.A)-floor) > 2 {
		t.Errorf("USA, the smallest value, is %v: want chart-1 at the floor alpha", usa)
	}
	if aus.A <= usa.A || aus.A >= bra.A {
		t.Errorf("Australia's alpha %d is not between USA %d and Brazil %d", aus.A, usa.A, bra.A)
	}
	if can.G != 128 || can.R != 128 {
		t.Errorf("Canada has no data and is %v: want muted", can)
	}
	if len(worldData) > konst.WorldMaxBytes {
		t.Errorf("embedded world is %d bytes, over %d", len(worldData), konst.WorldMaxBytes)
	}
	defer func() {
		if recover() == nil {
			t.Errorf("an unknown country code was accepted")
		}
	}()
	c.Labels[0] = "XXX"
	c.regions(c.model())
}

func TestEveryKindDrivesWithTooltipAndConsoleGlyphs(t *testing.T) {
	for _, k := range []struct {
		name  string
		build func(rt *twi.Runtime) *Chart
		want  []string
		at    image.Point
		tip   string
	}{
		{"pie", func(rt *twi.Runtime) *Chart { return sized(rt, browsers(Pie, 275, 200, 187, 173, 90)) }, []string{"■ Chrome"}, image.Pt(52, 6), `Chrome +275`},
		{"donut", func(rt *twi.Runtime) *Chart { return sized(rt, browsers(Donut, 275, 200, 287, 173, 190)) }, []string{"1,125", "Visitors"}, image.Pt(52, 4), `Chrome +275`},
		{"radial", func(rt *twi.Runtime) *Chart { return sized(rt, browsers(Radial, 275, 200, 187, 173, 90)) }, []string{"■ Edge"}, image.Pt(70, 12), `Chrome +275`},
		{"radar", func(rt *twi.Runtime) *Chart {
			return sized(rt, &Chart{Kind: Radar, Labels: []string{"January", "February", "March", "April", "May", "June"}, Series: []Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 273, 209, 214}}}})
		}, []string{"January", "April", "June"}, image.Pt(50, 3), `January[\s\S]*Desktop +186`},
		{"labelled bar", func(rt *twi.Runtime) *Chart {
			return sized(rt, &Chart{Kind: Bar, Labelled: true, Labels: []string{"Jan", "Feb", "Mar"}, Series: []Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 73}}}})
		}, []string{"186", "305", "73"}, image.Pt(10, 12), `Jan[\s\S]*Desktop +186`},
		{"horizontal bar", func(rt *twi.Runtime) *Chart {
			return sized(rt, &Chart{Kind: Bar, Horizontal: true, Labels: []string{"Jan", "Feb", "Mar"}, Series: []Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 73}}}})
		}, []string{"Jan", "Mar"}, image.Pt(12, 3), `Jan[\s\S]*Desktop +186`},
		{"dithered step area", func(rt *twi.Runtime) *Chart {
			return sized(rt, &Chart{Kind: Area, Step: true, Dither: true, Labels: []string{"Jan", "Feb", "Mar", "Apr"}, Series: []Series{{Label: "Desktop", Color: theme.Chart1, Values: []float64{186, 305, 237, 73}}}})
		}, []string{"Jan", "Apr"}, image.Pt(8, 12), `Jan[\s\S]*Desktop +186`},
		{"map", func(rt *twi.Runtime) *Chart {
			return sized(rt, &Chart{Kind: Map, Labels: []string{"BRA", "USA", "IND"}, Series: []Series{{Label: "Visitors", Color: theme.Chart1, Values: []float64{300, 100, 200}}}})
		}, []string{"■ Visitors", "100", "300"}, image.Pt(32, 15), `Brazil[\s\S]*Visitors +300`},
	} {
		d, c := chartDriver(t, k.build)
		frame := d.Frame().Text()
		for _, w := range k.want {
			if !strings.Contains(frame, w) {
				t.Errorf("%s: frame has no %q:\n%s", k.name, w, frame)
			}
		}
		if i := strings.IndexFunc(frame, func(r rune) bool {
			return r >= utf8.RuneSelf && !strings.ContainsRune(paintkonst.ConsoleGlyphs, r)
		}); i >= 0 {
			r, _ := utf8.DecodeRuneInString(frame[i:])
			t.Errorf("%s draws %q, outside the console-safe set", k.name, r)
		}
		d.Move(k.at.X, k.at.Y)
		hovered := d.Frame().Text()
		if !regexp.MustCompile(k.tip).MatchString(hovered) {
			t.Errorf("%s: pointer at %v (hover %d) shows no tooltip matching %q:\n%s", k.name, k.at, c.hover, k.tip, hovered)
		}
		t.Logf("%s, 100x30, no graphics, pointer at %v:\n%s", k.name, k.at, hovered)
	}
}

func BenchmarkKinds(b *testing.B) {
	for _, c := range []*Chart{
		browsers(Pie, 275, 200, 187, 173, 90),
		browsers(Radial, 275, 200, 187, 173, 90),
		{Kind: Radar, Labels: []string{"a", "b", "c", "d", "e", "f"}, Series: []Series{{Color: theme.Chart1, Values: []float64{186, 305, 237, 273, 209, 214}}}},
		{Kind: Map, Labels: []string{"BRA", "USA", "IND"}, Series: []Series{{Color: theme.Chart1, Values: []float64{300, 100, 200}}}},
		{Kind: Area, Dither: true, Labels: []string{"a", "b", "c", "d", "e", "f"}, Series: []Series{{Color: theme.Chart1, Values: []float64{186, 305, 237, 273, 209, 214}}}},
	} {
		dst := image.NewRGBA(image.Rect(0, 0, 900, 440))
		name := map[Kind]string{Pie: "pie", Radial: "radial", Radar: "radar", Map: "map", Area: "dithered-area"}[c.Kind]
		b.Run(name+"/pixels-900x440", func(b *testing.B) {
			for b.Loop() {
				draw(c, dst, image.Point{10, 20})
			}
		})
	}
}

func sized(rt *twi.Runtime, c *Chart) *Chart {
	c.rt, c.Width, c.Height = rt, 96, 26
	return c
}

func TestBrushDragZoomsAndClickResets(t *testing.T) {
	d, c := chartDriver(t, func(rt *twi.Runtime) *Chart {
		var labels []string
		var values []float64
		for i := range 40 {
			labels = append(labels, "D"+string(rune('a'+i/26))+string(rune('a'+i%26)))
			values = append(values, float64(100+i*i%70))
		}
		c := New(rt)
		c.Kind, c.Brush, c.Labels, c.Series = Area, true, labels, []Series{{Label: "Visitors", Color: theme.Chart1, Values: values}}
		return sized(rt, c)
	})
	lines := strings.Split(d.Frame().Text(), "\n")
	strip := len(lines) - 1
	for strip > 0 && !strings.ContainsAny(lines[strip], konst.Upper+konst.Lower+konst.Full) {
		strip--
	}
	axis := 7
	d.Move(axis+30, strip)
	if c.From != 0 || c.To != 0 {
		t.Fatalf("a pointer passing over the brush with no button moved the window to %d to %d", c.From, c.To)
	}
	d.Down(axis+30, strip)
	d.Move(0, strip)
	d.Move(99, strip)
	d.Move(0, strip)
	d.Up(0, strip)
	if c.From != 0 || c.To == 0 || c.To > 40 {
		t.Errorf("a drag past both edges left the window at %d to %d, want 0 to under 40", c.From, c.To)
	}
	d.Down(axis+20, strip)
	d.Move(axis+40, strip)
	d.Up(axis+40, strip)
	zoomed := d.Frame().Text()
	if c.From == 0 || c.To == 0 || c.To-c.From >= 40 {
		t.Fatalf("dragging the brush from column 20 to 40 left the window at %d to %d:\n%s", c.From, c.To, zoomed)
	}
	if strings.Contains(zoomed, "Daa") {
		t.Errorf("zoomed to %d..%d but the first label still shows:\n%s", c.From, c.To, zoomed)
	}
	d.Down(axis+60, strip)
	d.Move(axis+50, strip)
	d.Up(axis+50, strip)
	if c.To-c.From <= konst.MinWindow {
		t.Errorf("a right to left drag over ten columns gave the window %d to %d", c.From, c.To)
	}
	d.Click(axis+10, strip)
	if c.From != 0 || c.To != 0 {
		t.Errorf("a click without a drag leaves the window at %d to %d, want everything", c.From, c.To)
	}
	d.Down(98, strip)
	d.Resize(50, 30)
	d.Move(49, strip)
	d.Move(0, strip)
	d.Up(0, strip)
	if c.From < 0 || c.To > 40 {
		t.Errorf("a drag held across a resize left the window at %d to %d", c.From, c.To)
	}
	t.Logf("brush zoomed, 100x30:\n%s", zoomed)
}

func TestWindowSetByTheCallerOutsideTheLabelsPanics(t *testing.T) {
	for _, w := range [][2]int{{-2, 30}, {30, 61}, {40, 40}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("From %d To %d over 60 labels did not panic", w[0], w[1])
				}
			}()
			c := &Chart{Kind: Area, Labels: make([]string, 60), Series: []Series{{Values: make([]float64, 60)}}, From: w[0], To: w[1]}
			c.model()
		}()
	}
}
