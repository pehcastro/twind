package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func newProject(k kit) page {
	output := []struct{ mark, class, text string }{
		{"◇", "text-muted-foreground", "taskn 2.4.0 · 12 tasks"},
		{"✓", "text-primary", "lint       0.8 s  cached"},
		{"✓", "text-primary", "typecheck  2.1 s"},
		{"✓", "text-primary", "test       4.6 s  318 passed"},
		{"✓", "text-primary", "build      3.2 s"},
		{"●", "text-chart-3", "watching 214 files"},
	}
	step := 0
	var tick *twi.Timer
	advance := func() {
		tick = nil
		step = (step + 1) % (len(demoPrompt) + len(output) + 2)
		k.rt.Invalidate()
	}
	installs := ui.NewTabs(k.rt)
	installs.Value = "macOS"
	commands := map[string]string{
		"macOS":   "brew install taskn",
		"Linux":   "curl -fsSL https://taskn.example/install | sh",
		"Windows": "winget install taskn",
		"Go":      "go install taskn.example/cmd/taskn@latest",
	}
	release := func(version, date, notes, class string) twi.Node {
		return el("flex flex-row gap-2",
			el("flex flex-col w-2 shrink-0", txt(class, "●"), el("grow w-1 border-l border-primary/40")),
			el("flex flex-col grow pb-1",
				el("flex flex-row items-center gap-1", txt("rounded-full bg-primary/15 px-1 font-semibold text-primary", version), txt("text-muted-foreground", date)),
				txt("", notes)))
	}
	nav := func() twi.Node {
		return navbar("bg-background/80 shadow-md",
			el("flex flex-row items-center gap-1", txt("flex flex-row w-4 justify-center rounded-md bg-linear-to-br from-primary-300 to-primary-700 py-0.5 font-bold text-primary-foreground", "❯"), txt("font-bold py-0.5", "taskn")),
			k.links("text-muted-foreground", "Install", "Changelog"), el("grow"),
			el("flex flex-row items-center gap-1 rounded-full px-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-accent",
				twi.Focusable(), txt("text-chart-3", "★"), twi.Text("Star"), txt("rounded-full bg-muted px-1 text-muted-foreground", "18.4k")),
		)
	}
	body := func() twi.Node {
		if tick == nil {
			wait := typeDelay
			switch {
			case step > len(demoPrompt)+len(output):
				wait = loopPause
			case step >= len(demoPrompt):
				wait = lineDelay
			}
			tick = k.rt.After(wait, advance)
		}
		typed := demoPrompt[:min(step, len(demoPrompt))]
		term := []twi.NodeOption{el("flex flex-row whitespace-pre", txt("text-primary", "~/shop "), txt("text-chart-3", "❯ "), twi.Text(typed), txt("bg-primary w-1", " "))}
		for _, o := range output[:max(min(step-len(demoPrompt), len(output)), 0)] {
			term = append(term, el("flex flex-row whitespace-pre", txt(o.class, "  "+o.mark+" "), txt("", o.text)))
		}
		var tabs []twi.NodeOption
		for _, os := range []string{"macOS", "Linux", "Windows", "Go"} {
			tabs = append(tabs, installs.Trigger(os, twi.Text(os)))
		}
		command := commands[installs.Value]
		return el("flex flex-col shrink-0 gap-3 pt-2",
			section("flex-row items-center gap-3",
				el("flex flex-col w-58 shrink-0 gap-1",
					el("flex flex-row", txt("rounded-full bg-primary/15 px-2 py-0.5 text-primary shadow-[0_0_0_1px_var(--color-primary)]", "v2.4.0 · remote cache is here →")),
					el("pt-1", display("TASKN", 2, "text-primary-300", "text-primary", "text-primary-600")),
					txt("font-bold", "A tiny task runner that knows what changed."),
					txt("text-muted-foreground", "Describe tasks once, and taskn runs only what is affected, in parallel, with a cache your whole team shares."),
					el("flex flex-row items-center gap-1 pt-1",
						ui.Button(ui.Default, ui.SizeLG, twi.Class("rounded-lg py-0.5 shadow-lg"), twi.OnClick(func(*twi.Event) { k.rt.ScrollIntoView(sectionKey + "Install") }), twi.Text("Get started")),
						ui.Button(ui.Ghost, ui.SizeLG, twi.Class("rounded-lg py-0.5"), twi.Text("Read the docs →"))),
				),
				el("flex flex-col flex-1 min-w-0 overflow-hidden rounded-xl border bg-card shadow-xl",
					el("flex flex-row items-center gap-1 px-2 py-0.5 border-b bg-muted/60", lights(), el("grow"), txt("text-muted-foreground", "zsh · 96×24"), el("grow")),
					el("flex flex-col h-9 px-2 py-1", term...),
				),
			),
			section("",
				el("flex flex-row items-center justify-between rounded-xl bg-muted/40 px-4 py-1 shadow-[0_0_0_1px_var(--color-border)]",
					el("flex flex-col", txt("font-bold text-chart-3", "★ 18.4k"), txt("text-muted-foreground", "stars")),
					el("flex flex-col",
						el("flex flex-row",
							txt("rounded-full px-1 font-medium text-primary-foreground shadow-[0_0_0_1px_var(--color-background)] bg-primary-300", "ab"),
							txt("rounded-full px-1 font-medium text-primary-foreground shadow-[0_0_0_1px_var(--color-background)] bg-primary-400", "kt"),
							txt("rounded-full px-1 font-medium text-primary-foreground shadow-[0_0_0_1px_var(--color-background)] bg-primary-500", "mo"),
							txt("rounded-full px-1 font-medium text-primary-foreground shadow-[0_0_0_1px_var(--color-background)] bg-primary-600", "+309")),
						txt("text-muted-foreground", "contributors")),
					el("flex flex-col", txt("font-bold", "MIT"), txt("text-muted-foreground", "licence")),
					el("flex flex-col", txt("font-bold", "2.1M"), txt("text-muted-foreground", "downloads a month")),
				),
			),
			section("gap-1", anchor("Install"),
				el("flex flex-row items-end",
					el("flex flex-col grow", txt("text-primary font-medium", "Install"), txt("font-bold", "One binary, every platform.")),
					installs.Node(installs.List(tabs...))),
				el("flex flex-row items-center gap-2 rounded-xl border bg-card px-2 py-1 shadow-md",
					txt("py-0.5 text-primary", "$"), txt("grow py-0.5", command),
					el("flex flex-row items-center gap-1 rounded-lg px-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-accent",
						twi.Focusable(), k.copy(command), twi.Text("⧉ Copy"))),
			),
			section("gap-1 max-w-80", anchor("Changelog"),
				txt("text-primary font-medium", "Changelog"),
				txt("font-bold pb-1", "Small releases, often."),
				release("v2.4.0", "September 2026", "Remote cache: share build results across machines with one flag.", "text-primary"),
				release("v2.3.0", "July 2026", "Watch mode reruns only the tasks a saved file affects.", "text-muted-foreground"),
				release("v2.2.0", "May 2026", "Windows support, including long paths and console colours.", "text-muted-foreground"),
				release("v2.0.0", "January 2026", "Tasks in one typed file; the old YAML format still loads.", "text-muted-foreground"),
			),
			footer("❯ taskn", "A tiny task runner. MIT licensed.",
				[]string{"Project", "Docs", "Changelog", "Roadmap"},
				[]string{"Community", "Discussions", "Contributing", "Sponsors"},
			),
		)
	}
	return page{nav: nav, body: body}
}
