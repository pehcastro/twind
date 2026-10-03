//go:build windows

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/twind-dev/twind/twi/text"
)

type console struct {
	mu      sync.Mutex
	out     []byte
	stamps  []time.Time
	screen  screen
	in      windows.Handle
	process windows.Handle
	pc      windows.Handle
}

const screenCols, screenRows = 120, 36

type screen struct {
	cells    [screenRows][screenCols]rune
	row, col int
	rest     []byte
}

func (s *screen) feed(b []byte) {
	b = append(s.rest, b...)
	s.rest = nil
	for len(b) > 0 {
		switch {
		case b[0] == 0x1b:
			n := escapeLength(b)
			if n == 0 {
				s.rest = b
				return
			}
			s.csi(b[:n])
			b = b[n:]
		case b[0] == '\r':
			s.col, b = 0, b[1:]
		case b[0] == '\n':
			s.down()
			b = b[1:]
		case b[0] == '\b':
			s.col, b = max(0, s.col-1), b[1:]
		case b[0] < ' ':
			b = b[1:]
		default:
			if !utf8.FullRune(b) {
				s.rest = b
				return
			}
			r, size := utf8.DecodeRune(b)
			b = b[size:]
			if s.col >= screenCols {
				s.col = 0
				s.down()
			}
			s.cells[s.row][s.col] = r
			s.col++
			if text.Width(string(r)) == 2 && s.col < screenCols {
				s.cells[s.row][s.col] = -1
				s.col++
			}
		}
	}
}

func (s *screen) down() {
	if s.row < screenRows-1 {
		s.row++
		return
	}
	copy(s.cells[:], s.cells[1:])
	s.cells[screenRows-1] = [screenCols]rune{}
}

func escapeLength(b []byte) int {
	if len(b) < 2 {
		return 0
	}
	switch b[1] {
	case '[':
		for i := 2; i < len(b); i++ {
			if b[i] >= 0x40 && b[i] <= 0x7e {
				return i + 1
			}
		}
		return 0
	case ']', 'P', '_':
		for i := 2; i < len(b); i++ {
			if b[i] == 0x07 {
				return i + 1
			}
			if b[i] == 0x1b && i+1 < len(b) && b[i+1] == '\\' {
				return i + 2
			}
		}
		return 0
	case '(', ')':
		if len(b) < 3 {
			return 0
		}
		return 3
	}
	return 2
}

func (s *screen) csi(seq []byte) {
	if len(seq) < 3 || seq[1] != '[' {
		return
	}
	body, final := string(seq[2:len(seq)-1]), seq[len(seq)-1]
	if body != "" && strings.ContainsAny(body[:1], "?<>=") {
		return
	}
	var p []int
	for _, f := range strings.Split(body, ";") {
		n, _ := strconv.Atoi(f)
		p = append(p, n)
	}
	arg := func(i, fallback int) int {
		if i < len(p) && p[i] > 0 {
			return p[i]
		}
		return fallback
	}
	clear := func(row, from, to int) {
		for c := max(0, from); c < min(screenCols, to); c++ {
			s.cells[row][c] = 0
		}
	}
	switch final {
	case 'H', 'f':
		s.row, s.col = arg(0, 1)-1, arg(1, 1)-1
	case 'A':
		s.row -= arg(0, 1)
	case 'B':
		s.row += arg(0, 1)
	case 'C':
		s.col += arg(0, 1)
	case 'D':
		s.col -= arg(0, 1)
	case 'G':
		s.col = arg(0, 1) - 1
	case 'd':
		s.row = arg(0, 1) - 1
	case 'X':
		clear(s.row, s.col, s.col+arg(0, 1))
	case 'K':
		switch arg(0, 0) {
		case 0:
			clear(s.row, s.col, screenCols)
		case 1:
			clear(s.row, 0, s.col+1)
		default:
			clear(s.row, 0, screenCols)
		}
	case 'J':
		mode := arg(0, 0)
		for r := range screenRows {
			if mode >= 2 || mode == 0 && r > s.row || mode == 1 && r < s.row {
				clear(r, 0, screenCols)
			}
		}
		if mode == 0 {
			clear(s.row, s.col, screenCols)
		}
	}
	s.row, s.col = min(max(s.row, 0), screenRows-1), min(max(s.col, 0), screenCols)
}

