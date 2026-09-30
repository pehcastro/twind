package main

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func newForms(rt *twi.Runtime) func() twi.Node {
	name, email := ui.NewInput(rt), ui.NewInput(rt)
	name.Insert("shadcn")
	email.Placeholder = "m@example.com"
	bio := ui.NewTextarea(rt)
	bio.Placeholder = "Tell us a little bit about yourself"
	plan := ui.NewSelect(rt)
	plan.Value = "pro"
	terms, notify := ui.NewCheckbox(rt), ui.NewSwitch(rt)
	notify.Checked = true
	density := ui.NewRadioGroup(rt)
	density.Value = "Comfortable"
	marks := ui.NewToggleGroup(rt)
	marks.Variant, marks.Multiple, marks.Value = ui.Outline, true, []string{"B"}
	volume := ui.NewSlider(rt)
	volume.Value = 50
	code := ui.NewInputOTP(rt, 6)
	saved := false
	field := func(label string, children ...twi.NodeOption) twi.Node {
		return ui.Field(ui.Vertical, append([]twi.NodeOption{ui.FieldLabel(twi.Text(label))}, children...)...)
	}
	labelled := func(control twi.Node, label string) twi.Node {
		return el("flex flex-row items-center gap-2", control, ui.Label(twi.Text(label)))
	}
	return func() twi.Node {
		status := txt("text-muted-foreground", "Changes are kept until you leave.")
		if saved {
			status = el("flex flex-row items-center gap-1", ui.Badge(ui.Secondary, twi.Text("✓ Saved")), txt("text-muted-foreground", "Profile updated."))
		}
		return ui.Card(
			ui.CardHeader(
				ui.CardTitle(twi.Text("Profile")),
				ui.CardDescription(twi.Text("This is how others will see you on the site.")),
			),
			ui.CardContent(el("flex flex-row gap-3",
				el("flex flex-col flex-1 min-w-0 gap-1",
					field("Username", name.Node()),
					field("Email", email.Node()),
					field("Bio", bio.Node()),
					field("Plan", plan.Node(twi.Class("w-full"), plan.Trigger(twi.Class("w-full")), plan.Content(
						plan.Item("free", "Free"), plan.Item("pro", "Pro"), plan.Item("team", "Team"), plan.Item("enterprise", "Enterprise"),
					))),
				),
				el("flex flex-col w-26 shrink-0 gap-2",
					labelled(terms.Node(), "Accept terms"),
					labelled(notify.Node(), "Email updates"),
					ui.FieldSet(ui.FieldLegend(twi.Text("Density")), density.Node(
						density.Item("Default", ui.Label(twi.Text("Default"))),
						density.Item("Comfortable", ui.Label(twi.Text("Comfortable"))),
						density.Item("Compact", ui.Label(twi.Text("Compact"))),
					)),
				),
				el("flex flex-col w-28 shrink-0 gap-2",
					field("Text style", marks.Node(marks.Item("B", twi.Text("B")), marks.Item("I", twi.Text("I")), marks.Item("U", twi.Text("U")))),
					field("Volume", el("flex flex-row items-center gap-2", el("flex flex-row grow", volume.Node()), txt("w-3 text-right text-muted-foreground", strconv.Itoa(volume.Value)))),
					field("Verification code", code.Node(ui.InputOTPGroup(code.Slot(0), code.Slot(1), code.Slot(2)), ui.InputOTPSeparator(), ui.InputOTPGroup(code.Slot(3), code.Slot(4), code.Slot(5)))),
				),
			)),
			ui.CardFooter(twi.Class("gap-2 pt-1 border-t"),
				status,
				el("grow"),
				ui.Button(ui.Outline, ui.SizeDefault, clicked(rt, func() { saved = false }), twi.Text("Cancel")),
				ui.Button(ui.Default, ui.SizeDefault, clicked(rt, func() { saved = true }), twi.Text("Save changes")),
			),
		)
	}
}
