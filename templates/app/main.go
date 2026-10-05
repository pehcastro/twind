package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/theme"
	"github.com/pehcastro/twind/twi/ui"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func main() {
	scheme := flag.String("theme", "Light", "theme to open with, Light or Dark")
	flag.Parse()
	if err := run(*scheme); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(scheme string) error {
	if _, ok := parseScheme(scheme); !ok {
		return fmt.Errorf("-theme %q: want Light or Dark", scheme)
	}
	sheet, err := Styles()
	if err != nil {
		return err
	}
	rt := twi.New(twi.Styles(sheet))
	return rt.Run(app(rt, scheme))
}

func App(rt *twi.Runtime) func() twi.Node { return app(rt, "Light") }

func parseScheme(name string) (theme.Scheme, bool) {
	s, ok := map[string]theme.Scheme{"Light": theme.Light, "Dark": theme.Dark}[name]
	return s, ok
}

func el(class string, children ...twi.NodeOption) twi.Node {
	return twi.Element(append([]twi.NodeOption{twi.Class(class)}, children...)...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func app(rt *twi.Runtime, scheme string) func() twi.Node {
	picker, dialog := ui.NewDropdownMenu(rt), ui.NewDialog(rt)
	picker.Align = ui.AlignEnd
	picker.OnSelect = func(string) {
		s, _ := parseScheme(scheme)
		rt.SetTheme(theme.Default().WithScheme(s))
	}
	picker.OnSelect(scheme)
	shortcuts := twi.OnKeyDown(func(e *twi.Event) {
		k := e.Key
		if dialog.Open || k.Release || k.Key != input.KeyRune || k.Modifiers != 0 {
			return
		}
		switch k.Rune {
		case 'q':
			rt.Quit()
		case 't':
			picker.Open = true
			rt.Invalidate()
		}
	})
	stat := func(label, value, note string) twi.Node {
		return ui.Card(twi.Class("flex-1 py-1"),
			ui.CardHeader(ui.CardDescription(twi.Text(label)), ui.CardTitle(twi.Text(value))),
			ui.CardContent(txt("text-muted-foreground", note)),
		)
	}
	return func() twi.Node {
		return el("flex flex-row h-full bg-background text-foreground", shortcuts,
			el("flex flex-col w-26 shrink-0 gap-1 px-2 py-1 border-r bg-muted/40",
				el("flex flex-row items-center gap-1 pb-1", txt("rounded-md px-1 bg-primary text-primary-foreground font-bold", "tw"), txt("font-semibold", "Hello Twind")),
				txt("rounded-md px-1 bg-accent text-accent-foreground font-medium", "Overview"),
				txt("px-1 text-muted-foreground", "Projects"),
				txt("px-1 text-muted-foreground", "Settings"),
				el("grow"),
				el("flex flex-col gap-1",
					el("flex flex-row items-center gap-1", ui.Kbd(twi.Text("tab")), txt("text-muted-foreground", "focus")),
					el("flex flex-row items-center gap-1", ui.Kbd(twi.Text("t")), txt("text-muted-foreground", "theme")),
					el("flex flex-row items-center gap-1", ui.Kbd(twi.Text("q")), txt("text-muted-foreground", "quit")),
				),
			),
			el("flex flex-col grow gap-2 px-4 py-1",
				el("flex flex-row items-center gap-2",
					el("flex flex-col grow", txt("font-bold", "Overview"), txt("text-muted-foreground", "Your new Twind app. Edit main.go, then go run . again.")),
					picker.Node(picker.Trigger(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Theme: "+scheme)), picker.Content(
						ui.DropdownMenuLabel(twi.Text("Theme")),
						ui.DropdownMenuSeparator(),
						picker.RadioItem("Light", &scheme),
						picker.RadioItem("Dark", &scheme),
					)),
				),
				el("flex flex-row gap-2",
					stat("Components", "twi/ui", "shadcn/ui, styled by Tailwind"),
					stat("Frame budget", "16 ms", "one write per frame"),
					stat("Themes", "2", "twind light and dark"),
				),
				ui.Card(twi.Class("py-1"),
					ui.CardHeader(ui.CardTitle(twi.Text("Start a project")), ui.CardDescription(twi.Text("A dialog traps focus, dims the page and closes on Escape."))),
					ui.CardContent(el("flex flex-row items-center gap-1", ui.Badge(ui.BadgeSecondary, twi.Text("twi/ui")), ui.Badge(ui.BadgeOutline, twi.Text("Tailwind 4")))),
					ui.CardFooter(twi.Class("gap-2"),
						dialog.Trigger(ui.ButtonDefault, ui.ButtonSizeDefault, twi.Text("New project")),
						txt("text-muted-foreground", "tab to focus, enter to open"),
					),
				),
			),
			dialog.Content(
				dialog.Header(dialog.Title(twi.Text("Create project")), dialog.Description(twi.Text("Projects group your screens and styles. You can rename it later."))),
				dialog.Footer(dialog.Close(ui.ButtonOutline, ui.ButtonSizeDefault, twi.Text("Cancel")), dialog.Close(ui.ButtonDefault, ui.ButtonSizeDefault, twi.Text("Create"))),
			),
		)
	}
}
