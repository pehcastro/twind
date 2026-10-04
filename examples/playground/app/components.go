package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

type kit struct {
	dialog, sheet, alert, drawer *ui.Dialog
	palette                      *ui.CommandDialog
	toaster                      *ui.Toaster
	tip, hint                    *ui.Tooltip
	accordion                    *ui.Accordion
	collapsible                  *ui.Collapsible
	command                      *ui.Command
	checks                       [4]*ui.Checkbox
	context                      *ui.ContextMenu
	menu                         *ui.DropdownMenu
	invite                       *ui.DropdownMenuSub
	hover                        *ui.HoverCard
	email, off, bad, site        *ui.Input
	width, height                *ui.Input
	area                         *ui.Textarea
	otp                          *ui.InputOTP
	bar                          *ui.Menubar
	file, edit, view             *ui.MenubarMenu
	native                       *ui.NativeSelect
	popover                      *ui.Popover
	radio                        *ui.RadioGroup
	fruit                        *ui.Select
	slider                       *ui.Slider
	airplane                     *ui.Switch
	tabs                         *ui.Tabs
	toggles                      [3]*ui.Toggle
	align, marks                 *ui.ToggleGroup
	sidebar                      *ui.Sidebar
	spinners                     [4]*ui.Spinner
	lookup, query, url           *ui.Input
	moves                        *moves
	message                      *ui.Textarea
	framework                    *ui.Combobox
	calendar                     *ui.Calendar
	nav                          *ui.NavigationMenu
	started, parts               *ui.NavigationMenuItem
	scroller                     *ui.MessageScroller
	panes, stack                 *ui.Resizable
	carousel                     *ui.Carousel
	signup                       *ui.Form
	handle, mail, secret         *ui.Input
	plan                         *ui.Select
	agree                        *ui.Checkbox
	bio                          *ui.Textarea
	joined                       string
	sent                         int
	chosen, pressed, side        string
	person, panel, href          string
	bookmarks, statusBar         bool
	spinning, wide               bool
	at, progress, goal, row      int
}

