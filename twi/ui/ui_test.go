package ui

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen -o twir_gen_test.go -func styles

func zinc(t *testing.T, scheme theme.Scheme) theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if th.Name == "zinc" && th.Scheme == scheme {
			return th
		}
	}
	t.Fatal("no zinc theme")
	return theme.Theme{}
}

func computed(t *testing.T, th theme.Theme, n twi.Node, path ...int) style.ComputedStyle {
	t.Helper()
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	sheet = sheet.WithTheme(&th)
	v := reflect.ValueOf(n).FieldByName("tree")
	var s style.ComputedStyle
	for depth := 0; ; depth++ {
		list := v.FieldByName("Classes")
		classes := make([]string, list.Len())
		for i := range classes {
			classes[i] = list.Index(i).String()
		}
		s = sheet.Compute(s, classes)
		if depth == len(path) {
			return s
		}
		v = v.FieldByName("Children").Index(path[depth])
	}
}

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", "twir_gen_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Error("twir_gen_test.go is stale against the classes in twi/ui: run go generate")
	}
}

func TestParts(t *testing.T) {
	light, dark := zinc(t, theme.Light), zinc(t, theme.Dark)
	token := func(th theme.Theme, tok theme.Token, alpha uint8) color.Color {
		c := th.Tokens[tok]
		c.RGBA.A = alpha
		return c
	}
	ring := func(s style.ComputedStyle, c color.Color) bool {
		return len(s.Shadows) == 1 && len(s.InsetShadows) == 0 && s.Shadows[0].Spread == 1 && s.Shadows[0].Blur == 0 && s.Shadows[0].X == 0 && s.Shadows[0].Y == 0 && s.Shadows[0].Color == c
	}
	white := color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255}}
	cells := func(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }
	percent := func(n float64) style.Length { return style.Length{Unit: style.Percent, Value: n} }
	card := Card(
		CardHeader(CardTitle(twi.Text("Title")), CardDescription(twi.Text("Description")), CardAction(twi.Text("Action"))),
		CardContent(twi.Text("Content")),
		CardFooter(twi.Text("Footer")),
	)
	for _, c := range []struct {
		name string
		th   theme.Theme
		node twi.Node
		path []int
		want func(style.ComputedStyle) bool
	}{
		{"button default: bg-primary text-primary-foreground rounded-md", light, Button(Default, SizeDefault, twi.Text("Button")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Color == light.Tokens[theme.PrimaryForeground] && s.Radius == style.RadiusMd && s.Padding.Left == cells(2)
		}},
		{"button destructive: bg-destructive text-white", light, Button(Destructive, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Destructive] && s.Color == white
		}},
		{"button destructive, dark: bg-destructive/60", dark, Button(Destructive, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.Background == token(dark, theme.Destructive, 153)
		}},
		{"button outline: a one pixel ring outside the text cells, no layout border", light, Button(Outline, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Background] && ring(s, light.Tokens[theme.Border]) && s.BorderWidth == style.Edges{}
		}},
		{"button outline, dark: bg-input/30 and the dark border ring", dark, Button(Outline, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.Background == token(dark, theme.Input, uint8(math.Round(float64(dark.Tokens[theme.Input].RGBA.A)*0.3))) && ring(s, dark.Tokens[theme.Border])
		}},
		{"button secondary", light, Button(Secondary, SizeSM), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Secondary] && s.Color == light.Tokens[theme.SecondaryForeground]
		}},
		{"button ghost: no background", light, Button(Ghost, SizeXS), nil, func(s style.ComputedStyle) bool {
			return s.Background.Kind == color.Unset && s.Padding.Left == cells(1)
		}},
		{"button link: text-primary", light, Button(Link, SizeLG), nil, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Primary] && s.Padding.Left == cells(3)
		}},
		{"button icon: three cells wide", light, Button(Default, SizeIcon), nil, func(s style.ComputedStyle) bool {
			return s.Width == cells(3)
		}},
		{"card: bg-card rounded-xl border shadow-sm", light, card, nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Card] && s.Radius == style.RadiusLg && s.BorderWidth.Top == cells(1) && s.BorderWidth.Left == cells(1) && len(s.Shadows) == 2
		}},
		{"card title: font-semibold", light, card, []int{0, 0}, func(s style.ComputedStyle) bool { return s.Bold }},
		{"card description: text-muted-foreground", light, card, []int{0, 1}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"card action: top right of the header", light, card, []int{0, 2}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionAbsolute && s.Inset.Top == cells(0) && s.Inset.Right == cells(2)
		}},
		{"card footer: flex row", light, card, []int{2}, func(s style.ComputedStyle) bool {
			return s.Display == style.DisplayFlex && s.Direction == style.Row && s.Padding.Left == cells(2)
		}},
		{"badge default: bg-primary rounded-full", light, Badge(Default, twi.Text("Badge")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Radius == style.RadiusFull && s.Padding.Left == cells(1)
		}},
		{"badge outline: a one pixel ring", light, Badge(Outline), nil, func(s style.ComputedStyle) bool {
			return ring(s, light.Tokens[theme.Border]) && s.Color == light.Tokens[theme.Foreground] && s.BorderWidth == style.Edges{}
		}},
		{"separator horizontal: a top hairline across", light, Separator(Horizontal), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Top == cells(1) && s.BorderWidth.Left == cells(0) && s.Width == percent(100) && s.BorderColor == light.Tokens[theme.Border]
		}},
		{"separator vertical: a left hairline down", light, Separator(Vertical), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Left == cells(1) && s.BorderWidth.Top == cells(0) && s.AlignSelf == style.AlignStretch
		}},
		{"kbd: bg-muted text-muted-foreground rounded-sm", light, Kbd(twi.Text("K")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Muted] && s.Color == light.Tokens[theme.MutedForeground] && s.Radius == style.RadiusSm && s.UserSelect == style.SelectNone
		}},
		{"kbd group: a row", light, KbdGroup(Kbd(twi.Text("Ctrl")), Kbd(twi.Text("K"))), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.ColumnGap == cells(1)
		}},
		{"label: select-none flex", light, Label(twi.Text("Email")), nil, func(s style.ComputedStyle) bool {
			return s.Display == style.DisplayFlex && s.UserSelect == style.SelectNone
		}},
		{"skeleton: bg-accent rounded-md", light, Skeleton(), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent] && s.Radius == style.RadiusMd
		}},
		{"alert default: bg-card border rounded-lg", light, Alert(Default, AlertTitle(twi.Text("Heads up!"))), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Card] && s.Color == light.Tokens[theme.CardForeground] && s.Radius == style.RadiusLg && s.BorderWidth.Top == cells(1)
		}},
		{"alert destructive: the title takes text-destructive", light, Alert(Destructive, AlertTitle(twi.Text("Error")), AlertDescription(twi.Text("Try again."))), []int{0}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Destructive]
		}},
		{"alert description: text-muted-foreground", light, Alert(Default, AlertDescription(twi.Text("Body"))), []int{0}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"empty: centred column", light, Empty(EmptyHeader(EmptyMedia(Icon, twi.Text("?")), EmptyTitle(twi.Text("No projects")), EmptyDescription(twi.Text("None yet."))), EmptyContent(Button(Default, SizeSM, twi.Text("Create")))), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column && s.AlignItems == style.AlignCenter && s.Justify == style.JustifyCenter && s.TextAlign == style.TextCenter
		}},
		{"empty media icon: bg-muted rounded-lg", light, EmptyMedia(Icon, twi.Text("?")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Muted] && s.Radius == style.RadiusLg && s.Height == cells(3)
		}},
		{"empty description: text-muted-foreground", light, EmptyDescription(twi.Text("None yet.")), nil, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"progress track: bg-primary/20 rounded-full", light, Progress(37), nil, func(s style.ComputedStyle) bool {
			return s.Background == token(light, theme.Primary, 51) && s.Radius == style.RadiusFull && s.OverflowX == style.OverflowHidden
		}},
		{"progress 37: the indicator is 37% wide", light, Progress(37), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Primary] && s.Width == percent(37)
		}},
		{"progress below 0 clamps to 0", light, Progress(-5), []int{0}, func(s style.ComputedStyle) bool { return s.Width == percent(0) }},
		{"progress above 100 clamps to 100", light, Progress(140), []int{0}, func(s style.ComputedStyle) bool { return s.Width == percent(100) }},
		{"avatar: a circle of six by three cells", light, Avatar(SizeDefault, AvatarFallback(twi.Text("CN"))), nil, func(s style.ComputedStyle) bool {
			return s.Radius == style.RadiusFull && s.Width == cells(6) && s.Height == cells(3)
		}},
		{"avatar fallback: bg-muted text-muted-foreground", light, Avatar(SizeSM, AvatarFallback(twi.Text("CN"))), []int{0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Muted] && s.Color == light.Tokens[theme.MutedForeground] && s.AlignItems == style.AlignCenter
		}},
	} {
		if s := computed(t, c.th, c.node, c.path...); !c.want(s) {
			t.Errorf("%s: computed %+v", c.name, s)
		}
	}
}

