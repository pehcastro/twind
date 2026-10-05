package main

import (
	"fmt"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/ui"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func main() {
	sheet, err := Styles()
	if err != nil {
		panic(err)
	}
	out, err := twi.RenderString(ui.Button(ui.ButtonDestructive, ui.ButtonSizeDefault, twi.Text("Delete")), twi.Styles(sheet))
	if err != nil {
		panic(err)
	}
	fmt.Print(out)
}
