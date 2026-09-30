package main

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/icon"
)

func list() [][2]string {
	return [][2]string{
		{"activity", "∿"}, {"archive", "▤"}, {"arrow-down", "↓"}, {"arrow-left", "←"}, {"arrow-right", "→"},
		{"arrow-up", "↑"}, {"arrow-up-down", "⇅"}, {"arrow-up-right", "⇗"}, {"audio-waveform", "∿"}, {"badge-check", "✓"},
		{"bell", "♪"}, {"blocks", "▦"}, {"bold", "B"}, {"book-open", "▯"}, {"bookmark", "▯"},
		{"bot", "☻"}, {"bug", "¤"}, {"calculator", "▦"}, {"calendar", "▦"}, {"calendar-days", "▦"},
		{"chart-pie", "◔"}, {"check", "✓"}, {"chevron-down", "⌄"}, {"chevron-left", "‹"}, {"chevron-right", "›"},
		{"chevron-up", "⌃"}, {"chevrons-left", "«"}, {"chevrons-right", "»"}, {"chevrons-up-down", "⇕"}, {"circle", "○"},
		{"circle-alert", "!"}, {"circle-check", "✓"}, {"circle-dot", "◉"}, {"circle-question-mark", "?"}, {"circle-x", "⊗"},
		{"clock", "◷"}, {"cloud", "◠"}, {"code", "‹"}, {"command", "⌘"}, {"copy", "❐"},
		{"credit-card", "▭"}, {"database", "≡"}, {"dot", "·"}, {"download", "↓"}, {"ellipsis", "…"},
		{"ellipsis-vertical", "⋮"}, {"external-link", "⇗"}, {"eye", "◉"}, {"eye-off", "○"}, {"file", "▯"},
		{"file-text", "≡"}, {"folder", "▭"}, {"folder-open", "▭"}, {"frame", "#"}, {"funnel", "▽"},
		{"gallery-vertical-end", "▤"}, {"git-branch", "⑂"}, {"globe", "◍"}, {"grip-horizontal", "⠶"}, {"grip-vertical", "⠿"},
		{"heart", "◆"}, {"house", "⌂"}, {"image", "▣"}, {"inbox", "▭"}, {"info", "i"},
		{"italic", "I"}, {"key", "⚷"}, {"keyboard", "▤"}, {"layout-grid", "▦"}, {"life-buoy", "◎"},
		{"link", "∞"}, {"list", "≡"}, {"list-filter", "≡"}, {"loader", "✻"}, {"loader-circle", "◌"},
		{"lock", "▣"}, {"log-in", "→"}, {"log-out", "←"}, {"mail", "⊠"}, {"map", "▱"},
		{"maximize", "□"}, {"menu", "≡"}, {"message-circle", "○"}, {"minimize", "⊟"}, {"minus", "−"},
		{"moon", "☾"}, {"move", "✥"}, {"panel-left", "◧"}, {"pause", "‖"}, {"pencil", "∕"},
		{"play", "►"}, {"plus", "+"}, {"redo", "↷"}, {"refresh-cw", "↻"}, {"rocket", "↑"},
		{"rotate-ccw", "↺"}, {"save", "▣"}, {"search", "⌕"}, {"send", "➤"}, {"server", "▤"},
		{"settings", "✱"}, {"settings-2", "≡"}, {"share", "⇗"}, {"shield", "◊"}, {"slash", "/"},
		{"sparkles", "✦"}, {"square", "□"}, {"square-terminal", "›"}, {"star", "☆"}, {"strikethrough", "S"},
		{"sun", "☼"}, {"tag", "◇"}, {"terminal", "›"}, {"trash", "▯"}, {"trending-down", "⇘"},
		{"trending-up", "⇗"}, {"triangle-alert", "△"}, {"type", "T"}, {"underline", "U"}, {"undo", "↶"},
		{"upload", "↑"}, {"user", "☻"}, {"users", "☻"}, {"x", "✕"}, {"zap", "ϟ"},
		{"zoom-in", "+"}, {"zoom-out", "−"},
	}
}

type point struct{ x, y float64 }

type path struct {
	out        strings.Builder
	at, start  point
	ctrl       point
	lastCubic  bool
	lastQuad   bool
	quadHandle point
}

func main() {
	lucide := flag.String("lucide", "", "lucide icons directory")
	out := flag.String("out", "icons_gen.go", "output file")
	flag.Parse()
	icons := list()
	var src bytes.Buffer
	src.WriteString("package icon\n\nimport \"strconv\"\n\ntype Name uint16\n\nconst (\n")
	for i, icon := range icons {
		fmt.Fprintf(&src, "\t%s Name = %d\n", ident(icon[0]), i+1)
	}
	fmt.Fprintf(&src, "\tCount Name = %d\n)\n\nfunc (n Name) lucide() (name string, glyph rune, path string) {\n\tswitch n {\n", len(icons))
	for _, icon := range icons {
		d, err := read(filepath.Join(*lucide, icon[0]+".svg"))
		if err != nil {
			log.Fatalf("%s: %v", icon[0], err)
		}
		fmt.Fprintf(&src, "\tcase %s:\n\t\treturn %q, %q, %q\n", ident(icon[0]), icon[0], []rune(icon[1])[0], d)
	}
	src.WriteString("\t}\n\tpanic(\"icon: unknown name \" + strconv.Itoa(int(n)))\n}\n")
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, formatted, 0o644); err != nil {
		log.Fatal(err)
	}
}