func newKit(rt *twi.Runtime, today time.Time) *kit {
	k := &kit{
		dialog: ui.NewDialog(rt), sheet: ui.NewSheet(rt, ui.Right), alert: ui.NewAlertDialog(rt), drawer: ui.NewDrawer(rt, ui.Bottom),
		palette: ui.NewCommandDialog(rt), toaster: ui.NewToaster(rt), tip: ui.NewTooltip(rt), hint: ui.NewTooltip(rt),
		accordion: ui.NewAccordion(rt), collapsible: ui.NewCollapsible(rt), command: ui.NewCommand(rt),
		context: ui.NewContextMenu(rt), menu: ui.NewDropdownMenu(rt), hover: ui.NewHoverCard(rt),
		email: ui.NewInput(rt), off: ui.NewInput(rt), bad: ui.NewInput(rt), site: ui.NewInput(rt), width: ui.NewInput(rt), height: ui.NewInput(rt),
		area: ui.NewTextarea(rt), otp: ui.NewInputOTP(rt, 6), bar: ui.NewMenubar(rt), native: ui.NewNativeSelect(rt),
		popover: ui.NewPopover(rt), radio: ui.NewRadioGroup(rt), fruit: ui.NewSelect(rt), slider: ui.NewSlider(rt),
		airplane: ui.NewSwitch(rt), tabs: ui.NewTabs(rt), align: ui.NewToggleGroup(rt), marks: ui.NewToggleGroup(rt), sidebar: ui.NewSidebar(rt),
		chosen: "nothing yet", side: "Home", person: "Pedro Duarte", panel: "Bottom", statusBar: true, progress: 60, goal: 350, row: -1,
	}
	k.tip.Side, k.tip.Align = ui.Bottom, ui.End
	k.invite, k.moves = k.menu.Sub(), newMoves(rt)
	k.file, k.edit, k.view = k.bar.Menu(), k.bar.Menu(), k.bar.Menu()
	choose := func(item string) { k.chosen = item }
	k.command.OnSelect, k.menu.OnSelect, k.context.OnSelect = choose, choose, choose
	k.file.OnSelect, k.edit.OnSelect, k.view.OnSelect = choose, choose, choose
	for i := range k.checks {
		k.checks[i] = ui.NewCheckbox(rt)
	}
	k.checks[1].Checked, k.checks[2].Invalid, k.checks[3].Disabled = true, true, true
	for i, v := range []ui.Variant{ui.Default, ui.Outline, ui.Default} {
		k.toggles[i] = ui.NewToggle(rt)
		k.toggles[i].Variant, k.toggles[i].Size = v, []ui.Size{ui.SizeDefault, ui.SizeSM, ui.SizeLG}[i]
	}
	k.email.Placeholder, k.off.Placeholder, k.site.Placeholder = "m@example.com", "Disabled", "example"
	k.bad.Insert("not-an-email")
	k.width.Insert("100%")
	k.height.Insert("25px")
	k.off.Disabled, k.bad.Invalid = true, true
	k.area.Placeholder = "Type your message here."
	k.native.Options, k.native.Value = []string{"Todo", "In Progress", "Done", "Cancelled"}, "Todo"
	k.radio.Value, k.fruit.Placeholder, k.slider.Value = "comfortable", "Select a fruit", 33
	k.popover.Align, k.menu.Align = ui.Start, ui.Start
	k.align.Variant, k.align.Value = ui.Outline, []string{"left"}
	k.marks.Multiple, k.marks.Value = true, []string{"bold"}
	for i := range k.spinners {
		k.spinners[i] = ui.NewSpinner(rt)
	}
	k.lookup, k.query, k.url, k.message = ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewTextarea(rt)
	k.lookup.Insert("twind")
	k.query.Placeholder, k.message.Placeholder = "Search...", "Ask, search or chat..."
	k.url.Insert("twind.dev")
	k.framework = ui.NewCombobox(rt)
	k.framework.Placeholder, k.framework.Empty = "Select framework...", "No framework found."
	k.calendar = ui.NewCalendar(rt)
	k.calendar.Today = today
	k.nav = ui.NewNavigationMenu(rt)
	k.started, k.parts = k.nav.Item("started"), k.nav.Item("components")
	k.nav.OnSelect = func(href string) { k.href = href }
	k.scroller, k.sent = ui.NewMessageScroller(rt), len(chatLines())
	k.panes, k.stack, k.carousel = ui.NewResizable(rt), ui.NewResizable(rt), ui.NewCarousel(rt)
	k.panes.Sizes, k.stack.Orientation = []int{40, 60}, ui.Vertical
	k.signup, k.handle, k.mail, k.secret = ui.NewForm(rt), ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	k.plan, k.agree, k.bio, k.joined = ui.NewSelect(rt), ui.NewCheckbox(rt), ui.NewTextarea(rt), "not yet"
	k.handle.Placeholder, k.mail.Placeholder, k.secret.Placeholder, k.plan.Placeholder = "shadcn", "m@example.com", "8 or more characters", "Select a plan"
	k.bio.Placeholder = "A line about you"
	failing := func(bad bool, message string) string {
		if bad {
			return message
		}
		return ""
	}
	k.signup.Input("username", k.handle, func(v string) string { return failing(len(v) < shortestUsername, "Use 2 or more characters.") })
	k.signup.Input("email", k.mail,
		func(v string) string { return failing(v == "", "Email is required.") },
		func(v string) string { return failing(!strings.Contains(v, "@"), "Enter a valid email.") })
	k.signup.Input("password", k.secret, func(v string) string { return failing(len(v) < shortestPassword, "Use 8 or more characters.") })
	k.signup.Textarea("bio", k.bio, func(v string) string { return failing(len(v) > longestBio, "Keep it under 160.") })
	k.signup.Select("plan", k.plan, func(v string) string { return failing(v == "", "Choose a plan.") })
	k.signup.Checkbox("terms", k.agree, func(on bool) string { return failing(!on, "Accept the terms first.") })
	k.signup.OnSubmit = func(v ui.FormValues) { k.joined = v.Text["username"] + " on " + v.Text["plan"] }
	return k
}

