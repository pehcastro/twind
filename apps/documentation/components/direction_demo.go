package components

import (
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/text"
	"github.com/pehcastro/twind/twi/ui"
)

func DirectionDemo(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return ui.Direction(text.DirRTL, twi.Class("flex flex-col w-full max-w-56 gap-1"),
			ui.Card(twi.Class("gap-0.5 py-0.5 border-[0.5px]"),
				ui.CardHeader(
					ui.CardTitle(twi.Text("התחברות לחשבון")),
					ui.CardDescription(twi.Text("הזינו את כתובת ה-email שלכם")),
				),
				ui.CardFooter(twi.Class("gap-1"),
					ui.Button(ui.ButtonDefault, ui.ButtonSizeSM, twi.Text("כניסה")),
					ui.Button(ui.ButtonOutline, ui.ButtonSizeSM, twi.Text("ביטול")),
				),
			),
			ui.Direction(text.DirLTR, twi.Class("text-muted-foreground"), twi.Text("dir ltr inside: left to right again")),
		)
	}
}
