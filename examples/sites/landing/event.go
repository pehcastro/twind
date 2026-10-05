package main

import (
	"fmt"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func newEvent(k kit) page {
	left := eventStartsIn
	var tick *twi.Timer
	days := ui.NewTabs(k.rt)
	days.Value = "Day 1"
	type talk struct{ at, title, who, track, pill string }
	schedule := map[string][]talk{
		"Day 1": {
			{"09:30", "Doors, coffee and badges", "", "", ""},
			{"10:15", "Interfaces that wait well", "Ines Varga", "Craft", "bg-primary/20 text-primary"},
			{"11:00", "Layout is a negotiation", "Tomás Reyes", "Systems", "bg-chart-2/25 text-chart-2"},
			{"12:30", "Lunch on the river terrace", "", "", ""},
			{"14:00", "Motion with a budget", "Aiko Mori", "Motion", "bg-chart-3/25 text-chart-3"},
			{"15:00", "Text is the interface", "Dara Okafor", "Craft", "bg-primary/20 text-primary"},
		},
		"Day 2": {
			{"09:45", "Designing for the slow network", "Lena Brandt", "Systems", "bg-chart-2/25 text-chart-2"},
			{"10:45", "Colour you can trust", "Rui Sato", "Craft", "bg-primary/20 text-primary"},
			{"12:00", "Workshops in four rooms", "", "", ""},
			{"14:30", "The quiet power of defaults", "Maya Hollis", "Motion", "bg-chart-3/25 text-chart-3"},
			{"16:00", "Closing panel and drinks", "Everyone", "", ""},
		},
	}
	type speaker struct{ initials, name, role, bio, avatar string }
	speakers := []speaker{
		{"IV", "Ines Varga", "Design engineer", "Builds editors for a living and has opinions about loading states.", "from-primary to-chart-3"},
		{"TR", "Tomás Reyes", "Layout researcher", "Wrote a flexbox engine twice, on purpose the second time.", "from-chart-2 to-primary"},
		{"AM", "Aiko Mori", "Motion lead", "Animates interfaces for apps used by nine million people.", "from-chart-3 to-chart-2"},
		{"DO", "Dara Okafor", "Type designer", "Draws typefaces for screens of every size, including tiny ones.", "from-primary-300 to-primary-700"},
		{"LB", "Lena Brandt", "Performance engineer", "Measures everything twice and ships it once.", "from-chart-2 to-chart-3"},
		{"RS", "Rui Sato", "Colour scientist", "Turns perceptual colour spaces into design tokens.", "from-chart-3 to-primary"},
	}
	cards := make([]*ui.HoverCard, len(speakers))
	for i := range cards {
		cards[i] = ui.NewHoverCard(k.rt)
	}
	tile := func(n int, label string) twi.Node {
		return el("flex flex-col items-center gap-0.5 w-16 rounded-xl bg-background/60 py-1 shadow-[0_0_0_1px_var(--color-border)]",
			display(fmt.Sprintf("%02d", n), 1, "text-foreground"), txt("text-muted-foreground", label))
	}
	nav := func() twi.Node {
		return navbar("bg-background/80 shadow-md",
			el("flex flex-row items-center gap-1", txt("flex flex-row w-4 justify-center rounded-md bg-linear-to-br from-primary to-chart-2 py-0.5 font-bold text-white", "◆"), txt("font-bold py-0.5", "Fieldwork 26")),
			k.links("text-muted-foreground", "Schedule", "Speakers", "Venue"), el("grow"),
			ui.Button(ui.Default, ui.SizeSM, twi.Class("rounded-full py-0.5 shadow-md"), twi.OnClick(func(*twi.Event) { k.rt.ScrollIntoView(sectionKey + "Venue") }), twi.Text("Get tickets")),
		)
	}
	body := func() twi.Node {
		if tick == nil {
			tick = k.rt.After(time.Second, func() {
				tick, left = nil, max(left-time.Second, 0)
				k.rt.Invalidate()
			})
		}
		rows := []twi.NodeOption{}
		for _, t := range schedule[days.Value] {
			track := el("w-10")
			if t.track != "" {
				track = el("flex flex-row w-10", txt("rounded-full px-1 "+t.pill, t.track))
			}
			class := "group flex flex-row items-center gap-3 rounded-lg px-2 py-0.5 transition-colors duration-200 hover:bg-accent/60"
			if t.who == "" {
				class += " text-muted-foreground"
			}
			rows = append(rows, el(class, txt("w-6 font-medium text-primary", t.at), txt("grow font-semibold", t.title), txt("w-16 text-muted-foreground", t.who), track))
		}
		grid := []twi.NodeOption{}
		for i, s := range speakers {
			c := cards[i]
			grid = append(grid, el("flex flex-row items-center gap-2 rounded-xl border bg-card px-2 py-1 shadow-md transition duration-200 hover:-translate-y-0.5 hover:shadow-xl hover:border-primary/60",
				txt("flex flex-row w-6 h-3 shrink-0 items-center justify-center rounded-full bg-linear-to-br font-bold text-white "+s.avatar, s.initials),
				el("flex flex-col min-w-0",
					c.Node(c.Trigger(ui.Link, ui.SizeXS, twi.Class("px-0 font-semibold text-foreground"), twi.Text(s.name)),
						c.Content(txt("font-semibold", s.name), txt("text-primary", s.role), txt("text-muted-foreground", s.bio))),
					txt("text-muted-foreground", s.role))))
		}
		return el("flex flex-col shrink-0 gap-3 pt-1",
			section("",
				el("flex flex-col items-center gap-1 rounded-2xl bg-linear-to-br from-primary/45 via-chart-2/20 to-background px-4 py-2 shadow-xl",
					el("flex flex-row items-center gap-1 rounded-full bg-background/60 px-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)]",
						txt("text-primary", "●"), twi.Text("12 to 13 November · Porto · 900 seats")),
					el("pt-1", display("FIELDWORK 26", 1, "text-primary-300", "text-primary-400", "text-primary", "text-chart-3", "text-chart-2")),
					txt("font-bold text-center", "Two days on interface engineering, from layout engines to type."),
					el("flex flex-row items-center gap-2 pt-1",
						tile(int(left/(24*time.Hour)), "days"), tile(int(left/time.Hour)%24, "hours"),
						tile(int(left/time.Minute)%60, "minutes"), tile(int(left/time.Second)%60, "seconds")),
					txt("text-muted-foreground", "until the doors open"),
				),
			),
			section("gap-1", anchor("Schedule"),
				el("flex flex-row items-end",
					el("flex flex-col grow", txt("text-primary font-medium", "Schedule"), txt("font-bold", "Two days, one track, no clashes.")),
					days.Node(days.List(days.Trigger("Day 1", twi.Text("Thu 12")), days.Trigger("Day 2", twi.Text("Fri 13"))))),
				el("flex flex-col rounded-2xl border bg-card/60 px-1 py-1", rows...),
			),
			section("gap-1", anchor("Speakers"),
				txt("text-primary font-medium", "Speakers"),
				txt("font-bold", "People who build the things you use every day."),
				el("grid grid-cols-3 gap-2 pt-1", grid...),
			),
			section("flex-row gap-2", anchor("Venue"),
				el("flex flex-col flex-1 gap-0.5 rounded-2xl border bg-linear-to-br from-chart-2/25 via-card to-card px-3 py-1.5 shadow-md",
					txt("text-primary font-medium", "Venue"), txt("font-bold", "Casa da Ponte, Porto"),
					txt("text-muted-foreground", "A former tram depot on the river, ten minutes on foot from the old town."),
					txt("text-primary pt-0.5", "Directions ↗")),
				el("flex flex-col flex-1 gap-0.5 rounded-2xl border bg-card px-3 py-1.5 shadow-md",
					txt("text-primary font-medium", "Tickets"), el("flex flex-row items-end gap-1", txt("font-bold", "€290"), txt("text-muted-foreground line-through", "€390")),
					txt("text-muted-foreground", "Both days, lunch, the party and every recording."),
					el("flex flex-row pt-0.5", ui.Button(ui.Default, ui.SizeSM, twi.Class("rounded-full py-0.5"), twi.Text("Reserve a seat")))),
			),
			footer("◆ Fieldwork 26", "Interface engineering, in person.",
				[]string{"Event", "Schedule", "Speakers", "Venue"},
				[]string{"Attend", "Tickets", "Travel", "Code of conduct"},
			),
		)
	}
	bar := func() twi.Node {
		return el("flex flex-row shrink-0 justify-center border-t bg-card shadow-lg",
			section("flex-row items-center gap-2 py-0.5",
				txt("rounded-full bg-primary/20 px-2 py-0.5 font-medium text-primary", "Early bird"),
				txt("py-0.5", "€290 until 15 October"), txt("py-0.5 text-muted-foreground", "· 184 seats left"), el("grow"),
				ui.Button(ui.Default, ui.SizeSM, twi.Class("rounded-full py-0.5 shadow-md"), twi.Text("Get tickets →"))))
	}
	return page{nav, body, bar}
}
