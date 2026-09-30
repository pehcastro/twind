package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/theme"
	"github.com/twind-dev/twind/twi/ui"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	scheme := flag.String("scheme", "light", "zinc scheme, light or dark")
	name := flag.String("page", "wave1", "wave1, tables, breadcrumbs, pagination, items, button-groups, fields or form")
	focus := flag.String("focus", "", "on the form page, the control focused at start: email, textarea, checkbox, radio, toggles or otp")
	flag.Parse()
	sheet, err := Styles()
	if err != nil {
		fail(err)
	}
	var zinc theme.Theme
	for _, t := range theme.Builtin() {
		if t.Name == "zinc" && (t.Scheme == theme.Dark) == (*scheme == "dark") {
			zinc = t
		}
	}
	rt := twi.New(twi.Fullscreen(), twi.Styles(sheet), twi.Theme(zinc))
	body, ok := page(rt, *name, *focus)
	if !ok {
		fail(fmt.Errorf("no page %q", *name))
	}
	quit := twi.OnKey(func(k input.KeyEvent) {
		if !k.Release && k.Key == input.KeyRune && k.Rune == 'q' {
			rt.Quit()
		}
	})
	if err := rt.Run(func() twi.Node { return screen(quit, body()) }); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func text(class, s string) twi.Node { return el(class, twi.Text(s)) }

func section(title string, children ...twi.NodeOption) twi.Node {
	return el("flex flex-col gap-1", append([]twi.NodeOption{text("text-muted-foreground", title)}, children...)...)
}

func row(children ...twi.NodeOption) twi.Node {
	return el("flex flex-row items-center gap-2", children...)
}

func screen(quit twi.NodeOption, body twi.Node) twi.Node {
	return el("flex flex-row h-full gap-4 px-3 py-1 bg-background text-foreground", quit, body)
}

func page(rt *twi.Runtime, name, focus string) (func() twi.Node, bool) {
	var n twi.Node
	switch name {
	case "form":
		return form(rt, focus), true
	case "wave1":
		n = wave1()
	case "tables":
		n = tables()
	case "breadcrumbs":
		n = breadcrumbs()
	case "pagination":
		n = pagination()
	case "items":
		n = items()
	case "button-groups":
		n = buttonGroups()
	case "fields":
		n = fields()
	default:
		return nil, false
	}
	return func() twi.Node { return n }, true
}

func label(s string) twi.Node { return twi.Text(s) }

