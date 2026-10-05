package tailwind

import "github.com/pehcastro/twind/internal/tailwind"

func Stale(dir, generated string) (bool, error) { return tailwind.Stale(dir, generated) }
