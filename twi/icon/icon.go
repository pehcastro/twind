package icon

//go:generate go run ../../internal/icongen -out icons_gen.go

func (n Name) String() string {
	name, _, _ := n.lucide()
	return name
}

func (n Name) Glyph() rune {
	_, glyph, _ := n.lucide()
	return glyph
}
