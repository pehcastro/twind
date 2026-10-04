package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	konst "github.com/twind-dev/twind/internal/konst/style"
	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/drive"
	"github.com/twind-dev/twind/twi/tailwind"
	"github.com/twind-dev/twind/twi/theme"
)

func TestStylesFresh(t *testing.T) {
	stale, err := tailwind.Stale(".", konst.GeneratedFile)
	if err != nil {
		t.Fatal(err)
	}
	if stale {
		t.Errorf("%s is stale against the classes in this package: run go generate", konst.GeneratedFile)
	}
}

func TestTour(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("testdata", "tour.twd"))
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	posts, err := load(postFiles)
	if err != nil {
		t.Fatal(err)
	}
	start, _ := builtin("twind-light")
	out := t.TempDir()
	app := func(rt *twi.Runtime) func() twi.Node { return newSite(rt, start, posts).view }
	if err := drive.RunScript(strings.NewReader(string(script)), app, out, drive.Styles(sheet)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		frame          string
		shown, missing []string
	}{
		{"home", []string{"Noor Valenko", "Software engineer", "Code ↗", "Now", "Recent writing", "A diff renderer in one evening", "Small tools, kept small", "All posts ›"}, []string{"Notes from a year of on-call"}},
		{"projects", []string{"Projects", "tide", "loupe", "quay", "inkwell", "★ 1.2k · since 2022", "markdown"}, []string{"Recent writing"}},
		{"blog", []string{"Writing", "Sep 12, 2026", "Feb 14, 2026", "Notes from a year of on-call", "A quieter shell"}, []string{"loupe"}},
		{"post", []string{"‹ All posts", "A quieter shell", "Jul 3, 2026", "min read", "shell", "habits", "Every character in it is a tax", "What stayed", "bash"}, []string{"Writing"}},
		{"post-end", []string{"PROMPT_COMMAND=prompt", "What I learned", "‹ A diff renderer in one evening", "Small tools, kept small ›"}, []string{"‹ All posts"}},
		{"back", []string{"Writing", "Notes from a year of on-call"}, []string{"What stayed"}},
		{"contact", []string{"Say hello", "hello@noor.example", "social.example/@noor", "Copy", "Open to work"}, []string{"Writing"}},
	} {
		b, err := os.ReadFile(filepath.Join(out, c.frame+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		t.Logf("frame %s:\n%s", c.frame, text)
		for _, s := range append([]string{"Home", "Projects", "Blog", "Contact", "made with Twind"}, c.shown...) {
			if !strings.Contains(text, s) {
				t.Errorf("frame %s: no %q", c.frame, s)
			}
		}
		for _, s := range c.missing {
			if strings.Contains(text, s) {
				t.Errorf("frame %s: %q shown, want it gone", c.frame, s)
			}
		}
	}
	ansi, err := os.ReadFile(filepath.Join(out, "back.ansi"))
	if err != nil {
		t.Fatal(err)
	}
	a := start.Tokens[theme.Accent].RGBA
	accent := fmt.Sprintf("48;2;%d;%d;%d", a.R, a.G, a.B)
	for line := range strings.Lines(string(ansi)) {
		for title, want := range map[string]bool{"quieter": true, "diff": false} {
			if strings.Contains(line, title) && strings.Contains(line, accent) != want {
				t.Errorf("back: the row of %q is highlighted %t, want %t: the post just read keeps the focus", title, !want, want)
			}
		}
	}
}

func TestLoadRejects(t *testing.T) {
	for name, src := range map[string]string{
		"no heading": "<Meta date=\"2026-01-01\" tags=\"a\" />\n\nText.\n",
		"no meta":    "# Title\n\nText.\n",
		"bad date":   "# Title\n\n<Meta date=\"2026-13-01\" tags=\"a\" />\n\nText.\n",
		"no summary": "# Title\n\n<Meta date=\"2026-01-01\" tags=\"a\" />\n",
		"bad tag":    "# Title\n\n<Meta when=\"2026-01-01\" />\n\nText.\n",
	} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "posts"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "posts", "p.md"), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := load(os.DirFS(dir)); err == nil {
			t.Errorf("%s: loaded, want an error", name)
		}
	}
}
