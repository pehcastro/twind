package drive

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/drive"
	tkonst "github.com/twind-dev/twind/internal/konst/terminal"
	"github.com/twind-dev/twind/twi/buffer"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/terminal"
	"github.com/twind-dev/twind/twi/text"
)

type screen struct {
	cells     *buffer.Buffer
	x, y      int
	pen       buffer.Cell
	clipboard string
}

func (s *screen) Write(p []byte) (int, error) {
	for rest := p; len(rest) > 0; {
		at := bytes.IndexByte(rest, tkonst.CSI[0])
		switch {
		case at != 0:
			if at < 0 {
				at = len(rest)
			}
			if err := s.print(string(rest[:at])); err != nil {
				return 0, err
			}
			rest = rest[at:]
			continue
		case bytes.HasPrefix(rest, []byte(tkonst.ClipboardSet)):
			end := bytes.Index(rest, []byte(tkonst.BEL))
			if end < 0 {
				return 0, fmt.Errorf("drive: the screen got a cut clipboard write %q", rest)
			}
			text, err := base64.StdEncoding.DecodeString(string(rest[len(tkonst.ClipboardSet):end]))
			if err != nil {
				return 0, fmt.Errorf("drive: the screen got a clipboard write that is not base64: %w", err)
			}
			s.clipboard, rest = string(text), rest[end+len(tkonst.BEL):]
			continue
		case !bytes.HasPrefix(rest, []byte(tkonst.CSI)):
			return 0, fmt.Errorf("drive: the screen cannot read the escape in %q", rest)
		}
		end := len(tkonst.CSI)
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '?' {
			end++
		}
		if end == len(rest) {
			return 0, fmt.Errorf("drive: the screen got a cut sequence %q", rest)
		}
		if err := s.control(string(rest[:end+1])); err != nil {
			return 0, err
		}
		rest = rest[end+1:]
	}
	return len(p), nil
}

func (s *screen) print(run string) error {
	if strings.ContainsFunc(run, unicode.IsControl) {
		return fmt.Errorf("drive: the screen got a control character in %q", run)
	}
	for g := range text.Graphemes(run) {
		c := s.pen
		c.Grapheme = g
		if text.Width(g) == 2 {
			c.Width = buffer.Wide
		}
		s.cells.Set(s.x, s.y, c)
		s.x++
		if c.Width == buffer.Wide {
			s.x++
		}
	}
	return nil
}

func (s *screen) control(seq string) error {
	if seq == tkonst.SyncBegin || seq == tkonst.SyncEnd {
		return nil
	}
	params, final := seq[len(tkonst.CSI):len(seq)-1], seq[len(seq)-1]
	var n []int
	for f := range strings.SplitSeq(params, ";") {
		v, err := strconv.Atoi(f)
		if err != nil && f != "" {
			return fmt.Errorf("drive: the screen cannot read %q", seq)
		}
		n = append(n, v)
	}
	switch {
	case final == 'H' && len(n) == 2:
		s.x, s.y = n[1]-1, n[0]-1
	case final == 'C' && len(n) == 1:
		s.x += n[0]
	case final == 'm':
		return s.sgr(n)
	default:
		return fmt.Errorf("drive: the screen cannot read %q", seq)
	}
	return nil
}

func (s *screen) sgr(n []int) error {
	for i := 0; i < len(n); i++ {
		switch code := n[i]; code {
		case tkonst.SGRReset:
			s.pen = buffer.Cell{}
		case tkonst.SGRBold:
			s.pen.Attr |= buffer.Bold
		case tkonst.SGRDim:
			s.pen.Attr |= buffer.Dim
		case tkonst.SGRItalic:
			s.pen.Attr |= buffer.Italic
		case tkonst.SGRUnderline:
			s.pen.Attr |= buffer.Underline
		case tkonst.SGRInverse:
			s.pen.Attr |= buffer.Inverse
		case tkonst.SGRStrike:
			s.pen.Attr |= buffer.Strikethrough
		case tkonst.FgDefault:
			s.pen.Fg = color.Color{}
		case tkonst.FgDefault + tkonst.BgOffset:
			s.pen.Bg = color.Color{}
		case tkonst.FgExtended, tkonst.FgExtended + tkonst.BgOffset:
			if len(n) < i+5 || n[i+1] != tkonst.PaletteRGB {
				return fmt.Errorf("drive: the screen reads only 24-bit colour, got SGR %v", n[i:])
			}
			ink := color.Color{Kind: color.Literal, RGBA: color.RGBA{R: uint8(n[i+2]), G: uint8(n[i+3]), B: uint8(n[i+4]), A: konst.Opaque}}
			if code == tkonst.FgExtended {
				s.pen.Fg = ink
			} else {
				s.pen.Bg = ink
			}
			i += 4
		default:
			return fmt.Errorf("drive: the screen cannot read SGR %d", code)
		}
	}
	return nil
}

type Frame struct{ cells *buffer.Buffer }

func (s *screen) frame() Frame {
	c := buffer.New(s.cells.Width(), s.cells.Height())
	for y := range c.Height() {
		copy(c.Row(y), s.cells.Row(y))
	}
	return Frame{c}
}

func (f Frame) Cells() *buffer.Buffer { return f.cells }

func (f Frame) Text() string {
	var b strings.Builder
	for y := range f.cells.Height() {
		var row strings.Builder
		for _, c := range f.cells.Row(y) {
			if c.Width != buffer.Continuation {
				row.WriteString(c.Grapheme)
			}
		}
		b.WriteString(strings.TrimRight(row.String(), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

func (f Frame) ANSI() string {
	var b strings.Builder
	if err := (&terminal.Writer{Out: &b, Profile: color.TrueColor}).Static(f.cells); err != nil {
		panic(err)
	}
	return b.String()
}
