package main

import "github.com/twind-dev/twind/twi"

//go:generate go run github.com/twind-dev/twind/internal/twirgen

const drive, os, testing = "a package-level drive", "os", "testing"

func main() {}

func App(*twi.Runtime) func() twi.Node {
	return func() twi.Node {
		return twi.Element(twi.Class("px-1 text-red-500"), twi.Text(drive))
	}
}
