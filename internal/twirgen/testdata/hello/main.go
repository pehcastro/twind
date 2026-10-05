package main

import (
	"fmt"
	"strings"

	"github.com/pehcastro/twind/twi/style"
)

//go:generate go run github.com/pehcastro/twind/internal/twirgen

func main() {
	sheet, err := Styles()
	if err != nil {
		panic(err)
	}
	s := sheet.Compute(style.ComputedStyle{}, strings.Fields("flex flex-col gap-2 p-4 bg-zinc-950 text-zinc-100 border rounded-lg p-2"))
	fmt.Printf("Display=%d Direction=%d Gap=%v Padding=%v Border=%v BorderColor=%v Radius=%d Background=%v Color=%v\n",
		s.Display, s.Direction, s.RowGap, s.Padding, s.BorderWidth, s.BorderColor, s.Radius, s.Background.RGBA, s.Color.RGBA)
	card := sheet.Compute(style.ComputedStyle{}, strings.Fields("bg-card shadow-md inset-shadow-sm bg-linear-to-r from-indigo-600 via-40% to-pink-600"))
	fmt.Printf("Card=%v Shadows=%+v Inset=%+v Gradient=%+v\n", card.Background.RGBA, card.Shadows, card.InsetShadows, card.Gradient)
}