func (s *screen) lines() []string {
	out := make([]string, screenRows)
	for r := range screenRows {
		var b strings.Builder
		for _, c := range s.cells[r] {
			switch c {
			case -1:
			case 0:
				b.WriteByte(' ')
			default:
				b.WriteRune(c)
			}
		}
		out[r] = strings.TrimRight(b.String(), " ")
	}
	return out
}

func (s *screen) find(want string) (row, col int, ok bool) {
	target := []rune(want)
	for r := range screenRows {
		for c := 0; c+len(target) <= screenCols; c++ {
			if slices.Equal(s.cells[r][c:c+len(target)], target) {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

func (c *console) look(want string) (row, col int, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.screen.find(want)
}

func (c *console) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.Join(c.screen.lines(), "\n")
}

func (c *console) awaitText(t *testing.T, want string, limit time.Duration) (time.Time, bool) {
	t.Helper()
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(time.Millisecond) {
		if _, _, ok := c.look(want); ok {
			return time.Now(), true
		}
	}
	return time.Now(), false
}

func (c *console) click(row, col int) {
	c.type_(fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<0;%d;%dm", col+1, row+1, col+1, row+1))
}

func openConsole(t *testing.T, dir string, env []string, args ...string) *console {
	t.Helper()
	var inR, inW, outR, outW windows.Handle
	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		t.Fatal(err)
	}
	c := &console{in: inW}
	if dll := os.Getenv("TWIND_CONPTY"); dll != "" {
		create := windows.NewLazyDLL(dll).NewProc("ConptyCreatePseudoConsole")
		if r, _, err := create.Call(uintptr(screenCols)|uintptr(screenRows)<<16, uintptr(inR), uintptr(outW), 0, uintptr(unsafe.Pointer(&c.pc))); r != 0 {
			t.Fatalf("%s: %x %v", dll, r, err)
		}
	} else if err := windows.CreatePseudoConsole(windows.Coord{X: screenCols, Y: screenRows}, inR, outW, 0, &c.pc); err != nil {
		t.Fatal(err)
	}
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&c.pc)), unsafe.Sizeof(c.pc)); err != nil {
		t.Fatal(err)
	}
	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	si.Flags = windows.STARTF_USESTDHANDLES
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = windows.EscapeArg(a)
	}
	line, err := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		t.Fatal(err)
	}
	block := utf16.Encode([]rune(strings.Join(env, "\x00") + "\x00\x00"))
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, line, nil, nil, false, windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT, &block[0], cwd, &si.StartupInfo, &pi); err != nil {
		t.Fatal(err)
	}
	_ = windows.CloseHandle(pi.Thread)
	c.process = pi.Process
	go func() {
		buf := make([]byte, 1<<16)
		for {
			var n uint32
			if err := windows.ReadFile(outR, buf, &n, nil); err != nil || n == 0 {
				return
			}
			chunk := string(buf[:n])
			var reply string
			if strings.Contains(chunk, "\x1b[6n") {
				reply += "\x1b[1;1R"
			}
			if strings.Contains(chunk, "\x1b[c") {
				reply += "\x1b[?62;4;22c"
			}
			c.type_(reply)
			c.mu.Lock()
			for range n {
				c.stamps = append(c.stamps, time.Now())
			}
			c.out = append(c.out, buf[:n]...)
			c.screen.feed(buf[:n])
			c.mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = windows.TerminateProcess(c.process, 1)
		if dll := os.Getenv("TWIND_CONPTY"); dll != "" {
			go func() { _, _, _ = windows.NewLazyDLL(dll).NewProc("ConptyClosePseudoConsole").Call(uintptr(c.pc)) }()
			return
		}
		go windows.ClosePseudoConsole(c.pc)
	})
	return c
}

func (c *console) type_(s string) {
	if s != "" {
		var n uint32
		_ = windows.WriteFile(c.in, []byte(s), &n, nil)
	}
}

func (c *console) mark() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.out)
}

