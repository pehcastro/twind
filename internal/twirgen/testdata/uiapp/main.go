package main

import (
	"fmt"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

//go:generate go run github.com/twind-dev/twind/internal/twirgen

func main() {
	sheet, err := Styles()
	if err != nil {
		panic(err)
	}
	fmt.Print(twi.RenderString(ui.Button(ui.Destructive, ui.SizeDefault, twi.Text("Delete")), twi.Styles(sheet)))
}
