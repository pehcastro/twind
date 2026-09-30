package components

import "github.com/twind-dev/twind/twi"

func TextWrap(*twi.Runtime) func() twi.Node {
	sentence := "A long sentence wraps at word boundaries, measured in cells."
	panel := func(label string, body twi.Node) twi.Node {
		return twi.Element(twi.Class("flex flex-col w-20 rounded-md border px-1"),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text(label)),
			body,
		)
	}
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-row flex-wrap gap-1"),
			panel("wraps", twi.Element(twi.Text(sentence))),
			panel("truncate", twi.Element(twi.Class("truncate"), twi.Text(sentence))),
			panel("whitespace-pre", twi.Element(twi.Class("whitespace-pre"), twi.Text("a   b\n  c"))),
			panel("wide glyphs", twi.Element(twi.Text("漢字 한국어 🚀 two cells each"))),
		)
	}
}
