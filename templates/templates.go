package templates

import "embed"

//go:embed app/.gitignore.tmpl app/README.md.tmpl app/go.mod.tmpl app/main.go app/main_test.go app/twir_gen.go app/testdata/demo.twd
var App embed.FS