func components() []page {
	return []page{
		{"accordion", accordionPage, nil},
		{"alert", alertPage, nil},
		{"alert dialog", alertDialogPage, nil},
		{"aspect ratio", aspectRatioPage, nil},
		{"attachment", attachmentPage, nil},
		{"avatar", avatarPage, nil},
		{"badge", badgePage, nil},
		{"breadcrumb", breadcrumbPage, nil},
		{"bubble", bubblePage, nil},
		{"button", buttonPage, nil},
		{"button group", buttonGroupPage, nil},
		{"calendar", calendarPage, nil},
		{"card", cardPage, nil},
		{"carousel", carouselPage, nil},
		{"checkbox", checkboxPage, nil},
		{"collapsible", collapsiblePage, nil},
		{"combobox", comboboxPage, nil},
		{"command", commandPage, nil},
		{"command dialog", commandDialogPage, nil},
		{"context menu", contextMenuPage, nil},
		{"dialog", dialogPage, nil},
		{"direction", directionPage, nil},
		{"drawer", drawerPage, nil},
		{"dropdown menu", dropdownMenuPage, nil},
		{"empty", emptyPage, nil},
		{"field", fieldPage, nil},
		{"form", formPage, []key{{"enter", "submit"}}},
		{"hover card", hoverCardPage, nil},
		{"input", inputPage, nil},
		{"input group", inputGroupPage, nil},
		{"input otp", inputOTPPage, nil},
		{"item", itemPage, nil},
		{"kbd", kbdPage, nil},
		{"label", labelPage, nil},
		{"marker", markerPage, nil},
		{"menubar", menubarPage, nil},
		{"message", messagePage, nil},
		{"message scroller", messageScrollerPage, []key{{"n", "reply"}}},
		{"native select", nativeSelectPage, nil},
		{"navigation menu", navigationMenuPage, nil},
		{"pagination", paginationPage, nil},
		{"popover", popoverPage, nil},
		{"progress", progressPage, nil},
		{"radio group", radioGroupPage, nil},
		{"resizable", resizablePage, nil},
		{"scroll area", scrollAreaPage, nil},
		{"select", selectPage, nil},
		{"separator", separatorPage, nil},
		{"sheet", sheetPage, nil},
		{"sidebar", sidebarPage, nil},
		{"skeleton", skeletonPage, nil},
		{"slider", sliderPage, nil},
		{"spinner", spinnerPage, nil},
		{"switch", switchPage, nil},
		{"table", tablePage, nil},
		{"tabs", tabsPage, nil},
		{"textarea", textareaPage, nil},
		{"toaster", toasterPage, nil},
		{"toggle", togglePage, nil},
		{"toggle group", toggleGroupPage, nil},
		{"tooltip", tooltipPage, nil},
	}
}

func show(title, about string, demo ...twi.Node) twi.Node {
	return el("flex flex-col items-center gap-1", append([]twi.Node{heading(title, about)}, demo...)...)
}

func row(children ...twi.Node) twi.Node { return el("flex flex-row items-center gap-2", children...) }

func alertPage(controls) twi.Node {
	return show("Alert", "a callout that asks for attention, default and destructive",
		el("w-60 flex flex-col gap-1",
			ui.Alert(ui.Default, ui.AlertTitle(twi.Text("✓ Success! Your changes have been saved")), ui.AlertDescription(twi.Text("An alert with an icon, a title and a description."))),
			ui.Alert(ui.Destructive, ui.AlertTitle(twi.Text("⊗ Unable to process your payment.")), ui.AlertDescription(twi.Text("Verify your billing information and try again."))),
		))
}

func avatarPage(controls) twi.Node {
	return show("Avatar", "a user's image or initials in three sizes",
		row(
			ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("CN"))),
			ui.Avatar(ui.SizeDefault, ui.AvatarFallback(twi.Text("CN"))),
			ui.Avatar(ui.SizeLG, ui.AvatarFallback(twi.Text("ER"))),
		))
}

func badgePage(controls) twi.Node {
	var badges []twi.Node
	for i, v := range []ui.Variant{ui.Default, ui.Secondary, ui.Destructive, ui.Outline, ui.Ghost, ui.Link} {
		badges = append(badges, ui.Badge(v, twi.Text([]string{"default", "secondary", "destructive", "outline", "ghost", "link"}[i])))
	}
	return show("Badge", "six variants", row(badges...))
}

func breadcrumbPage(c controls) twi.Node {
	link := func(s string, page int) twi.Node {
		return ui.BreadcrumbItem(ui.BreadcrumbLink(twi.OnClick(func(*twi.Event) { c.update(func(s *state) { s.page = page }) }), twi.Text(s)))
	}
	return show("Breadcrumb", "a click on a link opens its page",
		ui.Breadcrumb(ui.BreadcrumbList(
			link("Surfaces", 0), ui.BreadcrumbSeparator(),
			ui.BreadcrumbItem(ui.BreadcrumbEllipsis()), ui.BreadcrumbSeparator(),
			link("Layout", 2), ui.BreadcrumbSeparator(),
			ui.BreadcrumbItem(ui.BreadcrumbPage(twi.Text("Breadcrumb"))),
		)))
}