func ident(name string) string {
	var b strings.Builder
	for part := range strings.SplitSeq(name, "-") {
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	return b.String()
}

func read(file string) (string, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	p := &path{}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		text := func(name string) string {
			for _, at := range el.Attr {
				if at.Name.Local == name {
					return at.Value
				}
			}
			return ""
		}
		var bad error
		a := func(name string) float64 {
			v, err := strconv.ParseFloat(cmp.Or(text(name), "0"), 64)
			bad = cmp.Or(bad, err)
			return v
		}
		switch el.Name.Local {
		case "svg":
			if text("viewBox") != "0 0 24 24" || text("stroke-width") != "2" || text("fill") != "none" {
				return "", fmt.Errorf("unexpected svg attributes %v", el.Attr)
			}
		case "path":
			if err := p.parse(text("d")); err != nil {
				return "", err
			}
		case "circle":
			p.ellipse(a("cx"), a("cy"), a("r"), a("r"))
		case "ellipse":
			p.ellipse(a("cx"), a("cy"), a("rx"), a("ry"))
		case "line":
			p.move(point{a("x1"), a("y1")})
			p.line(point{a("x2"), a("y2")})
		case "polyline", "polygon":
			var v []float64
			for f := range strings.FieldsFuncSeq(text("points"), func(r rune) bool { return r == ' ' || r == ',' }) {
				n, err := strconv.ParseFloat(f, 64)
				v, bad = append(v, n), cmp.Or(bad, err)
			}
			p.move(point{v[0], v[1]})
			for i := 2; i+1 < len(v); i += 2 {
				p.line(point{v[i], v[i+1]})
			}
			if el.Name.Local == "polygon" {
				p.close()
			}
		case "rect":
			rx, ry := a("rx"), a("ry")
			if text("rx") == "" {
				rx = ry
			}
			if text("ry") == "" {
				ry = rx
			}
			p.rect(a("x"), a("y"), a("width"), a("height"), rx, ry)
		default:
			return "", fmt.Errorf("unsupported element %s", el.Name.Local)
		}
		if bad != nil {
			return "", fmt.Errorf("%s: %w", el.Name.Local, bad)
		}
	}
	return strings.TrimSpace(p.out.String()), nil
}

func (p *path) emit(op byte, pts ...point) {
	p.out.WriteString(" " + string(op))
	for _, q := range pts {
		for _, v := range [2]float64{q.x, q.y} {
			p.out.WriteString(" " + strconv.FormatFloat(math.Round(v*konst.Precision)/konst.Precision, 'f', -1, 64))
		}
	}
}

func (p *path) move(to point) {
	p.emit('M', to)
	p.at, p.start = to, to
}

func (p *path) line(to point) {
	p.emit('L', to)
	p.at = to
}

func (p *path) cubic(c1, c2, to point) {
	p.emit('C', c1, c2, to)
	p.at, p.ctrl = to, c2
}

func (p *path) close() {
	p.emit('Z')
	p.at = p.start
}

func (p *path) ellipse(cx, cy, rx, ry float64) {
	p.move(point{cx + rx, cy})
	p.arc(rx, ry, 0, false, true, point{cx - rx, cy})
	p.arc(rx, ry, 0, false, true, point{cx + rx, cy})
	p.close()
}

func (p *path) rect(x, y, w, h, rx, ry float64) {
	rx, ry = min(rx, w/2), min(ry, h/2)
	p.move(point{x + rx, y})
	p.line(point{x + w - rx, y})
	p.arc(rx, ry, 0, false, true, point{x + w, y + ry})
	p.line(point{x + w, y + h - ry})
	p.arc(rx, ry, 0, false, true, point{x + w - rx, y + h})
	p.line(point{x + rx, y + h})
	p.arc(rx, ry, 0, false, true, point{x, y + h - ry})
	p.line(point{x, y + ry})
	p.arc(rx, ry, 0, false, true, point{x + rx, y})
	p.close()
}