func TestUnknownVariantPanics(t *testing.T) {
	for name, build := range map[string]func(){
		"alert outline":    func() { Alert(Outline) },
		"badge icon":       func() { Badge(Icon) },
		"button size icon": func() { Button(Icon, SizeDefault) },
		"avatar size icon": func() { Avatar(SizeIcon) },
		"empty media link": func() { EmptyMedia(Link) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", name)
				}
			}()
			build()
		}()
	}
}

func TestRendersEveryComponent(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	every := twi.Element(twi.Class("flex flex-col gap-1 w-60"),
		twi.Element(twi.Class("flex flex-row gap-1"), Button(Default, SizeDefault, twi.Text("Save")), Badge(Secondary, twi.Text("New")), KbdGroup(Kbd(twi.Text("Ctrl")), Kbd(twi.Text("K"))), Label(twi.Text("Email"))),
		Separator(Horizontal),
		Card(CardHeader(CardTitle(twi.Text("Card title")), CardDescription(twi.Text("Card description")), CardAction(twi.Text("Act"))), CardContent(twi.Text("Card content")), CardFooter(twi.Text("Card footer"))),
		Alert(Destructive, AlertTitle(twi.Text("Alert title")), AlertDescription(twi.Text("Alert description"))),
		Empty(EmptyHeader(EmptyMedia(Icon, twi.Text("?")), EmptyTitle(twi.Text("Empty title")), EmptyDescription(twi.Text("Empty description"))), EmptyContent(Button(Outline, SizeSM, twi.Text("Create")))),
		Progress(50),
		twi.Element(twi.Class("flex flex-row gap-1"), Avatar(SizeDefault, AvatarFallback(twi.Text("CN"))), Skeleton(twi.Class("h-1 w-10"))),
	)
	labels := []string{"Save", "New", "Ctrl", "Email", "Card title", "Card description", "Act", "Card content", "Card footer", "Alert title", "Alert description", "Empty title", "Empty description", "Create", "CN"}
	static := twi.RenderString(every, twi.Styles(sheet), twi.Theme(zinc(t, theme.Light)), twi.Width(60), twi.ColorProfile(color.None))
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		return func() twi.Node { return every }
	}, drive.Size(60, 40), drive.Styles(sheet))
	driven := d.Frame().Text()
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("driven, zinc dark, 60x40:\n%s", driven)
	for _, l := range labels {
		if !strings.Contains(static, l) || !strings.Contains(driven, l) {
			t.Errorf("%q missing: in RenderString %v, in the driven frame %v", l, strings.Contains(static, l), strings.Contains(driven, l))
		}
	}
}

func TestRendersOneRowControls(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []Variant{Default, Destructive, Outline, Secondary, Ghost, Link} {
		out := twi.RenderString(twi.Element(twi.Class("flex flex-row"), Button(v, SizeDefault, twi.Text("Button")), Badge(Default, twi.Text("Badge"))),
			twi.Styles(sheet), twi.Theme(zinc(t, theme.Light)), twi.Width(30), twi.ColorProfile(color.TrueColor))
		if rows := strings.Count(out, "\n"); rows != 1 || !strings.Contains(out, "Button") {
			t.Errorf("button variant %d renders %d rows, want one row holding its label:\n%s", v, rows, out)
		}
	}
}