func (c *console) await(t *testing.T, from int, want string, limit time.Duration) time.Time {
	t.Helper()
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(time.Millisecond) {
		c.mu.Lock()
		at := bytes.Index(c.out[from:], []byte(want))
		var seen time.Time
		if at >= 0 {
			seen = c.stamps[from+at+len(want)-1]
		}
		c.mu.Unlock()
		if at >= 0 {
			return seen
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	t.Fatalf("no %q on the screen after %v; the last output:\n%q", want, limit, c.out[max(from, len(c.out)-2000):])
	return time.Time{}
}

func (c *console) exited(limit time.Duration) bool {
	event, _ := windows.WaitForSingleObject(c.process, uint32(limit.Milliseconds()))
	return event == windows.WAIT_OBJECT_0
}

func awaitLog(t *testing.T, path string, swaps int, limit time.Duration) string {
	t.Helper()
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		text, _ := os.ReadFile(path)
		if lines := regexp.MustCompile(`\S+ swap: .*`).FindAllString(string(text), -1); len(lines) >= swaps {
			return lines[swaps-1]
		}
	}
	text, _ := os.ReadFile(path)
	t.Fatalf("fewer than %d swaps after %v:\n%s", swaps, limit, text)
	return ""
}

func awaitWarm(t *testing.T, path string) {
	t.Helper()
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if text, _ := os.ReadFile(path); bytes.Contains(text, []byte("compiled for dev")) {
			return
		}
	}
	t.Log("no warm build after 10 s")
}

func devExe(t *testing.T, scratch string) string {
	t.Helper()
	if exe := os.Getenv("TWIND_DEV_EXE"); exe != "" {
		return exe
	}
	exe := filepath.Join(scratch, "twind.exe")
	if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return exe
}

func keep(t *testing.T, clone string) string {
	t.Helper()
	input := filepath.Join(clone, "twi", "ui", "input.go")
	for _, path := range []string{input, filepath.Join(clone, "apps", "documentation", "twir_gen.go")} {
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.WriteFile(path, original, 0o644) })
	}
	return input
}

func logTime(t *testing.T, line string) time.Time {
	t.Helper()
	clock, _, _ := strings.Cut(line, " ")
	at, err := time.ParseInLocation("15:04:05.000000", clock, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), at.Hour(), at.Minute(), at.Second(), at.Nanosecond(), time.Local)
}

func (c *console) firstAfter(t *testing.T, at time.Time, limit time.Duration) time.Time {
	t.Helper()
	for end := time.Now().Add(limit); time.Now().Before(end); time.Sleep(time.Millisecond) {
		c.mu.Lock()
		i, _ := slices.BinarySearchFunc(c.stamps, at, time.Time.Compare)
		var seen time.Time
		if i < len(c.stamps) {
			seen = c.stamps[i]
		}
		c.mu.Unlock()
		if !seen.IsZero() {
			return seen
		}
	}
	t.Fatalf("no output after %v", at)
	return time.Time{}
}

func scratchEnv(root, temp string) []string {
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool {
		key, _, _ := strings.Cut(strings.ToUpper(kv), "=")
		return key == "LOCALAPPDATA" || key == "TMP" || key == "TEMP" || key == "TWIND_TRACE"
	})
	return append(env, "LOCALAPPDATA="+filepath.Join(root, "cache"), "TMP="+temp, "TEMP="+temp)
}

func edit(t *testing.T, path string, pairs ...string) time.Time {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var marked []byte
	for i := 0; i < len(pairs); i += 2 {
		at := bytes.Index(src, []byte(pairs[i]))
		if at < 0 {
			t.Fatalf("%s holds no %q", path, pairs[i])
		}
		marked = append(marked, src[:at]...)
		marked = append(marked, pairs[i+1]...)
		src = src[at+len(pairs[i]):]
	}
	if err := os.WriteFile(path, append(marked, src...), 0o644); err != nil {
		t.Fatal(err)
	}
	return time.Now()
}

func openDocs(t *testing.T, name string) (c *console, input, logPath, scratch string) {
	t.Helper()
	clone, root := os.Getenv("TWIND_DEV_CLONE"), os.Getenv("TWIND_E2E")
	if clone == "" || root == "" {
		t.Skip("TWIND_DEV_CLONE names a clone of this repo whose twi/ui/input.go the run edits, TWIND_E2E a scratch directory")
	}
	scratch, err := os.MkdirTemp(root, name+"-")
	if err != nil {
		t.Fatal(err)
	}
	input, logPath = keep(t, clone), filepath.Join(scratch, "dev.log")
	env := scratchEnv(root, scratch)
	c = openConsole(t, clone, env, devExe(t, scratch), "dev", "-log", logPath, "./cmd/twind/docsdev", "-page", "input")
	if _, ok := c.awaitText(t, "m@example.com", 5*time.Minute); !ok {
		t.Fatalf("the input page never showed m@example.com:\n%s", c.text())
	}
	awaitLog(t, logPath, 1, time.Minute)
	awaitWarm(t, logPath)
	return c, input, logPath, scratch
}

