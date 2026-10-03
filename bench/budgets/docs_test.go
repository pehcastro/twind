package budgets_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	docsapp "github.com/twind-dev/twind/apps/documentation"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/input"
	"github.com/twind-dev/twind/twi/style"
	"github.com/twind-dev/twind/twi/terminal"
)

const (
	docsQuiet         = 300 * time.Millisecond
	docsStreamsEnv    = "TWIND_DOCS_STREAMS"
	docsSidebarX      = 5
	docsContentRow    = 20
	docsContentColumn = 100
)

type docsFrame struct {
	at     time.Duration
	bytes  int
	images int
	pixels int
}

type docsBackend struct {
	events     chan input.Event
	frames     chan docsFrame
	now        func() time.Duration
	cols, rows int
	caps       terminal.Capabilities
	stream     bytes.Buffer
	mu         sync.Mutex
	written    *docsFrame
}

func (be *docsBackend) Write(p []byte) (int, error) {
	be.stream.Write(p)
	f := docsFrame{at: be.now(), bytes: len(p), images: bytes.Count(p, []byte("\x1bP")) + bytes.Count(p, []byte("\x1b_G"))}
	if be.caps.Graphics != terminal.GraphicsGDI {
		be.frames <- f
		return len(p), nil
	}
	be.mu.Lock()
	pending := be.written
	be.written = &f
	be.mu.Unlock()
	if pending != nil {
		be.frames <- *pending
	}
	return len(p), nil
}

func (be *docsBackend) Paint(p terminal.Pixels) bool {
	be.mu.Lock()
	f := docsFrame{at: be.now()}
	if be.written != nil {
		f, be.written = *be.written, nil
	}
	be.mu.Unlock()
	for _, t := range p.Tiles {
		f.pixels += t.Cells.Dx() * t.Cells.Dy() * p.Cell.X * p.Cell.Y
	}
	be.frames <- f
	return true
}

func (be *docsBackend) Events() <-chan input.Event           { return be.events }
func (be *docsBackend) Sync() bool                           { return true }
func (be *docsBackend) Exit() error                          { return nil }
func (be *docsBackend) Size() (width, height int, err error) { return be.cols, be.rows, nil }
func (be *docsBackend) Capabilities() terminal.Capabilities  { return be.caps }

func (be *docsBackend) quiet() (frames []docsFrame) {
	for {
		select {
		case f := <-be.frames:
			frames = append(frames, f)
		case <-time.After(docsQuiet):
			be.mu.Lock()
			if be.written != nil {
				frames, be.written = append(frames, *be.written), nil
			}
			be.mu.Unlock()
			return frames
		}
	}
}

type docsPath struct {
	name       string
	cols, rows int
	caps       terminal.Capabilities
}

type docsStep struct {
	name   string
	events func(i int, page string, rows map[string]int) []input.Event
}

func docsSidebar(b *testing.B, sheet style.Sheet, p docsPath) map[string]int {
	d := drive.New(func(rt *twi.Runtime) func() twi.Node {
		view, err := docsapp.New(rt, docsapp.Start{Page: "card", Theme: "twind-dark"})
		if err != nil {
			b.Fatal(err)
		}
		return view
	}, drive.Size(p.cols, p.rows), drive.Styles(sheet))
	rows := map[string]int{}
	for y, line := range strings.Split(d.Frame().Text(), "\n") {
		for _, entry := range []string{"Introduction", "Installation", "Theming"} {
			if _, seen := rows[entry]; !seen && strings.HasPrefix(strings.TrimSpace(line), entry) {
				rows[entry] = y
			}
		}
	}
	if err := d.Close(); err != nil || len(rows) != 3 {
		b.Fatalf("sidebar rows %v, %v", rows, err)
	}
	return rows
}

func docsClick(x, y int) []input.Event {
	return []input.Event{
		input.MouseEvent{X: x, Y: y, Button: input.MouseLeft, Action: input.MousePress},
		input.MouseEvent{X: x, Y: y, Button: input.MouseLeft, Action: input.MouseRelease},
	}
}

func BenchmarkDocs(b *testing.B) {
	sheet, err := docsapp.Styles()
	if err != nil {
		b.Fatal(err)
	}
	sixel := func(id terminal.Identity, cell image.Point) terminal.Capabilities {
		return terminal.Capabilities{Identity: id, Sync: true, Graphics: terminal.GraphicsSixel, CellPixels: cell}
	}
	paths := []docsPath{
		{"vscode", 248, 36, sixel(terminal.IdentityVSCode, image.Pt(7, 17))},
		{"wt", 248, 36, sixel(terminal.IdentityOther, image.Pt(7, 17))},
		{"zed", 187, 40, terminal.Capabilities{Identity: terminal.IdentityZed, Sync: true, Graphics: terminal.GraphicsGDI, CellPixels: image.Pt(8, 17)}},
	}
	steps := []docsStep{
		{"hover", func(i int, _ string, rows map[string]int) []input.Event {
			return []input.Event{input.MouseEvent{X: docsSidebarX, Y: []int{rows["Introduction"], rows["Installation"]}[i%2], Button: input.MouseNone, Action: input.MouseMove}}
		}},
		{"wheel", func(i int, _ string, _ map[string]int) []input.Event {
			return []input.Event{input.MouseEvent{X: docsContentColumn, Y: docsContentRow, Button: []input.MouseButton{input.MouseWheelDown, input.MouseWheelUp}[i%2], Action: input.MouseScroll}}
		}},
		{"page", func(i int, page string, rows map[string]int) []input.Event {
			targets := []int{rows["Theming"], rows["Introduction"]}
			if page == "theming" {
				targets[0], targets[1] = targets[1], targets[0]
			}
			return docsClick(docsSidebarX, targets[i%2])
		}},
		{"scheme", func(int, string, map[string]int) []input.Event {
			return []input.Event{input.KeyEvent{Key: input.KeyRune, Rune: 'm'}}
		}},
	}
	for _, p := range paths {
		rows := docsSidebar(b, sheet, p)
		for _, page := range []string{"card", "theming", "blocks"} {
			for _, s := range steps {
				b.Run(p.name+"/"+page+"/"+s.name, func(b *testing.B) { docsRun(b, sheet, p, page, s, rows) })
				if s.name != "page" {
					b.Run("loop/"+p.name+"/"+page+"/"+s.name, func(b *testing.B) { docsLoop(b, sheet, p, page, s, rows) })
				}
			}
		}
	}
}

