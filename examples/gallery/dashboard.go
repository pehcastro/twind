package main

import (
	"fmt"
	"strings"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/ui"
)

const (
	barHeights  = "h-0 h-1 h-2 h-3 h-4 h-5 h-6 h-7 h-8 h-9 h-10"
	rowsPerPage = 5
)

type document struct{ title, kind, status, reviewer string }

type month struct {
	name            string
	desktop, mobile int
}

func newDashboard(rt *twi.Runtime) func() twi.Node {
	docs := []document{
		{"Cover page", "Cover page", "In Process", "Eddie Lake"},
		{"Table of contents", "Table of contents", "Done", "Eddie Lake"},
		{"Executive summary", "Narrative", "Done", "Eddie Lake"},
		{"Technical approach", "Narrative", "Done", "Jamik Tashpulatov"},
		{"Design", "Narrative", "In Process", "Jamik Tashpulatov"},
		{"Capabilities", "Narrative", "In Process", "Jamik Tashpulatov"},
		{"Integration with existing systems", "Narrative", "In Process", "Jamik Tashpulatov"},
		{"Innovation and Advantages", "Narrative", "Done", ""},
		{"Overview of EMR's Solutions", "Technical content", "Done", ""},
		{"Advanced Algorithms", "Narrative", "Done", ""},
		{"Adaptive Protocols", "Narrative", "Done", ""},
		{"Advantages Over Current Tech", "Narrative", "Done", ""},
	}
	visitors := []month{{"Jan", 186, 80}, {"Feb", 305, 200}, {"Mar", 237, 120}, {"Apr", 73, 190}, {"May", 209, 130}, {"Jun", 214, 140}}
	peak, heights := 305, strings.Fields(barHeights)
	span := ui.NewToggleGroup(rt)
	span.Variant, span.Size, span.Value = ui.Outline, ui.SizeSM, []string{"6m"}
	at, pageCount := 0, (len(docs)+rowsPerPage-1)/rowsPerPage
	selected, picked := "", ""
	turnTo := func(page int) twi.NodeOption { return clicked(rt, func() { at = min(max(page, 0), pageCount-1) }) }
	stat := func(label, value string, up bool, delta, trend, note string) twi.Node {
		arrow := map[bool]string{true: "↗", false: "↘"}[up]
		class := "min-w-0 px-1 transition-colors hover:bg-muted/50"
		if picked == label {
			class += " border-ring"
		}
		return ui.Card(twi.Class(class), clicked(rt, func() { picked = label }),
			el("flex flex-col",
				ui.CardDescription(twi.Text(label)),
				el("flex flex-row items-center justify-between", ui.CardTitle(twi.Text(value)), ui.Badge(ui.Outline, twi.Text(arrow+" "+delta))),
			),
			el("flex flex-col", txt("font-medium", trend+" "+arrow), txt("text-muted-foreground truncate", note)),
		)
	}
	turn := onKeys(rt, func(k input.KeyEvent) bool {
		switch k.Key {
		case input.KeyArrowLeft:
			at = max(at-1, 0)
		case input.KeyArrowRight:
			at = min(at+1, pageCount-1)
		default:
			return false
		}
		return true
	})
	bar := func(value int, class string) twi.Node {
		return el("w-2 rounded-sm bg-linear-to-t " + class + " " + heights[max(value*(len(heights)-1)/peak, 1)])
	}
	legend := func(class, s string) twi.Node {
		return el("flex flex-row items-center gap-1", el("w-2 h-1 rounded-sm "+class), txt("text-muted-foreground", s))
	}
	return func() twi.Node {
		var rows []twi.NodeOption
		for _, d := range docs[at*rowsPerPage : min((at+1)*rowsPerPage, len(docs))] {
			mark := txt("text-muted-foreground", "◌")
			if d.status == "Done" {
				mark = txt("text-green-600 dark:text-green-400", "✓")
			}
			reviewer := txt("truncate", d.reviewer)
			if d.reviewer == "" {
				reviewer = txt("text-muted-foreground truncate", "Assign…")
			}
			rows = append(rows, ui.TableRow(twi.Class(map[bool]string{true: "bg-muted", false: ""}[d.title == selected]), clicked(rt, func() { selected = d.title }),
				ui.TableCell(twi.Class("min-w-0 truncate font-medium"), twi.Text(d.title)),
				ui.TableCell(twi.Class("hidden min-w-0 truncate lg:block"), twi.Text(d.kind)),
				ui.TableCell(twi.Class("flex-none w-17"), ui.Badge(ui.Outline, mark, twi.Text(d.status))),
				ui.TableCell(twi.Class("flex-none w-14 min-w-0"), reviewer),
			))
		}
		links := []twi.NodeOption{ui.PaginationItem(ui.PaginationPrevious(turnTo(at - 1)))}
		for i := range pageCount {
			links = append(links, ui.PaginationItem(ui.PaginationLink(i == at, turnTo(i), twi.Text(fmt.Sprint(i+1)))))
		}
		links = append(links, ui.PaginationItem(ui.PaginationNext(turnTo(at+1))))
		shown := visitors
		if len(span.Value) == 1 && span.Value[0] == "3m" {
			shown = visitors[3:]
		}
		var groups []twi.NodeOption
		for _, m := range shown {
			groups = append(groups, el("flex flex-col items-center w-4",
				el("flex flex-row items-end h-10", bar(m.desktop, "from-primary/60 to-primary"), bar(m.mobile, "from-primary/15 to-primary/40")),
				txt("text-muted-foreground", m.name),
			))
		}
		return el("flex flex-col grow gap-1",
			el("grid grid-cols-2 gap-2 md:grid-cols-4",
				stat("Total Revenue", "$1,250", true, "12.5%", "Trending up", "Last 6 months"),
				stat("New Customers", "1,234", false, "20%", "Down 20%", "Needs attention"),
				stat("Active Accounts", "45,678", true, "12.5%", "Retention up", "Above target"),
				stat("Growth Rate", "4.5%", true, "4.5%", "Steady growth", "Meets forecast"),
			),
			el("flex flex-row grow gap-2",
				ui.Card(twi.Class("flex-1 min-w-0"),
					ui.CardHeader(ui.CardTitle(twi.Text("Documents")), ui.CardDescription(twi.Text("Outline sections and their reviewers"))),
					ui.CardContent(ui.Table(
						ui.TableHeader(ui.TableRow(
							ui.TableHead(twi.Text("Header")),
							ui.TableHead(twi.Class("hidden lg:block"), twi.Text("Type")),
							ui.TableHead(twi.Class("flex-none w-17"), twi.Text("Status")),
							ui.TableHead(twi.Class("flex-none w-14"), twi.Text("Reviewer")),
						)),
						ui.TableBody(rows...),
					)),
					ui.CardFooter(twi.Class("justify-between"),
						txt("text-muted-foreground", fmt.Sprintf("Page %d of %d", at+1, pageCount)),
						el("flex flex-row rounded-md "+focusRing, twi.Focusable(), turn, ui.Pagination(ui.PaginationContent(links...))),
					),
				),
				ui.Card(twi.Class("w-30 shrink-0 lg:w-44"),
					ui.CardHeader(
						ui.CardTitle(twi.Text("Total Visitors")),
						ui.CardDescription(twi.Text(fmt.Sprintf("Last %d months", len(shown)))),
						ui.CardAction(span.Node(span.Item("6m", twi.Text("6m")), span.Item("3m", twi.Text("3m")))),
					),
					ui.CardContent(el("flex flex-row justify-around", groups...)),
					ui.CardFooter(twi.Class("gap-3"), legend("bg-primary", "Desktop"), legend("bg-primary/30", "Mobile")),
				),
			),
		)
	}
}
