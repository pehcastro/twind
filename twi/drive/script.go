package drive

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func RunScript(r io.Reader, app App, out string, opts ...Option) (err error) {
	var d *Driver
	defer func() {
		if d == nil {
			return
		}
		if closeErr := d.Close(); err == nil {
			err = closeErr
		}
	}()
	lines := bufio.NewScanner(r)
	for line := 1; lines.Scan(); line++ {
		verb, arg, _ := strings.Cut(strings.TrimRight(lines.Text(), "\r"), " ")
		if verb == "" {
			continue
		}
		if d == nil && verb != "size" {
			d = New(app, opts...)
		}
		switch verb {
		case "size":
			var w, h int
			w, h, err = parseSize(arg)
			if d != nil {
				err = errors.New("drive: size comes before every action, use resize")
			}
			opts = append(opts, Size(w, h))
		case "press":
			d.Press(arg)
		case "type":
			d.Type(arg)
		case "wait":
			var dt time.Duration
			if dt, err = time.ParseDuration(arg); err == nil {
				d.Advance(dt)
			}
		case "resize":
			var w, h int
			if w, h, err = parseSize(arg); err == nil {
				d.Resize(w, h)
			}
		case "wheel":
			var notches, x, y int
			if notches, x, y, err = parseWheel(arg); err == nil {
				d.Wheel(x, y, notches)
			}
		case "frame":
			err = writeFrame(d.Frame(), out, arg)
		default:
			err = fmt.Errorf("drive: unknown verb %q", verb)
		}
		if err == nil && d != nil {
			err = d.Err()
		}
		if err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
	}
	return lines.Err()
}

func parseSize(arg string) (width, height int, err error) {
	w, h, _ := strings.Cut(arg, "x")
	width, werr := strconv.Atoi(w)
	height, herr := strconv.Atoi(h)
	if werr != nil || herr != nil || width < 1 || height < 1 {
		return 0, 0, fmt.Errorf("drive: size %q is not WIDTHxHEIGHT", arg)
	}
	return width, height, nil
}

func parseWheel(arg string) (notches, x, y int, err error) {
	fields := strings.Fields(arg)
	if len(fields) == 3 {
		notches = map[string]int{"up": -1, "down": 1}[fields[0]]
		var xerr, yerr error
		x, xerr = strconv.Atoi(fields[1])
		y, yerr = strconv.Atoi(fields[2])
		if notches != 0 && xerr == nil && yerr == nil {
			return notches, x, y, nil
		}
	}
	return 0, 0, 0, fmt.Errorf("drive: wheel %q is not up|down X Y", arg)
}

func writeFrame(f Frame, dir, name string) error {
	if !filepath.IsLocal(name) || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("drive: frame name %q is not a plain file name", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Join(dir, name)
	return errors.Join(os.WriteFile(base+".ansi", []byte(f.ANSI()), 0o644), os.WriteFile(base+".txt", []byte(f.Text()), 0o644))
}
