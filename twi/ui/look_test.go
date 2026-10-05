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

	konst "github.com/pehcastro/twind/internal/konst/paint"
	stylekonst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/layout"
	"github.com/pehcastro/twind/twi/paint"
	"github.com/pehcastro/twind/twi/raster"
	"github.com/pehcastro/twind/twi/scene"
	"github.com/pehcastro/twind/twi/theme"
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
	glyph := func(x, y int) rune {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0
		}
		r, _ := utf8.DecodeRuneInString(cells.At(x, y).Grapheme)
		return r
	}
	corners := map[rune]int{'╭': -1, '╰': -1, '╮': 1, '╯': 1}
	ring := func(x, y int) bool {
		switch r := glyph(x, y); r {
		case '▄', '▀':
			return true
		case '▐', '▌':
			edge := func(n rune) bool { _, corner := corners[n]; return n == r || corner }
			return edge(glyph(x, y-1)) || edge(glyph(x, y+1))
		}
		return false
	}
	seen := func(x, y int) color.RGBA {
		c := cells.At(x, y)
		if r := glyph(x, y); !strings.ContainsRune(konst.HalfEdges, r) || c.Fg.Kind != color.Literal {
			return c.Bg.RGBA
		}
		if ring(x, y) {
			return c.Fg.RGBA
		}
		avg := func(a, b uint8) uint8 { return uint8((int(a) + int(b)) / 2) }
		return color.RGBA{R: avg(c.Fg.RGBA.R, c.Bg.RGBA.R), G: avg(c.Fg.RGBA.G, c.Bg.RGBA.G), B: avg(c.Fg.RGBA.B, c.Bg.RGBA.B), A: math.MaxUint8}
	}
	apart := func(a, b uint8) int { return max(int(a)-int(b), int(b)-int(a)) }
	wrong, rings, frames := 0, 0, 0
	for y := range h {
		for x := range w {
			if side, ok := corners[glyph(x, y)]; ok {
				if outside := cells.At(x+side, y).Bg.RGBA; cells.At(x, y).Bg.RGBA != outside {
					t.Errorf("corner %d,%d %q: bg %v, want the parent's %v", x, y, cells.At(x, y).Grapheme, cells.At(x, y).Bg.RGBA, outside)
				}
				continue
			}
			if r := glyph(x, y); r == '─' || r == '│' {
				cx, cy := x, y
				for glyph(cx, cy) == r {
					if r == '─' {
						cx--
					} else {
						cy--
					}
				}
				if _, ok := corners[glyph(cx, cy)]; ok {
					frames++
					if line, corner := cells.At(x, y).Bg.RGBA, cells.At(cx, cy).Bg.RGBA; line != corner {
						t.Errorf("frame line %d,%d %q: bg %v, want the parent's %v as at its corner; the fill stays inside the line", x, y, cells.At(x, y).Grapheme, line, corner)
					}
					continue
				}
			}
			if ring(x, y) {
				rings++
			}
			got, want := seen(x, y), raster.Mean(pixels, image.Rect(x*cell.X, y*cell.Y, (x+1)*cell.X, (y+1)*cell.Y))
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
	if frames == 0 {
		t.Error("no frame line found: the card and the alert should draw rounded box-drawing frames")
	}
	t.Logf("%d of %d cells differ from the pixel look, %d frame line cells on the parent's colour, %d ring cells compared by their inner half; cell look, %dx%d:\n%s", wrong, w*h, frames, rings, w, h, strings.Join(rows, "\n"))
}

func TestCellLookCardAndBadgeDriven(t *testing.T) {
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []theme.Scheme{theme.Dark, theme.Light} {
		th := zinc(t, scheme)
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			rt.SetTheme(th)
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col items-start gap-1 p-2 w-full h-full bg-background text-foreground"),
					Card(twi.Class("w-20"), CardContent(twi.Text("text"))),
					Badge(Secondary, twi.Text("Badge")),
				)
			}
		}, drive.Size(80, 24), drive.Styles(sheet))
		f := d.Frame()
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(f.Text(), "\n")
		t.Logf("scheme %d, 80x24:\n%s", scheme, f.Text())
		if strings.ContainsAny(f.Text(), "▄▌▀▐") {
			t.Errorf("scheme %d: a half-block sliver in the frame", scheme)
		}
		spot := func(s string) (x, y int) {
			for y, l := range lines {
				if before, _, ok := strings.Cut(l, s); ok {
					return utf8.RuneCountInString(before), y
				}
			}
			t.Fatalf("scheme %d: no %q in the frame:\n%s", scheme, s, f.Text())
			return 0, 0
		}
		left, top := spot("╭")
		_, bottom := spot("╰")
		right := left + 19
		page, card := th.Tokens[theme.Background].RGBA, th.Tokens[theme.Card].RGBA
		for y := top; y <= bottom; y++ {
			row := []rune(lines[y])
			edge := map[int][2]rune{top: {'╭', '╮'}, bottom: {'╰', '╯'}}[y]
			if edge == ([2]rune{}) {
				edge = [2]rune{'│', '│'}
			}
			if row[left] != edge[0] || row[right] != edge[1] {
				t.Errorf("scheme %d row %d: %q, want it framed by %q", scheme, y, lines[y], string(edge[:]))
			}
			for x := left; x <= right; x++ {
				want := card
				if x == left || x == right || y == top || y == bottom {
					want = page
				}
				if got := f.Cells().At(x, y).Bg.RGBA; got != want {
					t.Errorf("scheme %d cell %d,%d %q: bg %v, want %v; the fill stays inside the line", scheme, x, y, string(row[x]), got, want)
				}
			}
		}
		bx, by := spot("Badge")
		secondary := th.Tokens[theme.Secondary].RGBA
		for x := bx - 2; x <= bx+len("Badge")+1; x++ {
			want := secondary
			if x == bx-2 || x == bx+len("Badge")+1 {
				want = page
			}
			if got := f.Cells().At(x, by).Bg.RGBA; got != want {
				t.Errorf("scheme %d badge cell %d: bg %v, want %v; the pill is its background across the box, nothing outside", scheme, x, got, want)
			}
		}
	}
}
