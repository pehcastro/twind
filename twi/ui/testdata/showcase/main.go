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
	quit := twi.OnKey(func(k input.KeyEvent) {
		if !k.Release && k.Key == input.KeyRune && k.Rune == 'q' {
			rt.Quit()
		}
	})
	if err := rt.Run(func() twi.Node { return showcase(quit) }); err != nil {
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

func showcase(quit twi.NodeOption) twi.Node {
	label := twi.Text
	return el("flex flex-row h-full gap-4 px-3 py-1 bg-background text-foreground", quit,
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
