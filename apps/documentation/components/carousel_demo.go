package components

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func CarouselDemo(rt *twi.Runtime) func() twi.Node {
	carousel := ui.NewCarousel(rt)
	return func() twi.Node {
		var slides []twi.NodeOption
		for i := 1; i <= 5; i++ {
			slides = append(slides, carousel.Item(ui.Card(twi.Class("h-full items-center justify-center"), twi.Text(strconv.Itoa(i)))))
		}
		return twi.Element(twi.Class("flex px-5"),
			carousel.Node(twi.Class("w-24 h-8"), carousel.Content(slides...), carousel.Previous(), carousel.Next()),
		)
	}
}
