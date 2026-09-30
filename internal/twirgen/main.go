package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi/tailwind"
)

func main() {
	out := flag.String("o", konst.GeneratedFile, "generated Go file, in the current directory")
	pkg := flag.String("pkg", os.Getenv("GOPACKAGE"), "package name of the generated file")
	name := flag.String("func", "Styles", "name of the generated function returning the sheet")
	bin := flag.String("tailwind", "", "Tailwind executable; default .twind/bin in this or a parent directory")
	check := flag.Bool("check", false, "report whether the generated file is stale, without Tailwind")
	flag.Parse()
	if *check {
		stale, err := tailwind.Stale(".", *out)
		if err != nil {
			fail(err)
		}
		if stale {
			fmt.Println("stale:", *out)
			os.Exit(1)
		}
		fmt.Println("fresh:", *out)
		return
	}
	if err := generate(*out, *pkg, *name, *bin); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "twirgen:", err)
	os.Exit(1)
}

func generate(out, pkg, name, bin string) error {
	if pkg == "" {
		return errors.New("no package name: run through go generate or pass -pkg")
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if bin == "" {
		if bin, err = findTailwind(dir); err != nil {
			return err
		}
	}
	help, _ := exec.Command(bin, "--help").CombinedOutput()
	if version := regexp.MustCompile(`v(\d+\.\d+\.\d+)`).FindSubmatch(help); version == nil || string(version[1]) != konst.TailwindVersion {
		return fmt.Errorf("%s is not Tailwind %s", bin, konst.TailwindVersion)
	}
	sources, hash, err := tailwind.Inputs(dir, out)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "twirgen")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	input, output := filepath.Join(tmp, "input.css"), filepath.Join(tmp, "output.css")
	if err := os.WriteFile(input, []byte(tailwind.Input(sources)), 0o600); err != nil {
		return err
	}
	if msg, err := exec.Command(bin, "-i", input, "-o", output).CombinedOutput(); err != nil {
		return fmt.Errorf("tailwind: %w\n%s", err, msg)
	}
	css, err := os.ReadFile(output)
	if err != nil {
		return err
	}
	rules, warnings, err := tailwind.Compile(string(css))
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "twirgen: warning:", w)
	}
	consts := map[string]string{}
	for _, path := range []string{"github.com/twind-dev/twind/twi/style", "github.com/twind-dev/twind/twi/color", "github.com/twind-dev/twind/twi/theme"} {
		p, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(path)
		if err != nil {
			return err
		}
		for _, n := range p.Scope().Names() {
			if c, ok := p.Scope().Lookup(n).(*types.Const); ok && c.Exported() {
				consts[c.Type().String()+"="+c.Val().ExactString()] = p.Name() + "." + n
			}
		}
	}
	body := literal(reflect.ValueOf(rules), consts)
	var src bytes.Buffer
	src.WriteString(tailwind.Header(hash) + "package " + pkg + "\n\nimport (\n")
	for _, dep := range []string{"color", "theme"} {
		if strings.Contains(body, dep+".") {
			src.WriteString("\"github.com/twind-dev/twind/twi/" + dep + "\"\n")
		}
	}
	fmt.Fprintf(&src, "\"github.com/twind-dev/twind/twi/style\"\n)\n\nfunc %s() (style.Sheet, error) {\nreturn style.NewSheet(%d, %s)\n}\n", name, konst.IRVersion, body)
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, out), formatted, 0o644)
}

func findTailwind(dir string) (string, error) {
	exe := "tailwindcss-" + runtime.GOOS + "-" + map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	for {
		path := filepath.Join(dir, ".twind", "bin", exe)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no .twind/bin/" + exe + " here or above; pass -tailwind")
		}
		dir = parent
	}
}

func literal(v reflect.Value, names map[string]string) string {
	var number string
	switch v.Kind() {
	case reflect.Struct:
		var fields []string
		for i := range v.NumField() {
			if !v.Field(i).IsZero() {
				fields = append(fields, v.Type().Field(i).Name+": "+literal(v.Field(i), names))
			}
		}
		return v.Type().String() + "{" + strings.Join(fields, ", ") + "}"
	case reflect.Slice:
		var items strings.Builder
		for i := range v.Len() {
			items.WriteString("\n" + literal(v.Index(i), names) + ",")
		}
		return v.Type().String() + "{" + items.String() + "\n}"
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Uint8:
		number = strconv.FormatUint(v.Uint(), 10)
	case reflect.Int, reflect.Int64:
		number = strconv.FormatInt(v.Int(), 10)
	default:
		panic("twirgen: cannot write a " + v.Kind().String())
	}
	if name, ok := names[v.Type().PkgPath()+"."+v.Type().Name()+"="+number]; ok {
		return name
	}
	return number
}
