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

func token(th theme.Theme, tok theme.Token, alpha uint8) color.Color {
	c := th.Tokens[tok]
	c.RGBA.A = alpha
	return c
}

func ring(s style.ComputedStyle, c color.Color) bool {
	return len(s.Shadows) == 1 && len(s.InsetShadows) == 0 && s.Shadows[0].Spread == 1 && s.Shadows[0].Blur == 0 && s.Shadows[0].X == 0 && s.Shadows[0].Y == 0 && s.Shadows[0].Color == c
}

func cells(n float64) style.Length { return style.Length{Unit: style.Cells, Value: n} }

func percent(n float64) style.Length { return style.Length{Unit: style.Percent, Value: n} }

type partCase struct {
	name string
	th   theme.Theme
	node twi.Node
	path []int
	want func(style.ComputedStyle) bool
}

func checkParts(t *testing.T, cases []partCase) {
	t.Helper()
	for _, c := range cases {
		if s := computed(t, c.th, c.node, c.path...); !c.want(s) {
			t.Errorf("%s: computed %+v", c.name, s)
		}
	}
}

func TestParts(t *testing.T) {
	light, dark := zinc(t, theme.Light), zinc(t, theme.Dark)
	white := color.Color{Kind: color.Literal, RGBA: color.RGBA{R: 255, G: 255, B: 255, A: 255}}
	card := Card(
		CardHeader(CardTitle(twi.Text("Title")), CardDescription(twi.Text("Description")), CardAction(twi.Text("Action"))),
		CardContent(twi.Text("Content")),
		CardFooter(twi.Text("Footer")),
	)
	checkParts(t, []partCase{
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
	})
}

