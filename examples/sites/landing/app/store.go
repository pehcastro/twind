package app

import (
	"fmt"
	"strconv"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

func newStore(k kit) page {
	type colour struct{ name, ink, swatch, tint string }
	colours := []colour{
		{"Lake", "text-chart-1", "bg-chart-1", "from-chart-1/25"},
		{"Moss", "text-chart-2", "bg-chart-2", "from-chart-2/25"},
		{"Ochre", "text-chart-3", "bg-chart-3", "from-chart-3/25"},
		{"Clay", "text-chart-4", "bg-chart-4", "from-chart-4/25"},
		{"Plum", "text-chart-5", "bg-chart-5", "from-chart-5/25"},
	}
	type size struct {
		name  string
		price int
	}
	sizes := []size{{"8 oz", 24}, {"12 oz", 28}, {"16 oz", 32}}
	type view struct {
		name    string
		picture []string
	}
	views := []view{
		{"Side", []string{
			".##########.....",
			".##########.###.",
			".##########....#",
			".##########....#",
			".##########....#",
			".##########.###.",
			".##########.....",
			".##########.....",
			"..########......",
			"...######.......",
		}},
		{"Front", []string{
			"...##########...",
			"..#..........#..",
			"..############..",
			"..############..",
			"..####....####..",
			"..###..##..###..",
			"..####....####..",
			"..############..",
			"...##########...",
			"....########....",
		}},
		{"Top", []string{
			"....######......",
			"..##########....",
			".###......###...",
			"###........###..",
			"###........#####",
			"###........#####",
			"###........###..",
			".###......###...",
			"..##########....",
			"....######......",
		}},
	}
	type line struct {
		colour, size, qty, price int
	}
	var cart []line
	picked, sized, shown, qty := 0, 1, 0, 1
	drawer := ui.NewSheet(k.rt, ui.Right)
	k.toast.Avoid(drawer)
	pick := func(set func()) twi.NodeOption {
		return twi.OnClick(func(*twi.Event) {
			set()
			k.rt.Invalidate()
		})
	}
	items := func() int {
		n := 0
		for _, l := range cart {
			n += l.qty
		}
		return n
	}
	nav := func() twi.Node {
		return navbar("bg-background shadow-sm",
			el("flex flex-row items-center gap-1", txt("flex flex-row w-4 justify-center rounded-full bg-linear-to-br from-primary-300 to-primary-700 py-0.5 font-bold text-white", "◡"), txt("font-bold py-0.5", "Hearth")),
			k.links("text-muted-foreground", "Details", "Reviews"), el("grow"),
			ui.Button(ui.Outline, ui.SizeSM, twi.Class("rounded-full py-0.5"), pick(func() { drawer.Open = true }),
				twi.Text("◫ Cart"), txt("rounded-full bg-primary px-1 text-primary-foreground", strconv.Itoa(items()))),
		)
	}
	body := func() twi.Node {
		c, s := colours[picked], sizes[sized]
		thumbs := []twi.NodeOption{}
		for i, v := range views {
			ring := "border bg-card hover:border-primary/60"
			if i == shown {
				ring = "border border-primary bg-card shadow-md"
			}
			thumbs = append(thumbs, el("flex flex-col flex-1 items-center gap-0.5 rounded-xl py-1 transition duration-200 "+ring, twi.Focusable(), pick(func() { shown = i }),
				bitmap(c.ink, 1, v.picture...), txt("text-muted-foreground", v.name)))
		}
		swatches := []twi.NodeOption{}
		for i, o := range colours {
			ring := "shadow-[0_0_0_1px_var(--color-border)]"
			if i == picked {
				ring = "shadow-[0_0_0_2px_var(--color-foreground)]"
			}
			swatches = append(swatches, el("w-4 h-2 rounded-full transition duration-200 "+o.swatch+" "+ring, twi.Focusable(), pick(func() { picked = i })))
		}
		pills := []twi.NodeOption{}
		for i, o := range sizes {
			class := "rounded-lg px-2 py-0.5 shadow-[0_0_0_1px_var(--color-border)] transition-colors duration-200 hover:bg-accent"
			if i == sized {
				class = "rounded-lg px-2 py-0.5 bg-foreground text-background font-medium"
			}
			pills = append(pills, el(class, twi.Focusable(), pick(func() { sized = i }), twi.Text(o.name)))
		}
		count := strconv.Itoa(items()) + " items"
		if items() == 1 {
			count = "1 item"
		}
		lines := []twi.NodeOption{}
		total := 0
		for _, l := range cart {
			total += l.price * l.qty
			lines = append(lines, el("flex flex-row items-center gap-1 px-2 py-0.5 border-b",
				el("w-4 h-2 shrink-0 rounded-md "+colours[l.colour].swatch),
				el("flex flex-col grow", txt("font-medium", "Hearth mug"), txt("text-muted-foreground", colours[l.colour].name+" · "+sizes[l.size].name+" × "+strconv.Itoa(l.qty))),
				txt("font-medium", fmt.Sprintf("$%d", l.price*l.qty))))
		}
		if len(cart) == 0 {
			lines = append(lines, txt("px-2 py-1 text-muted-foreground", "Nothing here yet."))
		}
		detail := func(glyph, title, text string) twi.Node {
			return el("flex flex-col flex-1 gap-0.5 rounded-xl border bg-card px-2 py-1 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-lg",
				txt("text-primary font-bold", glyph), txt("font-semibold", title), txt("text-muted-foreground", text))
		}
		return el("flex flex-col shrink-0 gap-3 pt-2",
			section("flex-row gap-3",
				el("flex flex-col w-62 shrink-0 gap-1",
					el("flex flex-col items-center justify-center rounded-2xl bg-linear-to-br via-card to-muted py-3 shadow-lg transition-colors duration-300 "+c.tint,
						bitmap(c.ink+" transition-colors duration-300", 2, views[shown].picture...)),
					el("flex flex-row gap-1", thumbs...),
				),
				el("flex flex-col grow gap-1",
					txt("text-muted-foreground", "Kitchen · Mugs"),
					txt("font-bold", "Hearth mug"),
					el("flex flex-row items-center gap-1", txt("text-chart-3", "★★★★★"), txt("text-muted-foreground", "4.9 · 212 reviews")),
					el("flex flex-row items-center gap-1 pt-0.5", txt("font-bold text-muted-foreground", "$"), display(strconv.Itoa(s.price), 1, "text-foreground"), txt("pl-1 text-muted-foreground", "free shipping over $60")),
					txt("text-muted-foreground", "Thrown by hand in small batches, glazed inside and out, and fired twice so it keeps tea hot a little longer."),
					el("flex flex-row gap-1 pt-0.5", txt("font-medium", "Colour"), txt("text-muted-foreground", c.name)),
					el("flex flex-row items-center gap-2", swatches...),
					txt("font-medium pt-0.5", "Size"),
					el("flex flex-row items-center gap-1", pills...),
					el("flex flex-row items-center gap-2 pt-1",
						el("flex flex-row items-center rounded-lg shadow-[0_0_0_1px_var(--color-border)]",
							el("px-2 py-0.5 rounded-l-lg hover:bg-accent", twi.Text("−"), twi.Focusable(), pick(func() { qty = max(qty-1, 1) })),
							el("px-1 py-0.5 font-medium", twi.Text(strconv.Itoa(qty))),
							el("px-2 py-0.5 rounded-r-lg hover:bg-accent", twi.Text("+"), twi.Focusable(), pick(func() { qty = min(qty+1, 9) }))),
						ui.Button(ui.Default, ui.SizeLG, twi.Class("grow rounded-lg py-0.5 shadow-lg"), twi.OnClick(func(*twi.Event) {
							cart = append(cart, line{picked, sized, qty, s.price})
							drawer.Open = true
							k.toast.Success("Added to cart", fmt.Sprintf("Hearth mug, %s, %s", c.name, s.name), ui.ToastAction{})
							k.rt.Invalidate()
						}), twi.Text(fmt.Sprintf("Add to cart · $%d", s.price*qty))),
					),
				),
			),
			section("flex-row gap-2", anchor("Details"),
				detail("◌", "Made by hand", "Every mug is thrown on a wheel, so no two handles are quite the same."),
				detail("≋", "Dishwasher safe", "A food-safe glaze that survives the machine and the microwave."),
				detail("↺", "Free returns", "Thirty days to change your mind, with a prepaid label in the box."),
			),
			section("gap-1 max-w-80", anchor("Reviews"),
				txt("font-bold", "What people say"),
				el("flex flex-col gap-0.5 rounded-2xl border bg-linear-to-br from-primary/10 via-card to-card px-3 py-1 shadow-md",
					txt("text-chart-3", "★★★★★"), txt("font-medium", "The handle fits three fingers and the glaze looks different in every light."),
					txt("text-muted-foreground", "Noor, Rotterdam")),
			),
			footer("◡ Hearth", "Ceramics for everyday tables.",
				[]string{"Shop", "Mugs", "Bowls", "Plates"},
				[]string{"Help", "Shipping", "Returns", "Care"},
			),
			drawer.Content(
				drawer.Header(drawer.Title(twi.Text("Your cart")), drawer.Description(twi.Text(count))),
				el("flex flex-col", lines...),
				drawer.Footer(
					el("flex flex-row justify-between font-semibold", twi.Text("Subtotal"), twi.Text(fmt.Sprintf("$%d", total))),
					ui.Button(ui.Default, ui.SizeDefault, twi.Class("rounded-lg py-0.5"), twi.Text("Checkout →")),
					drawer.Close(ui.Outline, ui.SizeDefault, twi.Class("rounded-lg py-0.5"), twi.Text("Keep shopping")),
				),
			),
		)
	}
	return page{nav: nav, body: body}
}