func docsStart(b *testing.B, sheet style.Sheet, p docsPath, page string) (*docsBackend, *twi.Runtime, chan error) {
	be := &docsBackend{events: make(chan input.Event), frames: make(chan docsFrame, 64), now: clock(b), cols: p.cols, rows: p.rows, caps: p.caps}
	rt := twi.New(twi.Backend(be, realDocsClock{}), twi.Styles(sheet), twi.ColorProfile(color.TrueColor))
	view, err := docsapp.New(rt, docsapp.Start{Page: page, Theme: "twind-dark"})
	if err != nil {
		b.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- rt.Run(view) }()
	be.quiet()
	return be, rt, done
}

func docsLoop(b *testing.B, sheet style.Sheet, p docsPath, page string, s docsStep, rows map[string]int) {
	be, rt, done := docsStart(b, sheet, p, page)
	be.events <- input.MouseEvent{X: docsContentColumn, Y: docsContentRow, Button: input.MouseNone, Action: input.MouseMove}
	be.quiet()
	b.ReportAllocs()
	b.ResetTimer()
	c0 := cycles()
	for i := range b.N {
		for _, e := range s.events(i, page, rows) {
			be.events <- e
		}
		<-be.frames
	}
	b.ReportMetric(float64(cycles()-c0)/float64(b.N), "cycles/op")
	b.StopTimer()
	if extra := len(be.quiet()); extra > 0 {
		b.Fatalf("%d frames after the loop, want one per step", extra)
	}
	rt.Quit()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
}

func docsRun(b *testing.B, sheet style.Sheet, p docsPath, page string, s docsStep, rows map[string]int) {
	be, rt, done := docsStart(b, sheet, p, page)
	now := be.now
	if s.name != "hover" {
		be.events <- input.MouseEvent{X: docsContentColumn, Y: docsContentRow, Button: input.MouseNone, Action: input.MouseMove}
		be.quiet()
	}
	var records [][]byte
	records = append(records, slices.Clone(be.stream.Bytes()))
	var latency, perFrame []time.Duration
	var cycleSamples []float64
	total := docsFrame{}
	frames := 0
	var before, after runtime.MemStats
	var allocated, objects uint64
	var collections uint32
	b.ResetTimer()
	for i := range b.N {
		be.stream.Reset()
		runtime.ReadMemStats(&before)
		c0 := cycles()
		begin := now()
		for _, e := range s.events(i, page, rows) {
			be.events <- e
		}
		got := be.quiet()
		spent := cycles() - c0
		runtime.ReadMemStats(&after)
		allocated, objects, collections = allocated+after.TotalAlloc-before.TotalAlloc, objects+after.Mallocs-before.Mallocs, collections+after.NumGC-before.NumGC
		if len(got) == 0 {
			b.Fatalf("%s: no frame after the events", s.name)
		}
		latency = append(latency, got[0].at-begin)
		for _, f := range got {
			total.bytes += f.bytes
			total.images += f.images
			total.pixels += f.pixels
		}
		frames += len(got)
		cycleSamples = append(cycleSamples, float64(spent)/float64(len(got)))
		perFrame = append(perFrame, got[len(got)-1].at-begin)
		records = append(records, slices.Clone(be.stream.Bytes()))
	}
	b.StopTimer()
	rt.Quit()
	for waiting := true; waiting; {
		select {
		case <-be.frames:
		case err := <-done:
			if err != nil {
				b.Fatal(err)
			}
			waiting = false
		}
	}
	percentiles(b, "first-", latency)
	percentiles(b, "settled-", perFrame)
	slices.Sort(cycleSamples)
	b.ReportMetric(cycleSamples[len(cycleSamples)/2], "p50-cycles/frame")
	b.ReportMetric(cycleSamples[len(cycleSamples)*95/100], "p95-cycles/frame")
	n := float64(b.N)
	b.ReportMetric(float64(frames)/n, "frames/op")
	b.ReportMetric(float64(total.bytes)/n, "bytes/op")
	b.ReportMetric(float64(total.images)/n, "images/op")
	b.ReportMetric(float64(total.pixels)/n, "pixels/op")
	b.ReportMetric(float64(allocated)/n, "B-alloc/op")
	b.ReportMetric(float64(objects)/n, "objects/op")
	b.ReportMetric(float64(collections)/n, "gc/op")
	if dir := os.Getenv(docsStreamsEnv); dir != "" {
		var out bytes.Buffer
		for _, r := range records {
			out.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(r))))
			out.Write(r)
		}
		if err := os.WriteFile(filepath.Join(dir, p.name+"-"+page+"-"+s.name+".bin"), out.Bytes(), 0o644); err != nil {
			b.Fatal(err)
		}
	}
}

type realDocsClock struct{}

func (realDocsClock) Now() time.Time                         { return time.Now() }
func (realDocsClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