func TestPartsWave1b(t *testing.T) {
	light, dark := zinc(t, theme.Light), zinc(t, theme.Dark)
	text := twi.Text
	invoices := Table(
		TableHeader(TableRow(TableHead(twi.Class("flex-none w-12"), text("Invoice")), TableHead(text("Amount")))),
		TableBody(TableRow(TableCell(text("INV001")), TableCell(text("$250.00")))),
		TableFooter(TableRow(TableCell(text("Total")))),
		TableCaption(text("A list of your recent invoices.")),
	)
	crumbs := Breadcrumb(BreadcrumbList(
		BreadcrumbItem(BreadcrumbLink(text("Home"))),
		BreadcrumbSeparator(),
		BreadcrumbItem(BreadcrumbEllipsis()),
		BreadcrumbSeparator(text("/")),
		BreadcrumbItem(BreadcrumbPage(text("Breadcrumb"))),
	))
	pages := Pagination(PaginationContent(
		PaginationItem(PaginationPrevious()),
		PaginationItem(PaginationLink(false, text("1"))),
		PaginationItem(PaginationLink(true, text("2"))),
		PaginationItem(PaginationEllipsis()),
		PaginationItem(PaginationNext()),
	))
	item := Item(Outline, SizeDefault,
		ItemMedia(Icon, text("◆")),
		ItemContent(ItemTitle(text("Basic Item")), ItemDescription(text("A simple item with title and description."))),
		ItemActions(Button(Outline, SizeSM, text("Action"))),
	)
	field := FieldSet(
		FieldLegend(text("Payment Method")),
		FieldGroup(
			Field(Vertical, FieldLabel(text("Name on Card")), FieldDescription(text("As printed.")), FieldError(text("Required."))),
			FieldSeparator(text("Or")),
			Field(Horizontal, FieldLabel(text("Same as shipping"))),
		),
	)
	checkParts(t, []partCase{
		{"Table: a w-full box that scrolls sideways", light, invoices, nil, func(s style.ComputedStyle) bool {
			return s.Width == percent(100) && s.OverflowX == style.OverflowAuto && s.Position == style.PositionRelative
		}},
		{"Table: a full-width column of sections", light, invoices, []int{0}, func(s style.ComputedStyle) bool {
			return s.Display == style.DisplayFlex && s.Direction == style.Column && s.Width == percent(100)
		}},
		{"TableHeader: a column of rows", light, invoices, []int{0, 0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column
		}},
		{"TableRow: a row with one bottom line, no ring", light, invoices, []int{0, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.BorderWidth == style.Edges{Top: cells(0), Right: cells(0), Bottom: cells(1), Left: cells(0)}
		}},
		{"TableHead: font-medium text-foreground px-2, an equal share of the row", light, invoices, []int{0, 0, 0, 1}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Foreground] && s.Padding.Left == cells(1) && s.Grow == 1 && s.Basis == percent(0) && s.TextAlign == style.TextLeft
		}},
		{"TableHead: a caller's flex-none w-12 fixes the column", light, invoices, []int{0, 0, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Grow == 0 && s.Shrink == 0 && s.Width == cells(12)
		}},
		{"TableCell: p-2, an equal share of the row", light, invoices, []int{0, 1, 0, 1}, func(s style.ComputedStyle) bool {
			return s.Padding.Left == cells(1) && s.Padding.Right == cells(1) && s.Grow == 1 && s.Basis == percent(0)
		}},
		{"TableFooter: bg-muted/50, no second line on top", light, invoices, []int{0, 2}, func(s style.ComputedStyle) bool {
			return s.Background == token(light, theme.Muted, 128) && s.BorderWidth.Top == cells(0) && s.Direction == style.Column
		}},
		{"TableCaption: mt-4 text-muted-foreground", light, invoices, []int{0, 3}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground] && s.Margin.Top == cells(1) && s.TextAlign == style.TextCenter
		}},
		{"breadcrumb list: a muted row, a six pixel gap as one cell", light, crumbs, []int{0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.ColumnGap == cells(1) && s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"breadcrumb item: an inline row", light, crumbs, []int{0, 0}, func(s style.ComputedStyle) bool {
			return s.Display == style.DisplayFlex && s.Direction == style.Row && s.ColumnGap == cells(1)
		}},
		{"breadcrumb ellipsis: a centred three-cell box", light, crumbs, []int{0, 2, 0}, func(s style.ComputedStyle) bool {
			return s.Width == cells(3) && s.Justify == style.JustifyCenter && s.AlignItems == style.AlignCenter
		}},
		{"breadcrumb page: text-foreground", light, crumbs, []int{0, 4, 0}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Foreground] && !s.Bold
		}},
		{"pagination: centred across the width", light, pages, nil, func(s style.ComputedStyle) bool {
			return s.Display == style.DisplayFlex && s.Direction == style.Row && s.Justify == style.JustifyCenter && s.Width == percent(100)
		}},
		{"pagination content: a row, gap-1", light, pages, []int{0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.ColumnGap == cells(1)
		}},
		{"pagination link: ghost, icon size", light, pages, []int{0, 1, 0}, func(s style.ComputedStyle) bool {
			return s.Background.Kind == color.Unset && len(s.Shadows) == 0 && s.Width == cells(3) && s.Radius == style.RadiusMd
		}},
		{"pagination link active: outline, icon size", light, pages, []int{0, 2, 0}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Background] && ring(s, light.Tokens[theme.Border]) && s.Width == cells(3)
		}},
		{"pagination previous: ghost, ten pixels of padding as one cell", light, pages, []int{0, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Background.Kind == color.Unset && s.Padding.Left == cells(1) && s.ColumnGap == cells(1)
		}},
		{"pagination ellipsis: a centred three-cell box", light, pages, []int{0, 3, 0}, func(s style.ComputedStyle) bool {
			return s.Width == cells(3) && s.Justify == style.JustifyCenter
		}},
		{"item outline: rounded-md border-border, p-4 as the border row and one cell", light, item, nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.Radius == style.RadiusMd && s.BorderWidth.Top == cells(1) && s.BorderColor == light.Tokens[theme.Border] && s.Padding.Left == cells(1) && s.ColumnGap == cells(2)
		}},
		{"item default: a transparent border of the same size", light, Item(Default, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Top == cells(1) && s.BorderColor.RGBA.A == 0 && s.Background.RGBA.A == 0
		}},
		{"item muted: bg-muted/50", light, Item(Muted, SizeDefault), nil, func(s style.ComputedStyle) bool {
			return s.Background == token(light, theme.Muted, 128) && s.BorderColor.RGBA.A == 0
		}},
		{"item sm: a ten pixel gap as one cell", light, Item(Default, SizeSM), nil, func(s style.ComputedStyle) bool { return s.ColumnGap == cells(1) }},
		{"item media icon: a one-row muted tile with a ring, no layout border", light, item, []int{0}, func(s style.ComputedStyle) bool {
			return s.Height == cells(1) && s.Width == cells(3) && s.Background == light.Tokens[theme.Muted] && s.Radius == style.RadiusSm && ring(s, light.Tokens[theme.Border]) && s.BorderWidth == style.Edges{} && s.Shrink == 0
		}},
		{"item media image: size-10 overflow-hidden rounded-sm", light, ItemMedia(Image), nil, func(s style.ComputedStyle) bool {
			return s.Height == cells(3) && s.Width == cells(5) && s.OverflowX == style.OverflowHidden && s.Radius == style.RadiusSm
		}},
		{"item content: flex-1 column", light, item, []int{1}, func(s style.ComputedStyle) bool {
			return s.Grow == 1 && s.Direction == style.Column
		}},
		{"item title: font-medium row", light, item, []int{1, 0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter
		}},
		{"item description: text-muted-foreground", light, item, []int{1, 1}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"item actions: a row, gap-2", light, item, []int{2}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.ColumnGap == cells(1)
		}},
		{"item group: a column", light, ItemGroup(item, ItemSeparator(), item), nil, func(s style.ComputedStyle) bool { return s.Direction == style.Column }},
		{"item separator: a horizontal hairline", light, ItemSeparator(), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Top == cells(1) && s.Width == percent(100)
		}},
		{"button group horizontal: a row, no gap, stretched", light, ButtonGroup(Horizontal, Button(Outline, SizeDefault)), nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.ColumnGap == cells(0) && s.AlignItems == style.AlignStretch
		}},
		{"button group vertical: a column", light, ButtonGroup(Vertical), nil, func(s style.ComputedStyle) bool { return s.Direction == style.Column }},
		{"button group separator, dark: bg-input as a vertical hairline", dark, ButtonGroupSeparator(Vertical), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Left == cells(1) && s.BorderWidth.Top == cells(0) && s.BorderColor == dark.Tokens[theme.Input] && s.AlignSelf == style.AlignStretch
		}},
		{"button group separator horizontal: a top hairline", light, ButtonGroupSeparator(Horizontal), nil, func(s style.ComputedStyle) bool {
			return s.BorderWidth.Top == cells(1) && s.BorderWidth.Left == cells(0)
		}},
		{"button group text: bg-muted rounded-md, a ring instead of a layout border", light, ButtonGroupText(text("https://")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Muted] && s.Radius == style.RadiusMd && ring(s, light.Tokens[theme.Border]) && s.BorderWidth == style.Edges{} && s.Padding.Left == cells(2)
		}},
		{"field set: a column, gap-6", light, field, nil, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column && s.RowGap == cells(1)
		}},
		{"field legend: mb-3", light, field, []int{0}, func(s style.ComputedStyle) bool { return s.Margin.Bottom == cells(1) }},
		{"field group: a full-width column, gap-7", light, field, []int{1}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column && s.Width == percent(100) && s.RowGap == cells(1)
		}},
		{"field vertical: label over control", light, field, []int{1, 0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Column && s.Width == percent(100)
		}},
		{"field label: a select-none row", light, field, []int{1, 0, 0}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.UserSelect == style.SelectNone
		}},
		{"field description: text-muted-foreground", light, field, []int{1, 0, 1}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.MutedForeground]
		}},
		{"field error: text-destructive", light, field, []int{1, 0, 2}, func(s style.ComputedStyle) bool {
			return s.Color == light.Tokens[theme.Destructive]
		}},
		{"field separator: one row holding the line and its label", light, field, []int{1, 1}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionRelative && s.Height == cells(1) && s.Justify == style.JustifyCenter
		}},
		{"field separator line: an absolute top hairline across", light, field, []int{1, 1, 0}, func(s style.ComputedStyle) bool {
			return s.Position == style.PositionAbsolute && s.BorderWidth.Top == cells(1) && s.Inset.Left == cells(0) && s.Inset.Right == cells(0)
		}},
		{"field separator label: bg-background over the line", light, field, []int{1, 1, 1}, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Background] && s.Color == light.Tokens[theme.MutedForeground] && s.Position == style.PositionRelative && s.Padding.Left == cells(1)
		}},
		{"field horizontal: label beside control", light, field, []int{1, 2}, func(s style.ComputedStyle) bool {
			return s.Direction == style.Row && s.AlignItems == style.AlignCenter && s.ColumnGap == cells(1)
		}},
	})
}