func buttonPage(c controls) twi.Node {
	k := c.kit
	pressable := func(v ui.Variant, s ui.Size, label string, extra ...twi.NodeOption) twi.Node {
		return ui.Button(v, s, append(extra, c.clicked(func() { k.pressed = label }), twi.Text(label))...)
	}
	var variants []twi.Node
	for i, v := range []ui.Variant{ui.Default, ui.Secondary, ui.Destructive, ui.Outline, ui.Ghost, ui.Link} {
		variants = append(variants, pressable(v, ui.SizeDefault, []string{"Default", "Secondary", "Destructive", "Outline", "Ghost", "Link"}[i]))
	}
	return show("Button", "six variants, five sizes and disabled; a click or Enter presses",
		row(variants...),
		row(
			pressable(ui.Outline, ui.SizeXS, "XS"), pressable(ui.Outline, ui.SizeSM, "Small"), pressable(ui.Outline, ui.SizeDefault, "Default size"),
			pressable(ui.Outline, ui.SizeLG, "Large"), pressable(ui.Outline, ui.SizeIcon, "◆"), pressable(ui.Default, ui.SizeDefault, "Disabled", twi.Disabled()),
		),
		txt("text-muted-foreground", "pressed: "+k.pressed),
	)
}

func buttonGroupPage(controls) twi.Node {
	outline := func(s string) twi.Node { return ui.Button(ui.Outline, ui.SizeDefault, twi.Text(s)) }
	return show("Button group", "buttons joined in a row or a column",
		row(
			ui.ButtonGroup(ui.Horizontal, outline("Archive"), outline("Report"), outline("Snooze")),
			ui.ButtonGroup(ui.Horizontal, ui.ButtonGroupText(twi.Text("https://")), outline("twind.dev")),
			ui.ButtonGroup(ui.Vertical, outline("+"), outline("-")),
		))
}

func cardPage(c controls) twi.Node {
	return show("Card", "header, action, content and footer",
		ui.Card(twi.Class("w-56"),
			ui.CardHeader(
				ui.CardTitle(twi.Text("Login to your account")),
				ui.CardDescription(twi.Text("Enter your email below to login")),
				ui.CardAction(ui.Button(ui.Link, ui.SizeXS, twi.Text("Sign Up"))),
			),
			ui.CardContent(ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), c.kit.email.Node())),
			ui.CardFooter(twi.Class("gap-2"), ui.Button(ui.Default, ui.SizeDefault, twi.Text("Login")), ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Login with Google"))),
		))
}

func emptyPage(controls) twi.Node {
	return show("Empty", "what a list shows before it has anything",
		ui.Empty(twi.Class("w-60 border border-dashed"),
			ui.EmptyHeader(
				ui.EmptyMedia(ui.Icon, twi.Text("▣")),
				ui.EmptyTitle(twi.Text("No projects yet")),
				ui.EmptyDescription(twi.Text("You have not created a project yet. Start by creating your first one.")),
			),
			ui.EmptyContent(row(ui.Button(ui.Default, ui.SizeDefault, twi.Text("Create project")), ui.Button(ui.Outline, ui.SizeDefault, twi.Text("Import project")))),
		))
}

func fieldPage(c controls) twi.Node {
	k := c.kit
	return show("Field", "a set with a legend, labels, a description, a separator and an error",
		ui.FieldSet(twi.Class("w-56"),
			ui.FieldLegend(twi.Text("Payment method")),
			ui.FieldGroup(
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Name on card")), k.email.Node(), ui.FieldDescription(twi.Text("As it is printed on the card"))),
				ui.FieldSeparator(twi.Text("or")),
				ui.Field(ui.Horizontal, k.checks[0].Node(), ui.FieldLabel(twi.Text("Same as the shipping address"))),
				ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Card number")), k.bad.Node(), ui.FieldError(twi.Text("Enter a valid card number"))),
			),
		))
}

