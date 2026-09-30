package ui

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"

	konst "github.com/twind-dev/twind/internal/konst/paint"
	stylekonst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/internal/render"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/layout"
	"github.com/twind-dev/twind/twi/paint"
	"github.com/twind-dev/twind/twi/raster"
	"github.com/twind-dev/twind/twi/scene"
	"github.com/twind-dev/twind/twi/theme"
)

func TestGlyphUIDrawsOnlyConsoleGlyphsOrStandIns(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range string(src) {
			if r < utf8.RuneSelf {
				continue
			}
			checked++
			if !strings.ContainsRune(konst.ConsoleGlyphs, r) && !strings.ContainsRune(konst.ConsoleMissing, r) {
				t.Errorf("%s draws %q (U+%04X): not in the console fonts and no stand-in", name, r, r)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no glyph outside ASCII read from twi/ui")
	}
	t.Logf("%d non-ASCII glyphs in twi/ui, each safe or with a stand-in", checked)
}

func TestCellLookPageMatchesPixels(t *testing.T) {
	const w, h = 60, 24
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	light := zinc(t, theme.Light)
	rt := twi.New()
	input := NewInput(rt)
	input.Placeholder = "Email"
	on := NewSwitch(rt)
	on.Checked = true
	page := twi.Element(twi.Class("flex flex-col gap-1 p-2 w-full h-full bg-blue-600"),
		Card(twi.Class("py-1"),
			CardHeader(CardTitle(twi.Text("Card")), CardDescription(twi.Text("A description"))),
			CardContent(twi.Text("Content inside the card")),
			CardFooter(Button(Default, SizeSM, twi.Text("Save")), Button(Outline, SizeSM, twi.Text("Cancel"))),
		),
		twi.Element(twi.Class("flex flex-row gap-2 items-center"), Badge(Secondary, twi.Text("Badge")), Kbd(twi.Text("Ctrl")), on.Node()),
		input.Node(),
		Alert(Default, AlertTitle(twi.Text("Heads up")), AlertDescription(twi.Text("An alert with a border"))),
	)
	tree := rendered(page)
	built := reflect.NewAt(tree.Type(), unsafe.Pointer(tree.UnsafeAddr())).Elem().Interface().(render.Node)
	frame := render.Frame{Sheet: sheet.WithTheme(&light), Width: w, Height: layout.Length{Unit: layout.Cells, Value: h}}
	root, err := render.Scene(built, frame)
	if err != nil {
		t.Fatal(err)
	}
	frame.Graphics, frame.Cell = true, image.Pt(9, 19)
	pixelRoot, err := render.Scene(built, frame)
	if err != nil {
		t.Fatal(err)
	}
	var boxes func(a, b *scene.Node, path string)
	boxes = func(a, b *scene.Node, path string) {
		if a.Bounds != b.Bounds || a.Padding != b.Padding || a.Content != b.Content || len(a.Children) != len(b.Children) {
			t.Errorf("box %s: %v without graphics, %v with a 9x19 cell and graphics", path, a.Bounds, b.Bounds)
			return
		}
		for i := range a.Children {
			boxes(&a.Children[i], &b.Children[i], path+"/"+strconv.Itoa(i))
		}
	}
	boxes(&root, &pixelRoot, "root")
	cell := image.Pt(stylekonst.NominalCellX, stylekonst.NominalCellY)
	var f scene.Frame
	f.Record(&root, cell)
	pixels := image.NewRGBA(image.Rect(0, 0, w*cell.X, h*cell.Y))
	for _, l := range f.Layers {
		if l.Origin != (image.Point{}) || l.Opacity != 1 {
			t.Fatalf("a layer at %v with opacity %v: the page should paint in one plane", l.Origin, l.Opacity)
		}
		new(raster.Raster).Draw(pixels, l.Ops, pixels.Rect)
	}
	cells := buffer.New(w, h)
	paint.Paint(cells, root, paint.Composited)
	seen := func(c buffer.Cell) color.RGBA {
		if r, _ := utf8.DecodeRuneInString(c.Grapheme); !strings.ContainsRune(konst.HalfEdges+konst.PillCaps, r) || c.Fg.Kind != color.Literal {
			return c.Bg.RGBA
		}
		avg := func(a, b uint8) uint8 { return uint8((int(a) + int(b)) / 2) }
		return color.RGBA{R: avg(c.Fg.RGBA.R, c.Bg.RGBA.R), G: avg(c.Fg.RGBA.G, c.Bg.RGBA.G), B: avg(c.Fg.RGBA.B, c.Bg.RGBA.B), A: math.MaxUint8}
	}
	apart := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	wrong := 0
	for y := range h {
		for x := range w {
			got, want := seen(cells.At(x, y)), raster.Mean(pixels, image.Rect(x*cell.X, y*cell.Y, (x+1)*cell.X, (y+1)*cell.Y))
			if max(apart(got.R, want.R), apart(got.G, want.G), apart(got.B, want.B)) > math.MaxUint8/2 {
				wrong++
				t.Errorf("cell %d,%d %q: the cell look shows %v, the pixels %v", x, y, cells.At(x, y).Grapheme, got, want)
			}
		}
	}
	rows := make([]string, h)
	for y := range h {
		var line strings.Builder
		for _, c := range cells.Row(y) {
			line.WriteString(c.Grapheme)
		}
		rows[y] = line.String()
	}
	t.Logf("%d of %d cells differ from the pixel look; cell look, %dx%d:\n%s", wrong, w*h, w, h, strings.Join(rows, "\n"))
}