func TestUnknownVariantPanics(t *testing.T) {
	for name, build := range map[string]func(){
		"alert outline":      func() { Alert(Outline) },
		"badge icon":         func() { Badge(Icon) },
		"button size icon":   func() { Button(Icon, SizeDefault) },
		"avatar size icon":   func() { Avatar(SizeIcon) },
		"empty media link":   func() { EmptyMedia(Link) },
		"item destructive":   func() { Item(Destructive, SizeDefault) },
		"item size lg":       func() { Item(Default, SizeLG) },
		"item media outline": func() { ItemMedia(Outline) },
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
	for name, c := range map[string]struct {
		node twi.Node
		rows int
	}{
		"button group with text":  {ButtonGroup(Horizontal, ButtonGroupText(twi.Text("https://")), Button(Outline, SizeDefault, twi.Text("Go"))), 1},
		"item with a description": {Item(Outline, SizeDefault, ItemMedia(Icon, twi.Text("◆")), ItemContent(ItemTitle(twi.Text("Title")), ItemDescription(twi.Text("Body")))), 4},
	} {
		out := twi.RenderString(c.node, twi.Styles(sheet), twi.Theme(zinc(t, theme.Light)), twi.Width(30), twi.ColorProfile(color.None))
		if rows := strings.Count(out, "\n"); rows != c.rows {
			t.Errorf("%s renders %d rows, want %d:\n%s", name, rows, c.rows, out)
		}
	}
}

func TestRendersWave1b(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	text := twi.Text
	row := func(invoice, status, amount string) twi.Node {
		return TableRow(TableCell(twi.Class("flex-none w-10"), text(invoice)), TableCell(text(status)), TableCell(twi.Class("text-right"), text(amount)))
	}
	every := twi.Element(twi.Class("flex flex-col gap-1 w-60"),
		Table(
			TableHeader(TableRow(TableHead(twi.Class("flex-none w-10"), text("Invoice")), TableHead(text("Status")), TableHead(twi.Class("text-right"), text("Amount")))),
			TableBody(row("INV001", "Paid", "$250.00"), row("INV002", "Pending", "$150.00")),
			TableFooter(TableRow(TableCell(twi.Class("flex-none w-10"), text("Total")), TableCell(), TableCell(twi.Class("text-right"), text("$400.00")))),
			TableCaption(text("A list of your recent invoices.")),
		),
		Breadcrumb(BreadcrumbList(
			BreadcrumbItem(BreadcrumbLink(text("Home"))), BreadcrumbSeparator(),
			BreadcrumbItem(BreadcrumbEllipsis()), BreadcrumbSeparator(text("/")),
			BreadcrumbItem(BreadcrumbPage(text("Breadcrumb"))),
		)),
		Pagination(PaginationContent(
			PaginationItem(PaginationPrevious()), PaginationItem(PaginationLink(false, text("1"))), PaginationItem(PaginationLink(true, text("2"))),
			PaginationItem(PaginationEllipsis()), PaginationItem(PaginationNext()),
		)),
		FieldSeparator(text("Or continue with")),
	)
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		return func() twi.Node { return every }
	}, drive.Size(60, 30), drive.Styles(sheet))
	frame := d.Frame().Text()
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("driven, zinc dark, 60x30:\n%s", frame)
	lines := strings.Split(frame, "\n")
	find := func(s string) (int, int) {
		for i, l := range lines {
			if at := strings.Index(l, s); at >= 0 {
				return i, len([]rune(l[:at]))
			}
		}
		t.Errorf("no %q in the frame", s)
		return -1, -1
	}
	head, statusHead := find("Status")
	_, status := find("Pending")
	_, amountEnd := find("Amount")
	_, valueEnd := find("$150.00")
	total, _ := find("Total")
	if statusHead != status || amountEnd+len("Amount") != valueEnd+len("$150.00") {
		t.Errorf("columns do not line up: Status at %d, Pending at %d; Amount ends at %d, $150.00 at %d", statusHead, status, amountEnd+len("Amount"), valueEnd+len("$150.00"))
	}
	if total-head != 6 {
		t.Errorf("the footer is %d rows below the header, want 6: three text rows, each with one line under it", total-head)
	}
	crumbs, _ := find("Home")
	if l := lines[max(crumbs, 0)]; !strings.Contains(l, "Home ›  …  / Breadcrumb") {
		t.Errorf("breadcrumb reads %q, want the chevron by default, the given separator, and the ellipsis", l)
	}
	pages, _ := find("Previous")
	if l := lines[max(pages, 0)]; !strings.Contains(l, "‹ Previous") || !strings.Contains(l, "Next ›") || !strings.Contains(l, "…") {
		t.Errorf("pagination reads %q", l)
	}
	find("Or continue with")
}