func wave1() twi.Node {
	return el("flex flex-row grow gap-4",
		el("flex flex-col gap-1 w-46",
			section("Button, variants",
				row(ui.Button(ui.Default, ui.SizeDefault, label("Default")), ui.Button(ui.Destructive, ui.SizeDefault, label("Destructive")), ui.Button(ui.Outline, ui.SizeDefault, label("Outline"))),
				row(ui.Button(ui.Secondary, ui.SizeDefault, label("Secondary")), ui.Button(ui.Ghost, ui.SizeDefault, label("Ghost")), ui.Button(ui.Link, ui.SizeDefault, label("Link"))),
			),
			section("Button, sizes",
				row(ui.Button(ui.Outline, ui.SizeXS, label("xs")), ui.Button(ui.Outline, ui.SizeSM, label("sm")), ui.Button(ui.Outline, ui.SizeDefault, label("default")), ui.Button(ui.Outline, ui.SizeLG, label("lg")), ui.Button(ui.Outline, ui.SizeIcon, label("+"))),
			),
			section("Badge",
				row(ui.Badge(ui.Default, label("Default")), ui.Badge(ui.Secondary, label("Secondary")), ui.Badge(ui.Destructive, label("Destructive"))),
				row(ui.Badge(ui.Outline, label("Outline")), ui.Badge(ui.Ghost, label("Ghost")), ui.Badge(ui.Link, label("Link"))),
			),
			section("Kbd and Label",
				row(ui.KbdGroup(ui.Kbd(label("Ctrl")), label("+"), ui.Kbd(label("K"))), ui.Kbd(label("⌘")), ui.Label(label("Accept terms"))),
			),
			section("Separator",
				el("flex flex-col",
					text("font-medium", "Radix Primitives"),
					text("text-muted-foreground", "An open-source UI component library."),
					ui.Separator(ui.Horizontal),
					row(label("Blog"), ui.Separator(ui.Vertical), label("Docs"), ui.Separator(ui.Vertical), label("Source")),
				),
			),
			section("Avatar",
				row(ui.Avatar(ui.SizeSM, ui.AvatarFallback(label("CN"))), ui.Avatar(ui.SizeDefault, ui.AvatarFallback(label("CN"))), ui.Avatar(ui.SizeLG, ui.AvatarFallback(label("LR")))),
			),
		),
		el("flex flex-col gap-1 w-50",
			section("Progress",
				ui.Progress(0), ui.Progress(33), ui.Progress(66), ui.Progress(100),
			),
			section("Skeleton",
				row(ui.Skeleton(twi.Class("h-3 w-6 rounded-full")), el("flex flex-col gap-1 grow", ui.Skeleton(twi.Class("h-1 w-36")), ui.Skeleton(twi.Class("h-1 w-28")))),
			),
			section("Alert",
				ui.Alert(ui.Default, ui.AlertTitle(label("Success! Your changes have been saved")), ui.AlertDescription(label("This is an alert with icon, title and description."))),
				ui.Alert(ui.Destructive, ui.AlertTitle(label("Unable to process your payment.")), ui.AlertDescription(label("Please verify your billing information and try again."))),
			),
		),
		el("flex flex-col gap-1 grow",
			section("Card",
				ui.Card(
					ui.CardHeader(ui.CardTitle(label("Login to your account")), ui.CardDescription(label("Enter your email below to login")), ui.CardAction(ui.Button(ui.Link, ui.SizeDefault, label("Sign Up")))),
					ui.CardContent(el("flex flex-col gap-1", ui.Label(label("Email")), text("text-muted-foreground", "m@example.com"))),
					ui.CardFooter(el("flex flex-col grow gap-1", ui.Button(ui.Default, ui.SizeDefault, label("Login")), ui.Button(ui.Outline, ui.SizeDefault, label("Login with Google")))),
				),
			),
			section("Empty",
				ui.Empty(twi.Class("border border-dashed"),
					ui.EmptyHeader(ui.EmptyMedia(ui.Icon, label("▣")), ui.EmptyTitle(label("No Projects Yet")), ui.EmptyDescription(label("You haven't created any projects yet. Get started by creating your first project."))),
					ui.EmptyContent(row(ui.Button(ui.Default, ui.SizeDefault, label("Create Project")), ui.Button(ui.Outline, ui.SizeDefault, label("Import Project")))),
				),
			),
		),
	)
}

func tables() twi.Node {
	invoice := twi.Class("flex-none w-12")
	right := twi.Class("text-right")
	var body []twi.NodeOption
	for _, v := range [][4]string{
		{"INV001", "Paid", "Credit Card", "$250.00"},
		{"INV002", "Pending", "PayPal", "$150.00"},
		{"INV003", "Unpaid", "Bank Transfer", "$350.00"},
		{"INV004", "Paid", "Credit Card", "$450.00"},
		{"INV005", "Paid", "PayPal", "$550.00"},
		{"INV006", "Pending", "Bank Transfer", "$200.00"},
		{"INV007", "Unpaid", "Credit Card", "$300.00"},
	} {
		body = append(body, ui.TableRow(ui.TableCell(invoice, twi.Class("font-medium"), label(v[0])), ui.TableCell(label(v[1])), ui.TableCell(label(v[2])), ui.TableCell(right, label(v[3]))))
	}
	return section("Table",
		el("w-90", ui.Table(
			ui.TableHeader(ui.TableRow(ui.TableHead(invoice, label("Invoice")), ui.TableHead(label("Status")), ui.TableHead(label("Method")), ui.TableHead(right, label("Amount")))),
			ui.TableBody(body...),
			ui.TableFooter(ui.TableRow(ui.TableCell(label("Total")), ui.TableCell(right, label("$2,500.00")))),
			ui.TableCaption(label("A list of your recent invoices.")),
		)),
	)
}

