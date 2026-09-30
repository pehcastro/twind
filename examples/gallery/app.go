package main

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/input"
)

func el(class string, children ...twi.Node) twi.Node {
	opts := []twi.NodeOption{twi.Class(class)}
	for _, c := range children {
		opts = append(opts, c)
	}
	return twi.Element(opts...)
}

func txt(class, s string) twi.Node { return el(class, twi.Text(s)) }

func diffRow(class, num, sign, indent, code string) twi.Node {
	return el("flex flex-row px-1 "+class,
		txt("w-6 text-muted-foreground", num),
		txt("w-2 font-bold", sign),
		txt(indent, code),
	)
}

func card(title string, rows ...twi.Node) twi.Node {
	return el("flex-1 flex flex-col px-2 border rounded-lg shadow-md bg-card text-card-foreground",
		append([]twi.Node{txt("font-bold pb-1", title)}, rows...)...,
	)
}

func tip(key, label string) twi.Node {
	return el("flex flex-row gap-1 text-muted-foreground", txt("font-bold text-indigo-500", key), twi.Text(label))
}

func session(name, when string) twi.Node {
	return el("flex flex-row", txt("grow", name), txt("w-10 text-right text-muted-foreground", when))
}

func gallery(rt *twi.Runtime) func() twi.Node {
	onKey := func(k input.KeyEvent) {
		if !k.Release && k.Key == input.KeyRune && k.Rune == 'q' {
			rt.Quit()
		}
	}
	return func() twi.Node {
		var menu []twi.Node
		for i, item := range [][2]string{{"👎", ":thumbsdown:"}, {"👍", ":thumbsup:"}, {"🎉", ":tada:"}} {
			class, mark := "flex flex-row gap-1 px-1", el("w-1")
			if i == 1 {
				class, mark = class+" bg-indigo-500/10 text-indigo-600 font-bold", txt("w-1", "❯")
			}
			menu = append(menu, el(class, mark, twi.Text(item[0]), twi.Text(item[1])))
		}
		return twi.Element(twi.Class("flex flex-col h-full bg-background text-foreground"), twi.OnKey(onKey),
			el("grow flex flex-col px-2 pt-1 pb-1",
				el("flex flex-row gap-1",
					txt("rounded-sm bg-indigo-600 text-white", "●"),
					txt("font-bold", "Edit"),
					txt("text-muted-foreground", "src/agent/loop.ts:6720"),
					txt("ml-3 px-2 bg-green-500/15 text-green-600", "+1 -1"),
				),
				el("flex flex-col mr-2",
					diffRow("bg-muted", "6718", "", "pl-2", "});"),
					diffRow("bg-muted", "6719", "", "", ""),
					diffRow("bg-red-500/15", "6720", "-", "pl-2", "it.each(["),
					diffRow("bg-green-500/15", "6720", "+", "pl-2", "it.each<{ preserves: boolean; expected: string[] }>(["),
					diffRow("bg-muted", "6721", "", "pl-4", `{ preserves: false, expected: ["execute"] },`),
				),
				txt("pt-1 pb-1 font-bold", "Welcome back!"),
				el("flex flex-row gap-5 px-1",
					card("Tips",
						tip("#", "prompt actions"),
						tip("/", "commands"),
						tip("!", "run bash"),
						txt("pt-1 text-muted-foreground", "Use /tan to fork into a background agent"),
					),
					card("Recent sessions",
						session("Raft Consensus Partitions", "just now"),
						session("clone repo and run release.ts", "30m ago"),
						session("Debug: Show FPS", "3h ago"),
						session("Fix Conflicts", "3h ago"),
					),
				),
				el("flex flex-row gap-1 pt-2",
					txt("text-indigo-500", "❯"),
					twi.Text("Using a mermaid diagram explain how partitions work under raft"),
				),
				el("flex flex-row items-center gap-2 pt-1 pl-2",
					el("w-50 h-1 rounded-full bg-linear-to-r from-indigo-500 via-fuchsia-500 to-pink-600"),
					txt("text-muted-foreground", "64%"),
				),
				el("grow"),
				el("relative flex flex-row px-2 border rounded-md bg-background",
					txt("pr-1 text-indigo-500", "❯"),
					twi.Text(":thumbs"),
					txt("text-indigo-500", "▏"),
					el("absolute bottom-2 left-6 z-10 w-36 flex flex-col border rounded-md shadow-lg bg-popover text-popover-foreground", menu...),
					el("absolute bottom-2 right-2 z-10 flex flex-col items-end gap-1",
						el("flex flex-row gap-1 px-1 border rounded-md shadow-sm bg-card text-card-foreground",
							twi.Text("95 lines up"), txt("text-muted-foreground", "⌄")),
						txt("px-1 text-muted-foreground", "59.5 tok/s"),
					),
				),
			),
			el("flex flex-row items-center gap-4 pr-4 bg-muted text-muted-foreground",
				txt("rounded-full px-3 bg-indigo-600 text-white font-bold", "omp"),
				el("flex flex-row gap-1", txt("text-indigo-500", "✧"), twi.Text("GPT-6 Luna")),
				el("flex flex-row gap-1",
					el("w-4 rounded-sm bg-border", el("w-3 h-1 rounded-sm bg-foreground")),
					twi.Text("off"),
				),
				el("grow"),
				twi.Text("2% of 872K · $ 0.01"),
			),
		)
	}
}
