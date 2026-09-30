package markdown

import (
	"image"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/theme"
)

const corners = "╭╮╰╯┌┐└┘"

func TestCodeBlockOneRing(t *testing.T) {
	for _, c := range []struct {
		name, src, want string
		label           bool
	}{
		{"with a language", "```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n", "func main() {\n    fmt.Println(\"hi\")\n}", true},
		{"without a language", "```\nplain text\n```\n", "plain text", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			var copied []string
			d := driven(t, c.src, &Options{Copy: func(code string) { copied = append(copied, code) }})
			frame := d.Frame().Text()
			t.Logf("80x30 frame:\n%s", frame)
			if top, bottom := strings.Count(frame, "╭")+strings.Count(frame, "┌"), strings.Count(frame, "╰")+strings.Count(frame, "└"); top != 1 || bottom != 1 {
				t.Fatalf("%d top and %d bottom corners, want one ring around the code block", top, bottom)
			}
			lines := strings.Split(frame, "\n")
			top := slices.IndexFunc(lines, func(l string) bool { return strings.ContainsAny(l, "╭┌") })
			bottom := slices.IndexFunc(lines, func(l string) bool { return strings.ContainsAny(l, "╰└") })
			rows := lines[top : bottom+1]
			cells, page := d.Frame().Cells(), d.Frame().Cells().At(0, bottom+1).Bg
			for y := top; y <= bottom; y++ {
				for x, cell := range cells.Row(y) {
					if strings.ContainsAny(cell.Grapheme, "─│"+corners) && cell.Bg != page {
						t.Fatalf("the ring glyph %q at %d,%d sits on %v, the page is %v: a fill band outside the line reads as a second frame", cell.Grapheme, x, y, cell.Bg, page)
					}
				}
			}
			for i, row := range rows[1 : len(rows)-1] {
				inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(row), "│"), "│")
				if strings.ContainsAny(inner, "─│├┤┬┴┼"+corners) {
					t.Errorf("inner row %d draws a line inside the ring: %q", i+1, row)
				}
			}
			if first := rows[1]; c.label != strings.Contains(first, " go ") || !strings.Contains(first, "Copy") {
				t.Errorf("first row inside the ring %q: want the copy area, and the language label %v", first, c.label)
			}
			if code := strings.Index(frame, strings.Split(c.want, "\n")[0]); code < strings.Index(frame, "Copy") {
				t.Errorf("the code is missing or above the copy area")
			}
			for y, line := range lines {
				if x := strings.Index(line, "Copy"); x >= 0 {
					d.Click(len([]rune(line[:x])), y)
					break
				}
			}
			if len(copied) != 1 || copied[0] != c.want {
				t.Errorf("the copy area copied %q, want the raw source %q once", copied, c.want)
			}
		})
	}
}

func TestTableFrame(t *testing.T) {
	d := driven(t, "| Block | Drawn as |\n| :---- | -------: |\n| Code | muted surface |\n| Table | bordered grid |\n\nAfter.\n", &Options{})
	frame := d.Frame().Text()
	t.Logf("80x30 frame:\n%s", frame)
	if strings.ContainsAny(frame, "│"+corners) {
		t.Errorf("the table draws a ring; want rules between rows only")
	}
	var shape []string
	for _, line := range strings.Split(frame, "\n") {
		switch line = strings.TrimSpace(line); {
		case line == "":
		case strings.Trim(line, "─") == "":
			shape = append(shape, "rule")
		default:
			shape = append(shape, strings.Fields(line)[0])
		}
	}
	if got, want := strings.Join(shape, " "), "Block rule Code rule Table rule After."; got != want {
		t.Errorf("rows read %q, want %q", got, want)
	}
}

func TestFrameBoxesSameWithGraphics(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	var zinc theme.Theme
	for _, th := range theme.Builtin() {
		if th.Name == "zinc" && th.Scheme == theme.Dark {
			zinc = th
		}
	}
	page, err := Parse("sample.md", sample, callout())
	if err != nil {
		t.Fatal(err)
	}
	var built render.Node
	var find func(reflect.Value) bool
	find = func(v reflect.Value) bool {
		switch {
		case v.Type() == reflect.TypeFor[render.Node]():
			built = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Interface().(render.Node)
			return true
		case v.Kind() == reflect.Pointer && !v.IsNil():
			return find(v.Elem())
		case v.Kind() == reflect.Struct:
			for i := range v.NumField() {
				if find(v.Field(i)) {
					return true
				}
			}
		}
		return false
	}
	root := Render(page, Options{Copy: func(string) {}})
	if !find(reflect.ValueOf(&root).Elem()) {
		t.Fatal("a twi.Node holds no render.Node")
	}
	frame := render.Frame{Sheet: sheet.WithTheme(&zinc), Width: 60, Height: layout.Length{Unit: layout.Cells, Value: 40}}
	cells, err := render.Scene(built, frame)
	if err != nil {
		t.Fatal(err)
	}
	frame.Graphics, frame.Cell = true, image.Pt(9, 19)
	pixels, err := render.Scene(built, frame)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	var boxes func(a, b *scene.Node, path string)
	boxes = func(a, b *scene.Node, path string) {
		checked++
		if a.Bounds != b.Bounds || a.Padding != b.Padding || a.Content != b.Content || len(a.Children) != len(b.Children) {
			t.Errorf("box %s: %v without graphics, %v with a 9x19 cell and graphics", path, a.Bounds, b.Bounds)
			return
		}
		for i := range a.Children {
			boxes(&a.Children[i], &b.Children[i], path+"/"+strconv.Itoa(i))
		}
	}
	boxes(&cells, &pixels, "root")
	t.Logf("%d boxes compared", checked)
}