func TestDevSwapTime(t *testing.T) {
	c, input, logPath, _ := openDocs(t, "swap")
	colours := [][2]string{{"text-primary", "38;2;216;242;90"}, {"text-destructive", "38;2;239;67;67"}, {"text-accent-foreground", "38;2;148;90;242"}}
	class := "text-muted-foreground"
	took := map[bool][]time.Duration{}
	for i := range 2 * len(colours) {
		next := colours[i%len(colours)]
		from := c.mark()
		saved := edit(t, input, `placeholderClass = "`+class+`"`, `placeholderClass = "`+next[0]+`"`)
		shown := c.await(t, from, next[1], time.Minute)
		returning := i >= len(colours)
		took[returning] = append(took[returning], shown.Sub(saved).Round(time.Millisecond))
		t.Logf("%s, returning %t: %s; save to first frame %v", next[0], returning, awaitLog(t, logPath, i+2, time.Second), shown.Sub(saved).Round(time.Millisecond))
		class = next[0]
		time.Sleep(1500 * time.Millisecond)
	}
	for _, returning := range []bool{false, true} {
		slices.Sort(took[returning])
		t.Logf("save to first frame, returning to an earlier source %t: %v", returning, took[returning])
	}
	text, _ := os.ReadFile(logPath)
	t.Logf("log:\n%s", text)
}

func TestDevDrivenSession(t *testing.T) {
	c, input, logPath, scratch := openDocs(t, "session")
	frames := 0
	shot := func(name string) {
		frames++
		path := filepath.Join(scratch, fmt.Sprintf("%02d-%s.txt", frames, name))
		_ = os.WriteFile(path, []byte(c.text()), 0o644)
		t.Logf("frame %s", path)
	}
	shot("start")
	field := func() (int, int) {
		row, col, ok := c.look("Email")
		if !ok {
			shot("no-email-label")
			t.Fatal("no Email label on the screen")
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		for r := row + 1; r < min(row+6, screenRows-1); r++ {
			if slices.Contains(c.screen.cells[r][:], '╭') {
				return r + 1, col + 2
			}
		}
		return row + 3, col + 2
	}
	token, last := 1000, ""
	usable := func(why string, since time.Time, shows bool) {
		t.Helper()
		for try := range 3 {
			token++
			next := strconv.Itoa(token)
			if try > 0 {
				c.click(field())
			}
			sent := time.Now()
			c.type_(next)
			seen, ok := c.awaitText(t, next, time.Second)
			if !ok {
				continue
			}
			_, _, kept := c.look(last)
			t.Logf("%s: %s shows %v after the keys, %v after the save; clicked first %t; earlier %q still shown %t", why, next, seen.Sub(sent).Round(time.Millisecond), seen.Sub(since).Round(time.Millisecond), try > 0, last, kept)
			if !shows || try > 0 || !kept && last != "" {
				t.Errorf("%s: want the field focused and its text kept, typed text shown %t", why, shows)
			}
			last = next
			return
		}
		shot("not-shown-" + strings.ReplaceAll(why, " ", "-"))
		if shows {
			t.Errorf("%s: nothing typed into the Email field shows, focused or clicked", why)
		}
		t.Logf("%s: nothing typed shows", why)
	}
	c.click(field())
	time.Sleep(300 * time.Millisecond)
	usable("before any edit", time.Now(), true)
	shot("typed")
	swaps := 1
	step := func(why string, swapped, shows bool, pairs ...string) {
		t.Helper()
		from := c.mark()
		saved := edit(t, input, pairs...)
		if !swapped {
			first := c.await(t, from, "undefined", time.Minute)
			t.Logf("%s: error line %v after the save", why, first.Sub(saved).Round(time.Millisecond))
		} else {
			swaps++
			line := awaitLog(t, logPath, swaps, time.Minute)
			started := logTime(t, line)
			first := c.firstAfter(t, started, time.Minute)
			t.Logf("%s: %s; new app's first output %v after its start, %v after the save", why, line, first.Sub(started).Round(time.Millisecond), first.Sub(saved).Round(time.Millisecond))
		}
		usable(why, saved, shows)
		shot(strings.ReplaceAll(why, " ", "-"))
	}
	step("h-2", true, true, `inputHeight      = "shrink-0 `, `inputHeight      = "h-2 shrink-0 `)
	step("h-2 back to auto", true, true, `inputHeight      = "h-2 shrink-0 `, `inputHeight      = "shrink-0 `)
	step("placeholder colour", true, true, `placeholderClass = "text-muted-foreground"`, `placeholderClass = "text-destructive"`)
	step("a new class", true, true, `placeholderClass = "text-destructive"`, `placeholderClass = "text-lime-700"`)
	step("a compile error", false, true, `in.edge(fieldEdge)`, `in.edge(fieldEdgeX)`)
	step("the fix", true, true, `in.edge(fieldEdgeX)`, `in.edge(fieldEdge)`)
	c.type_("\x03")
	if !c.exited(30 * time.Second) {
		t.Error("twind dev still runs 30 s after Ctrl+C")
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(input), "..", "..", ".twind", "cache", "dev", "*")); len(left) > 0 {
		t.Errorf("twind dev left %v", left)
	}
	text, _ := os.ReadFile(logPath)
	t.Logf("log:\n%s", text)
}

