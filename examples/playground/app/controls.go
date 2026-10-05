package app

import (
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func accordionPage(c controls) twi.Node {
	a := c.kit.accordion
	item := func(value, question, answer string) twi.Node {
		return a.Item(value, a.Trigger(value, twi.Text(question)), a.Content(value, twi.Text(answer)))
	}
	return show("Accordion", "one section open at a time; a click, or arrows and Enter",
		a.Node(twi.Class("w-60"),
			item("shipping", "What are your shipping options?", "Standard in 5 to 7 days, express in 2 to 3, or overnight."),
			item("returns", "What is your return policy?", "Returns within 30 days of purchase, in the original packaging."),
			item("support", "How can I reach support?", "By email, chat or phone, around the clock."),
		))
}

func checkboxPage(c controls) twi.Node {
	var boxes []twi.Node
	for i, label := range []string{"Accept terms and conditions", "Enable notifications", "Invalid until checked", "Disabled"} {
		boxes = append(boxes, row(c.kit.checks[i].Node(), ui.Label(twi.Text(label))))
	}
	return show("Checkbox", "unchecked, checked, invalid and disabled; a click or Space", el("flex flex-col gap-1", boxes...))
}

func collapsiblePage(c controls) twi.Node {
	cl := c.kit.collapsible
	repo := func(name string) twi.Node { return txt("px-2 rounded-md border", name) }
	return show("Collapsible", "the trigger shows and hides the rest",
		cl.Node(twi.Class("w-50 gap-1"),
			el("flex flex-row items-center justify-between", txt("font-semibold", "@peduarte starred 3 repositories"), cl.Trigger(ui.Ghost, ui.SizeIcon, twi.Text("⇅"))),
			repo("@radix-ui/primitives"),
			cl.Content(twi.Class("gap-1"), repo("@radix-ui/colors"), repo("@stitches/react")),
		))
}

func commandPage(c controls) twi.Node {
	cmd := c.kit.command
	shortcut := func(name, keys string) ui.CommandItem {
		return cmd.Item(name, twi.Text(name), ui.CommandShortcut(twi.Text(keys)))
	}
	return show("Command", "type to filter, arrows and Enter or a click to choose",
		cmd.Node(twi.Class("w-50 border rounded-lg shadow-md"),
			cmd.Input("Type a command or search"),
			cmd.List(
				cmd.Group("Suggestions", cmd.Item("Calendar"), cmd.Item("Search Emoji"), cmd.Item("Calculator")),
				cmd.Separator(),
				cmd.Group("Settings", shortcut("Profile", "⌘P"), shortcut("Billing", "⌘B"), shortcut("Settings", "⌘S")),
			),
		),
		txt("text-muted-foreground", "chosen: "+c.kit.chosen),
	)
}

func inputPage(c controls) twi.Node {
	k := c.kit
	return show("Input", "a field, disabled, invalid, and a group with addons",
		el("w-50 flex flex-col gap-1",
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Email")), k.email.Node()),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Disabled")), k.off.Node()),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Invalid")), k.bad.Node()),
			ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Website")), k.site.Group(
				ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(twi.Text("https://"))),
				ui.InputGroupAddon(ui.InlineEnd, twi.Text(".com")),
			)),
		))
}

func inputOTPPage(c controls) twi.Node {
	o := c.kit.otp
	return show("Input OTP", "six letters or digits; Backspace removes the last",
		o.Node(ui.InputOTPGroup(o.Slot(0), o.Slot(1), o.Slot(2)), ui.InputOTPSeparator(), ui.InputOTPGroup(o.Slot(3), o.Slot(4), o.Slot(5))),
		txt("text-muted-foreground", "entered: "+o.Value),
	)
}

func nativeSelectPage(c controls) twi.Node {
	return show("Native select", "arrows, Home and End step through the options",
		c.kit.native.Node(twi.Class("w-30")),
	)
}

func radioGroupPage(c controls) twi.Node {
	g := c.kit.radio
	return show("Radio group", "one of three; arrows move, a click chooses",
		g.Node(
			g.Item("default", ui.Label(twi.Text("Default"))),
			g.Item("comfortable", ui.Label(twi.Text("Comfortable"))),
			g.Item("compact", ui.Label(twi.Text("Compact"))),
		))
}

func selectPage(c controls) twi.Node {
	f := c.kit.fruit
	return show("Select", "a list over the page with labels and a separator",
		f.Node(twi.Class("w-30"), f.Trigger(twi.Class("w-30")), f.Content(
			ui.SelectLabel(twi.Text("Fruits")),
			f.Item("apple", "Apple"), f.Item("banana", "Banana"), f.Item("blueberry", "Blueberry"),
			ui.SelectSeparator(),
			ui.SelectLabel(twi.Text("Vegetables")),
			f.Item("carrot", "Carrot"), f.Item("leek", "Leek"),
		)),
	)
}

func sliderPage(c controls) twi.Node {
	s := c.kit.slider
	return show("Slider", "arrows step by one, PageUp and PageDown by ten",
		row(el("w-40 flex flex-row", s.Node()), txt("w-3 text-right text-muted-foreground", strconv.Itoa(s.Value))),
	)
}

func switchPage(c controls) twi.Node {
	s := c.kit.airplane
	return show("Switch", "a click or Space flips it",
		row(s.Node(), ui.Label(twi.Text("Airplane mode")), txt("text-muted-foreground", map[bool]string{true: "on", false: "off"}[s.Checked])),
	)
}

func tabsPage(c controls) twi.Node {
	t := c.kit.tabs
	panel := func(title, about, field string) twi.Node {
		return ui.Card(ui.CardHeader(ui.CardTitle(twi.Text(title)), ui.CardDescription(twi.Text(about))), ui.CardContent(ui.Field(ui.Vertical, ui.FieldLabel(twi.Text(field)), txt("text-muted-foreground", "shadcn"))))
	}
	return show("Tabs", "arrows or a click switch the panel",
		t.Node(twi.Class("w-50"),
			t.List(t.Trigger("account", twi.Text("Account")), t.Trigger("password", twi.Text("Password"))),
			t.Content("account", panel("Account", "Make changes to your account here.", "Name")),
			t.Content("password", panel("Password", "Change your password here.", "Current password")),
		))
}

func textareaPage(c controls) twi.Node {
	return show("Textarea", "many lines; Enter breaks a line",
		el("w-50 flex flex-col", ui.Field(ui.Vertical, ui.FieldLabel(twi.Text("Your message")), c.kit.area.Node(), ui.FieldDescription(twi.Text("Your message is copied to the support team.")))),
	)
}

func togglePage(c controls) twi.Node {
	t := c.kit.toggles
	return show("Toggle", "default, outline small and large; a click or Space",
		row(t[0].Node(twi.Text("B")), t[1].Node(twi.Text("I")), t[2].Node(twi.Text("U"))),
	)
}

func toggleGroupPage(c controls) twi.Node {
	a, m := c.kit.align, c.kit.marks
	return show("Toggle group", "single choice in outline, several at once in default",
		row(
			a.Node(a.Item("left", twi.Text("left")), a.Item("center", twi.Text("center")), a.Item("right", twi.Text("right"))),
			m.Node(m.Item("bold", twi.Text("B")), m.Item("italic", twi.Text("I")), m.Item("underline", twi.Text("U"))),
		))
}