func breadcrumbs() twi.Node {
	return el("flex flex-col gap-2",
		section("Breadcrumb",
			ui.Breadcrumb(ui.BreadcrumbList(
				ui.BreadcrumbItem(ui.BreadcrumbLink(label("Home"))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbEllipsis()), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbLink(label("Components"))), ui.BreadcrumbSeparator(),
				ui.BreadcrumbItem(ui.BreadcrumbPage(label("Breadcrumb"))),
			)),
		),
		section("Custom separator",
			ui.Breadcrumb(ui.BreadcrumbList(
				ui.BreadcrumbItem(ui.BreadcrumbLink(label("Home"))), ui.BreadcrumbSeparator(label("/")),
				ui.BreadcrumbItem(ui.BreadcrumbLink(label("Components"))), ui.BreadcrumbSeparator(label("/")),
				ui.BreadcrumbItem(ui.BreadcrumbPage(label("Breadcrumb"))),
			)),
		),
	)
}

func pagination() twi.Node {
	return section("Pagination",
		el("w-60", ui.Pagination(ui.PaginationContent(
			ui.PaginationItem(ui.PaginationPrevious()),
			ui.PaginationItem(ui.PaginationLink(false, label("1"))),
			ui.PaginationItem(ui.PaginationLink(true, label("2"))),
			ui.PaginationItem(ui.PaginationLink(false, label("3"))),
			ui.PaginationItem(ui.PaginationEllipsis()),
			ui.PaginationItem(ui.PaginationNext()),
		))),
	)
}

func items() twi.Node {
	open := ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, label("Open")))
	var people []twi.NodeOption
	for i, p := range [][2]string{{"shadcn", "shadcn@vercel.com"}, {"maxleiter", "maxleiter@vercel.com"}, {"evilrabbit", "evilrabbit@vercel.com"}} {
		if i > 0 {
			people = append(people, ui.ItemSeparator())
		}
		people = append(people, ui.Item(ui.Default, ui.SizeDefault,
			ui.ItemMedia(ui.Default, ui.Avatar(ui.SizeSM, ui.AvatarFallback(label(p[0][:1])))),
			ui.ItemContent(ui.ItemTitle(label(p[0])), ui.ItemDescription(label(p[1]))),
			ui.ItemActions(ui.Button(ui.Ghost, ui.SizeIcon, label("+"))),
		))
	}
	return el("flex flex-row grow gap-4",
		el("flex flex-col gap-1 w-56",
			section("Item",
				ui.Item(ui.Outline, ui.SizeDefault,
					ui.ItemContent(ui.ItemTitle(label("Basic Item")), ui.ItemDescription(label("A simple item with title and description."))),
					ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, label("Action"))),
				),
				ui.Item(ui.Outline, ui.SizeSM,
					ui.ItemMedia(ui.Default, label("✓")),
					ui.ItemContent(ui.ItemTitle(label("Your profile has been verified."))),
					ui.ItemActions(label("›")),
				),
			),
			section("Icon media",
				ui.Item(ui.Outline, ui.SizeDefault,
					ui.ItemMedia(ui.Icon, label("!")),
					ui.ItemContent(ui.ItemTitle(label("Security Alert")), ui.ItemDescription(label("New login detected from unknown device."))),
					ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, label("Review"))),
				),
			),
		),
		el("flex flex-col gap-1 w-56",
			section("Variants",
				ui.Item(ui.Default, ui.SizeDefault, ui.ItemContent(ui.ItemTitle(label("Default Variant")), ui.ItemDescription(label("Standard styling with subtle background and borders."))), open),
				ui.Item(ui.Outline, ui.SizeDefault, ui.ItemContent(ui.ItemTitle(label("Outline Variant")), ui.ItemDescription(label("Outlined style with clear borders and transparent background."))), open),
				ui.Item(ui.Muted, ui.SizeDefault, ui.ItemContent(ui.ItemTitle(label("Muted Variant")), ui.ItemDescription(label("Subdued appearance with muted colors for secondary content."))), open),
			),
		),
		el("flex flex-col gap-1 w-50",
			section("Group", ui.ItemGroup(people...)),
		),
	)
}