func formPage(c controls) twi.Node {
	f, k := c.kit.signup, c.kit
	text := func(key, label string) twi.Node {
		return f.Item(key, f.Label(key, twi.Text(label)), f.Control(key))
	}
	return show("Form", "rules per field; a message on blur and on submit, focus on the first error",
		el("flex flex-row gap-3",
			el("w-28 flex flex-col gap-1", text("username", "Username"), text("email", "Email")),
			el("w-28 flex flex-col gap-1",
				text("password", "Password"),
				f.Item("plan", f.Label("plan", twi.Text("Plan")), k.plan.Node(twi.Class("w-full"), f.Control("plan", twi.Class("w-full")), k.plan.Content(k.plan.Item("free", "Free"), k.plan.Item("pro", "Pro"), k.plan.Item("team", "Team")))),
				f.Item("terms", ui.Field(ui.Horizontal, f.Control("terms"), f.Label("terms", twi.Text("Accept the terms")))),
			),
			el("w-28 flex flex-col", text("bio", "Bio")),
		),
		row(c.uiButton("form-submit", ui.Default, "Create account", func(*state) { f.Submit() }), c.uiButton("form-reset", ui.Outline, "Reset", func(*state) { f.Reset() }), txt("text-muted-foreground", "joined: "+k.joined)))
}

func itemPage(controls) twi.Node {
	return show("Item", "media, content and actions in a row; outline, muted and small",
		ui.ItemGroup(twi.Class("w-60 gap-1"),
			ui.Item(ui.Outline, ui.SizeDefault,
				ui.ItemContent(ui.ItemTitle(twi.Text("Basic item")), ui.ItemDescription(twi.Text("A title and a description."))),
				ui.ItemActions(ui.Button(ui.Outline, ui.SizeSM, twi.Text("Action"))),
			),
			ui.ItemSeparator(),
			ui.Item(ui.Muted, ui.SizeSM,
				ui.ItemMedia(ui.Icon, twi.Text("✓")),
				ui.ItemContent(ui.ItemTitle(twi.Text("Your profile has been verified."))),
				ui.ItemActions(twi.Text("›")),
			),
			ui.Item(ui.Default, ui.SizeDefault,
				ui.ItemMedia(ui.Default, ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text("ER")))),
				ui.ItemContent(ui.ItemTitle(twi.Text("evilrabbit")), ui.ItemDescription(twi.Text("Last seen 5 months ago"))),
			),
		))
}

func kbdPage(controls) twi.Node {
	return show("Kbd", "keys and chords",
		row(ui.KbdGroup(ui.Kbd(twi.Text("ctrl")), ui.Kbd(twi.Text("⇧")), ui.Kbd(twi.Text("k"))), txt("text-muted-foreground", "then"), ui.KbdGroup(ui.Kbd(twi.Text("ctrl")), twi.Text("+"), ui.Kbd(twi.Text("b")))),
	)
}

func labelPage(c controls) twi.Node {
	box := c.kit.checks[1]
	return show("Label", "a click on the label flips its checkbox",
		row(box.Node(), ui.Label(c.clicked(func() { box.Checked = !box.Checked }), twi.Text("Accept terms and conditions"))),
	)
}

func paginationPage(c controls) twi.Node {
	k := c.kit
	links := []twi.NodeOption{ui.PaginationItem(ui.PaginationPrevious(c.clicked(func() { k.at = max(k.at-1, 0) })))}
	for i := range 3 {
		links = append(links, ui.PaginationItem(ui.PaginationLink(i == k.at, c.clicked(func() { k.at = i }), twi.Text(strconv.Itoa(i+1)))))
	}
	links = append(links, ui.PaginationItem(ui.PaginationEllipsis()), ui.PaginationItem(ui.PaginationNext(c.clicked(func() { k.at = min(k.at+1, 2) }))))
	return show("Pagination", "previous, pages and next; a click or Enter turns",
		el("w-60", ui.Pagination(ui.PaginationContent(links...))),
		txt("text-muted-foreground", "page "+strconv.Itoa(k.at+1)+" of 3"),
	)
}

func progressPage(c controls) twi.Node {
	k := c.kit
	step := func(label string, by int) twi.Node {
		return ui.Button(ui.Outline, ui.SizeSM, c.clicked(func() { k.progress = min(max(k.progress+by, 0), 100) }), twi.Text(label))
	}
	return show("Progress", "a bar in hundredths of its width",
		el("w-60 flex flex-col", ui.Progress(k.progress)),
		row(step("- 10", -10), txt("w-5 text-center", strconv.Itoa(k.progress)+"%"), step("+ 10", 10)),
	)
}

func separatorPage(controls) twi.Node {
	return show("Separator", "a hairline across or down",
		el("w-50 flex flex-col gap-1",
			el("flex flex-col", txt("font-medium", "Radix Primitives"), txt("text-muted-foreground", "An open-source UI component library.")),
			ui.Separator(ui.Horizontal),
			el("flex flex-row h-1 gap-2", twi.Text("Blog"), ui.Separator(ui.Vertical), twi.Text("Docs"), ui.Separator(ui.Vertical), twi.Text("Source")),
		))
}

