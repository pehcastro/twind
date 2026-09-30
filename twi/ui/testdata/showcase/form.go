package main

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func form(rt *twi.Runtime, focus string) func() twi.Node {
	auto := func(name string) []twi.NodeOption {
		if name == focus {
			return []twi.NodeOption{twi.AutoFocus()}
		}
		return nil
	}
	email, username, site := ui.NewInput(rt), ui.NewInput(rt), ui.NewInput(rt)
	email.Placeholder, site.Placeholder = "Email", "example"
	username.Insert("shadcn!")
	username.Invalid = true
	message := ui.NewTextarea(rt)
	message.Placeholder = "Type your message here."
	status := ui.NewNativeSelect(rt)
	status.Options, status.Value = []string{"Select status", "Todo", "In Progress", "Done", "Cancelled"}, "Select status"
	terms, notices, off := ui.NewCheckbox(rt), ui.NewCheckbox(rt), ui.NewCheckbox(rt)
	notices.Checked, off.Disabled = true, true
	airplane := ui.NewSwitch(rt)
	spacing := ui.NewRadioGroup(rt)
	spacing.Value = "comfortable"
	bookmark := ui.NewToggle(rt)
	bookmark.Variant, bookmark.Size = ui.Outline, ui.SizeSM
	marks := ui.NewToggleGroup(rt)
	marks.Variant, marks.Multiple, marks.Value = ui.Outline, true, []string{"b"}
	volume := ui.NewSlider(rt)
	volume.Value = 50
	code := ui.NewInputOTP(rt, 6)
	return func() twi.Node {
		labelled := func(control twi.Node, name string) twi.Node { return row(control, ui.Label(label(name))) }
		field := func(name string, children ...twi.NodeOption) twi.Node {
			return ui.Field(ui.Vertical, append([]twi.NodeOption{ui.FieldLabel(label(name))}, children...)...)
		}
		radios := append(auto("radio"), spacing.Item("default", ui.Label(label("Default"))), spacing.Item("comfortable", ui.Label(label("Comfortable"))), spacing.Item("compact", ui.Label(label("Compact"))))
		toggles := append(auto("toggles"), marks.Item("b", label("B")), marks.Item("i", label("I")), marks.Item("u", label("U")))
		slots := append(auto("otp"), ui.InputOTPGroup(code.Slot(0), code.Slot(1), code.Slot(2)), ui.InputOTPSeparator(), ui.InputOTPGroup(code.Slot(3), code.Slot(4), code.Slot(5)))
		return el("flex flex-row grow gap-6",
			el("flex flex-col gap-2 w-50",
				field("Email", email.Node(auto("email")...)),
				field("Username", username.Node(), ui.FieldError(label("Only letters and digits."))),
				field("Your message", message.Node(auto("textarea")...)),
				field("Website", site.Group(ui.InputGroupAddon(ui.InlineStart, ui.InputGroupText(label("https://"))), ui.InputGroupAddon(ui.InlineEnd, ui.InputGroupText(label(".com"))))),
				field("Status", status.Node()),
			),
			el("flex flex-col gap-2 w-44",
				section("Checkbox",
					labelled(terms.Node(auto("checkbox")...), "Accept terms and conditions"),
					el("flex flex-row gap-2", notices.Node(), el("flex flex-col", ui.Label(label("Enable notifications")), text("text-muted-foreground", "You can turn them off at any time."))),
					labelled(off.Node(), "Disabled"),
				),
				section("Switch", labelled(airplane.Node(), "Airplane Mode")),
				section("Radio group", spacing.Node(radios...)),
			),
			el("flex flex-col gap-2 grow",
				section("Toggle", row(bookmark.Node(label("Bookmark")))),
				section("Toggle group", row(marks.Node(toggles...))),
				section("Slider", row(el("flex flex-row w-28", volume.Node()), text("text-muted-foreground", strconv.Itoa(volume.Value)))),
				section("Input OTP", code.Node(slots...)),
			),
		)
	}
}
