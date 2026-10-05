package terminal

import (
	"encoding/base64"
	"io"
	"strconv"
	"strings"
	"unicode"

	konst "github.com/pehcastro/twind/internal/konst/terminal"
	"github.com/pehcastro/twind/twi/buffer"
	"github.com/pehcastro/twind/twi/color"
)

type Writer struct {
	Out     io.Writer
	Profile color.Profile
	Sync    bool

	buf         []byte
	runs        []buffer.Run
	pen         pen
	penKnown    bool
	x, y        int
	cursorKnown bool
}

type ink struct {
	set   bool
	index uint8
	rgb   [3]uint8
}

type pen struct {
	fg, bg ink
	attr   buffer.Attr
}

func Clipboard(text string) []byte {
	return append(base64.StdEncoding.AppendEncode([]byte(konst.ClipboardSet), []byte(text)), konst.BEL...)
}

func (w *Writer) Diff(prev, cur *buffer.Buffer) error {
	w.runs = buffer.Diff(w.runs[:0], prev, cur)
	return w.Runs(cur, w.runs)
}

func (w *Writer) Runs(cur *buffer.Buffer, runs []buffer.Run) error {
	if len(runs) == 0 {
		return nil
	}
	w.buf = w.buf[:0]
	if w.Sync {
		w.buf = append(w.buf, konst.SyncBegin...)
	}
	for _, r := range runs {
		w.move(r.X, r.Y)
		for _, c := range cur.Row(r.Y)[r.X : r.X+r.Len] {
			w.cell(c)
		}
	}
	if w.Sync {
		w.buf = append(w.buf, konst.SyncEnd...)
	}
	return w.flush()
}

func (w *Writer) Erase(dst []byte, x, y, n int, bg color.Color) []byte {
	w.buf = dst
	w.style(buffer.Cell{Bg: bg})
	w.move(x, y)
	w.buf = append(w.buf, konst.CSI...)
	w.param(n)
	w.buf[len(w.buf)-1] = 'X'
	dst, w.buf = w.buf, nil
	return dst
}

func (w *Writer) Static(b *buffer.Buffer) error {
	w.buf = w.buf[:0]
	w.pen, w.penKnown = pen{}, true
	for y := range b.Height() {
		row := b.Row(y)
		end := len(row)
		for end > 0 && w.blank(row[end-1]) {
			end--
		}
		for _, c := range row[:end] {
			w.cell(c)
		}
		if w.pen != (pen{}) {
			w.buf = append(w.buf, konst.Reset...)
			w.pen = pen{}
		}
		w.buf = append(w.buf, '\n')
	}
	w.cursorKnown = false
	return w.flush()
}

func (w *Writer) blank(c buffer.Cell) bool {
	space := c.Grapheme == " "
	visible := c.Attr&(buffer.Underline|buffer.Strikethrough|buffer.Inverse) != 0
	painted := c.Bg.Kind == color.Current || c.Bg.Kind == color.Literal && c.Bg.RGBA.A != 0
	switch w.Profile {
	case color.None:
		return space
	case color.Attributes:
		return space && !visible
	case color.ANSI16, color.ANSI256, color.TrueColor:
		return space && !visible && !painted
	}
	panic("terminal: unknown colour profile")
}

func (w *Writer) flush() error {
	_, err := w.Out.Write(w.buf)
	if err != nil {
		w.penKnown, w.cursorKnown = false, false
	}
	return err
}

func (w *Writer) move(x, y int) {
	switch {
	case w.cursorKnown && y == w.y && x == w.x:
	case w.cursorKnown && y == w.y && x > w.x:
		w.buf = append(w.buf, konst.CSI...)
		w.param(x - w.x)
		w.buf[len(w.buf)-1] = 'C'
	default:
		w.buf = append(w.buf, konst.CSI...)
		w.param(y+1, x+1)
		w.buf[len(w.buf)-1] = 'H'
	}
	w.x, w.y, w.cursorKnown = x, y, true
}

func (w *Writer) cell(c buffer.Cell) {
	if c.Width == buffer.Continuation {
		return
	}
	w.style(c)
	switch {
	case c.Grapheme != "" && !strings.ContainsFunc(c.Grapheme, unicode.IsControl):
		w.buf = append(w.buf, c.Grapheme...)
	case c.Width == buffer.Wide:
		w.buf = append(w.buf, "  "...)
	default:
		w.buf = append(w.buf, ' ')
	}
	w.x++
	if c.Width == buffer.Wide {
		w.x++
	}
}

func (w *Writer) style(c buffer.Cell) {
	if w.Profile == color.None {
		return
	}
	next := pen{fg: w.ink(c.Fg), bg: w.ink(c.Bg), attr: c.Attr}
	if c.Bg.Kind == color.Current {
		next.bg = next.fg
	}
	if w.penKnown && next == w.pen {
		return
	}
	from := w.pen
	w.buf = append(w.buf, konst.CSI...)
	if !w.penKnown || from.attr&^next.attr != 0 {
		w.param(konst.SGRReset)
		from = pen{}
	}
	for _, a := range [...]struct {
		attr buffer.Attr
		code int
	}{
		{buffer.Bold, konst.SGRBold},
		{buffer.Dim, konst.SGRDim},
		{buffer.Italic, konst.SGRItalic},
		{buffer.Underline, konst.SGRUnderline},
		{buffer.Inverse, konst.SGRInverse},
		{buffer.Strikethrough, konst.SGRStrike},
	} {
		if next.attr&^from.attr&a.attr != 0 {
			w.param(a.code)
		}
	}
	if next.fg != from.fg {
		w.inkParams(next.fg, 0)
	}
	if next.bg != from.bg {
		w.inkParams(next.bg, konst.BgOffset)
	}
	w.buf[len(w.buf)-1] = 'm'
	w.pen, w.penKnown = next, true
}

func (w *Writer) ink(c color.Color) ink {
	if c.Kind != color.Literal || c.RGBA.A == 0 {
		return ink{}
	}
	switch w.Profile {
	case color.TrueColor:
		return ink{set: true, rgb: [3]uint8{c.RGBA.R, c.RGBA.G, c.RGBA.B}}
	case color.ANSI256:
		return ink{set: true, index: c.RGBA.ANSI256()}
	case color.ANSI16:
		return ink{set: true, index: c.RGBA.ANSI16()}
	case color.None, color.Attributes:
		return ink{}
	}
	panic("terminal: unknown colour profile")
}

func (w *Writer) inkParams(i ink, offset int) {
	switch {
	case !i.set:
		w.param(konst.FgDefault + offset)
	case w.Profile == color.TrueColor:
		w.param(konst.FgExtended+offset, konst.PaletteRGB, int(i.rgb[0]), int(i.rgb[1]), int(i.rgb[2]))
	case w.Profile == color.ANSI256:
		w.param(konst.FgExtended+offset, konst.Palette256, int(i.index))
	case i.index < konst.BaseColors:
		w.param(konst.FgBase + offset + int(i.index))
	default:
		w.param(konst.FgBrightBase + offset + int(i.index) - konst.BaseColors)
	}
}

func (w *Writer) param(codes ...int) {
	for _, n := range codes {
		w.buf = strconv.AppendInt(w.buf, int64(n), 10)
		w.buf = append(w.buf, ';')
	}
}