func buttonGroups() twi.Node {
	outlined := func(s string) twi.Node { return ui.Button(ui.Outline, ui.SizeDefault, label(s)) }
	return el("flex flex-col gap-2",
		section("Button group",
			ui.ButtonGroup(ui.Horizontal, twi.Class("gap-2"),
				ui.ButtonGroup(ui.Horizontal, ui.Button(ui.Outline, ui.SizeIcon, label("←"))),
				ui.ButtonGroup(ui.Horizontal, outlined("Archive"), outlined("Report")),
				ui.ButtonGroup(ui.Horizontal, outlined("Snooze"), ui.Button(ui.Outline, ui.SizeIcon, label("…"))),
			),
		),
		section("Orientation",
			row(ui.ButtonGroup(ui.Vertical, ui.Button(ui.Outline, ui.SizeIcon, label("+")), ui.Button(ui.Outline, ui.SizeIcon, label("-")))),
		),
		section("Separator",
			row(ui.ButtonGroup(ui.Horizontal, ui.Button(ui.Secondary, ui.SizeSM, label("Copy")), ui.ButtonGroupSeparator(ui.Vertical), ui.Button(ui.Secondary, ui.SizeSM, label("Paste")))),
		),
		section("Text",
			row(ui.ButtonGroup(ui.Horizontal, ui.ButtonGroupText(label("https://")), outlined("example.com"), ui.Button(ui.Default, ui.SizeDefault, label("Go")))),
		),
	)
}

func fields() twi.Node {
	input := func(placeholder string) twi.Node {
		return text("h-1 px-1 rounded-md text-muted-foreground shadow-[0_0_0_1px_var(--color-input)] dark:bg-input/30", placeholder)
	}
	return el("flex flex-row grow gap-6",
		el("flex flex-col w-56",
			ui.FieldGroup(
				ui.FieldSet(
					ui.FieldLegend(label("Payment Method")),
					ui.FieldDescription(label("All transactions are secure and encrypted")),
					ui.FieldGroup(
						ui.Field(ui.Vertical, ui.FieldLabel(label("Name on Card")), input("Evil Rabbit")),
						ui.Field(ui.Vertical, ui.FieldLabel(label("Card Number")), input("1234 5678 9012 3456"), ui.FieldDescription(label("Enter your 16-digit card number"))),
						el("flex flex-row gap-2",
							ui.Field(ui.Vertical, ui.FieldLabel(label("Month")), input("MM")),
							ui.Field(ui.Vertical, ui.FieldLabel(label("Year")), input("YYYY")),
							ui.Field(ui.Vertical, ui.FieldLabel(label("CVV")), input("123")),
						),
					),
				),
				ui.FieldSeparator(),
				ui.FieldSet(
					ui.FieldLegend(label("Billing Address")),
					ui.FieldDescription(label("The billing address associated with your payment method")),
					ui.Field(ui.Horizontal, text("flex w-2 justify-center rounded-sm bg-primary text-primary-foreground", "✓"), ui.FieldLabel(label("Same as shipping address"))),
				),
				ui.Field(ui.Horizontal, ui.Button(ui.Default, ui.SizeDefault, label("Submit")), ui.Button(ui.Outline, ui.SizeDefault, label("Cancel"))),
			),
		),
		el("flex flex-col gap-2 w-50",
			section("Error",
				ui.Field(ui.Vertical, ui.FieldLabel(label("Email")), input("m@example.com"), ui.FieldError(label("Enter a valid email address."))),
			),
			section("Separator with a label",
				ui.FieldSeparator(label("Or continue with")),
			),
		),
	)
}