func (p *path) arc(rx, ry, angle float64, large, sweep bool, to point) {
	from := p.at
	if rx == 0 || ry == 0 || from == to {
		if from != to {
			p.line(to)
		}
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	sin, cos := math.Sincos(angle * math.Pi / 180)
	dx, dy := (from.x-to.x)/2, (from.y-to.y)/2
	x1, y1 := cos*dx+sin*dy, -sin*dx+cos*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	k := math.Sqrt(max(num/den, 0))
	if large == sweep {
		k = -k
	}
	cx1, cy1 := k*rx*y1/ry, -k*ry*x1/rx
	cx, cy := cos*cx1-sin*cy1+(from.x+to.x)/2, sin*cx1+cos*cy1+(from.y+to.y)/2
	angleOf := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	theta := angleOf(1, 0, (x1-cx1)/rx, (y1-cy1)/ry)
	delta := angleOf((x1-cx1)/rx, (y1-cy1)/ry, (-x1-cx1)/rx, (-y1-cy1)/ry)
	switch {
	case !sweep && delta > 0:
		delta -= 2 * math.Pi
	case sweep && delta < 0:
		delta += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	step := delta / float64(n)
	t := 4.0 / 3 * math.Tan(step/4)
	onEllipse := func(a float64) (point, point) {
		s, c := math.Sincos(a)
		pt := point{cx + rx*c*cos - ry*s*sin, cy + rx*c*sin + ry*s*cos}
		d := point{-rx*s*cos - ry*c*sin, -rx*s*sin + ry*c*cos}
		return pt, d
	}
	for i := range n {
		a0, a1 := theta+float64(i)*step, theta+float64(i+1)*step
		p0, d0 := onEllipse(a0)
		p1, d1 := onEllipse(a1)
		if i == n-1 {
			p1 = to
		}
		p.cubic(point{p0.x + t*d0.x, p0.y + t*d0.y}, point{p1.x - t*d1.x, p1.y - t*d1.y}, p1)
	}
}

func (p *path) parse(d string) error {
	var cmd byte
	i := 0
	p.at = point{}
	skip := func() {
		for i < len(d) && (d[i] == ' ' || d[i] == ',' || d[i] == '\n' || d[i] == '\t' || d[i] == '\r') {
			i++
		}
	}
	number := func() (float64, error) {
		skip()
		j := i
		if j < len(d) && (d[j] == '-' || d[j] == '+') {
			j++
		}
		dot := false
		for j < len(d) && (d[j] >= '0' && d[j] <= '9' || d[j] == '.' && !dot) {
			dot = dot || d[j] == '.'
			j++
		}
		if j < len(d) && (d[j] == 'e' || d[j] == 'E') {
			j++
			if j < len(d) && (d[j] == '-' || d[j] == '+') {
				j++
			}
			for j < len(d) && d[j] >= '0' && d[j] <= '9' {
				j++
			}
		}
		v, err := strconv.ParseFloat(d[i:j], 64)
		i = j
		return v, err
	}
	bit := func() (float64, error) {
		skip()
		if i >= len(d) || d[i] != '0' && d[i] != '1' {
			return 0, fmt.Errorf("bad arc flag at %d in %q", i, d)
		}
		i++
		return float64(d[i-1] - '0'), nil
	}
	for {
		skip()
		if i >= len(d) {
			return nil
		}
		if c := d[i]; c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
			cmd = c
			i++
		} else if cmd == 'M' || cmd == 'm' {
			cmd -= 'M' - 'L'
		}
		upper := cmd &^ 0x20
		rel := cmd != upper
		var args [7]float64
		count := map[byte]int{'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6, 'S': 4, 'Q': 4, 'T': 2, 'A': 7, 'Z': 0}[upper]
		for k := range count {
			read := number
			if upper == 'A' && (k == 3 || k == 4) {
				read = bit
			}
			var err error
			if args[k], err = read(); err != nil {
				return fmt.Errorf("%c at %d in %q: %w", cmd, i, d, err)
			}
		}
		abs := func(x, y float64) point {
			if rel {
				return point{p.at.x + x, p.at.y + y}
			}
			return point{x, y}
		}
		cubic, quad := false, false
		switch upper {
		case 'M':
			p.move(abs(args[0], args[1]))
		case 'L':
			p.line(abs(args[0], args[1]))
		case 'H':
			x := args[0]
			if rel {
				x += p.at.x
			}
			p.line(point{x, p.at.y})
		case 'V':
			y := args[0]
			if rel {
				y += p.at.y
			}
			p.line(point{p.at.x, y})
		case 'C':
			p.cubic(abs(args[0], args[1]), abs(args[2], args[3]), abs(args[4], args[5]))
			cubic = true
		case 'S':
			c1 := p.at
			if p.lastCubic {
				c1 = point{2*p.at.x - p.ctrl.x, 2*p.at.y - p.ctrl.y}
			}
			p.cubic(c1, abs(args[0], args[1]), abs(args[2], args[3]))
			cubic = true
		case 'Q', 'T':
			q := p.at
			switch {
			case upper == 'Q':
				q = abs(args[0], args[1])
			case p.lastQuad:
				q = point{2*p.at.x - p.quadHandle.x, 2*p.at.y - p.quadHandle.y}
			}
			to := abs(args[count-2], args[count-1])
			from := p.at
			p.cubic(point{from.x + 2*(q.x-from.x)/3, from.y + 2*(q.y-from.y)/3}, point{to.x + 2*(q.x-to.x)/3, to.y + 2*(q.y-to.y)/3}, to)
			p.quadHandle, quad = q, true
		case 'A':
			p.arc(args[0], args[1], args[2], args[3] == 1, args[4] == 1, abs(args[5], args[6]))
		case 'Z':
			p.close()
		default:
			return fmt.Errorf("unknown command %c in %q", cmd, d)
		}
		p.lastCubic, p.lastQuad = cubic, quad
	}
}
