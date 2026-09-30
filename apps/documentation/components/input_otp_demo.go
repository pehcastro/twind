package components

import (
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func InputOTPDemo(rt *twi.Runtime) func() twi.Node {
	code := ui.NewInputOTP(rt, 6)
	return func() twi.Node {
		return twi.Element(twi.Class("flex flex-col items-center gap-1"),
			code.Node(
				ui.InputOTPGroup(code.Slot(0), code.Slot(1), code.Slot(2)),
				ui.InputOTPSeparator(),
				ui.InputOTPGroup(code.Slot(3), code.Slot(4), code.Slot(5)),
			),
			twi.Element(twi.Class("text-muted-foreground"), twi.Text("entered: "+code.Value)),
		)
	}
}