func skeletonPage(controls) twi.Node {
	return show("Skeleton", "placeholders while content loads; still here, so no frames",
		row(ui.Skeleton(twi.Class("h-3 w-6 rounded-full")), el("flex flex-col gap-1", ui.Skeleton(twi.Class("h-1 w-40")), ui.Skeleton(twi.Class("h-1 w-30")))),
	)
}

func tablePage(c controls) twi.Node {
	k := c.kit
	invoices := [][4]string{{"INV001", "Paid", "Credit Card", "$250.00"}, {"INV002", "Pending", "PayPal", "$150.00"}, {"INV003", "Unpaid", "Bank Transfer", "$350.00"}, {"INV004", "Paid", "Credit Card", "$450.00"}}
	var rows []twi.NodeOption
	for i, inv := range invoices {
		selected := map[bool]string{true: "bg-muted", false: ""}[i == k.row]
		rows = append(rows, ui.TableRow(twi.Class(selected), c.clicked(func() { k.row = i }),
			ui.TableCell(twi.Class("font-medium"), twi.Text(inv[0])), ui.TableCell(twi.Text(inv[1])), ui.TableCell(twi.Text(inv[2])), ui.TableCell(twi.Class("text-right"), twi.Text(inv[3])),
		))
	}
	return show("Table", "header, rows, footer and caption; a click selects a row",
		el("w-70", ui.Table(
			ui.TableHeader(ui.TableRow(ui.TableHead(twi.Text("Invoice")), ui.TableHead(twi.Text("Status")), ui.TableHead(twi.Text("Method")), ui.TableHead(twi.Class("text-right"), twi.Text("Amount")))),
			ui.TableBody(rows...),
			ui.TableFooter(ui.TableRow(ui.TableCell(twi.Text("Total")), ui.TableCell(), ui.TableCell(), ui.TableCell(twi.Class("text-right"), twi.Text("$1,200.00")))),
			ui.TableCaption(twi.Text("A list of your recent invoices.")),
		)),
	)
}

func scrollAreaPage(controls) twi.Node {
	tags := []twi.NodeOption{twi.Class("h-12 w-30 rounded-md border"), txt("px-1 font-medium", "Tags")}
	for i := range 30 {
		tags = append(tags, txt("px-1", "v1.2.0-beta."+strconv.Itoa(30-i)))
	}
	return show("Scroll area", "the wheel, arrows and PageDown scroll it", ui.ScrollArea(tags...))
}

func sidebarPage(c controls) twi.Node {
	k := c.kit
	button := func(glyph, name string) twi.Node {
		return ui.SidebarMenuButton(ui.SizeDefault, k.side == name, c.clicked(func() { k.side = name }), txt("w-1", glyph), twi.Text(name))
	}
	sub := func(name string) twi.Node {
		return ui.SidebarMenuSubItem(ui.SidebarMenuSubButton(k.side == name, c.clicked(func() { k.side = name }), twi.Text(name)))
	}
	return show("Sidebar", "ctrl+b or the trigger collapses it to icons",
		el("h-16 w-80 flex flex-row rounded-lg border overflow-hidden",
			k.sidebar.Provider(
				k.sidebar.Node(
					ui.SidebarHeader(txt("px-1 font-semibold", "◆ Acme Inc.")),
					ui.SidebarContent(ui.SidebarGroup(ui.SidebarGroupLabel(twi.Text("Application")), ui.SidebarGroupContent(ui.SidebarMenu(
						ui.SidebarMenuItem(button("⌂", "Home")),
						ui.SidebarMenuItem(button("▤", "Inbox"), ui.SidebarMenuBadge(twi.Text("24"))),
						ui.SidebarMenuItem(button("▦", "Calendar")),
						ui.SidebarMenuItem(button("◎", "Settings"), ui.SidebarMenuSub(sub("Profile"), sub("Billing"))),
					)))),
					ui.SidebarSeparator(),
					ui.SidebarFooter(txt("px-1", "shadcn")),
				),
				ui.SidebarInset(
					el("flex flex-row items-center gap-1 px-1 border-b", k.sidebar.Trigger(), txt("text-muted-foreground", "ctrl+b")),
					txt("px-1 pt-1", "Open: "+k.side),
				),
			),
		))
}