func TestDevOutsideTheRepo(t *testing.T) {
	root := os.Getenv("TWIND_E2E")
	if root == "" {
		t.Skip("TWIND_E2E names a scratch directory for twind new and twind dev outside the repo")
	}
	twind, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(root, "run-")
	if err != nil {
		t.Fatal(err)
	}
	app, temp := filepath.Join(scratch, "hello"), filepath.Join(scratch, "tmp")
	if err := os.Mkdir(temp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := newApp([]string{app, "-module", "example.com/hello"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	env := scratchEnv(root, temp)
	for _, args := range [][]string{
		{"mod", "edit", "-replace", "github.com/twind-dev/twind=" + twind},
		{"mod", "tidy"},
		{"tool", "twind", "check", "."},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir, cmd.Env = app, slices.Concat(env, []string{"GOPROXY=off", "GOFLAGS=-mod=mod"})
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	logPath := filepath.Join(scratch, "dev.log")
	c := openConsole(t, app, env, "go", "tool", "twind", "dev", "-log", logPath, ".")
	c.await(t, 0, "watching 1 files", time.Minute)
	c.await(t, 0, "Hello Twind", 10*time.Minute)
	t.Log("first: ", awaitLog(t, logPath, 1, time.Second))

	main := filepath.Join(app, "main.go")
	from := c.mark()
	saved := edit(t, main, `txt("px-1 text-muted-foreground", "Projects")`, `txt("px-1 text-lime-700", "Projects")`)
	shown := c.await(t, from, "38;2;73;125;0", 2*time.Minute)
	t.Logf("class edit, new utility: %s; save to first frame %v", awaitLog(t, logPath, 2, time.Second), shown.Sub(saved).Round(time.Millisecond))

	from = c.mark()
	saved = edit(t, main, `txt("px-1 text-muted-foreground", "Settings"),`, `txt("px-1 text-lime-700", "Settings"),`)
	shown = c.await(t, from, "38;2;73;125;0", time.Minute)
	t.Logf("class edit, known utility: %s; save to first frame %v", awaitLog(t, logPath, 3, time.Second), shown.Sub(saved).Round(time.Millisecond))

	from = c.mark()
	saved = edit(t, main, `"Projects"),`, `"Settings"),`, `"Settings"),`, `"Projects"),`)
	shown = c.await(t, from, "Projects", time.Minute)
	c.mu.Lock()
	if frame := c.out[from:]; bytes.Index(frame, []byte("Settings")) > bytes.Index(frame, []byte("Projects")) {
		t.Errorf("the body edit put Settings above Projects, the new frame does not:\n%q", frame)
	}
	c.mu.Unlock()
	t.Logf("body edit: %s; save to first frame %v", awaitLog(t, logPath, 4, time.Second), shown.Sub(saved).Round(time.Millisecond))

	c.type_("q")
	if !c.exited(30 * time.Second) {
		t.Fatal("twind dev still runs 30 s after q")
	}
	left, _ := filepath.Glob(filepath.Join(app, ".twind", "cache", "dev", "*"))
	if len(left) > 0 {
		t.Errorf("twind dev left %v", left)
	}
	text, _ := os.ReadFile(logPath)
	t.Logf("log:\n%s", text)
}
