package twi_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/input"
	"github.com/pehcastro/twind/twi/terminal"
	"github.com/pehcastro/twind/twi/testdata/hello"
	"github.com/pehcastro/twind/twi/theme"
)

type frames chan string

func (f frames) Write(p []byte) (int, error) {
	f <- string(p)
	return len(p), nil
}

func (frames) Events() <-chan input.Event           { return nil }
func (frames) Size() (width, height int, err error) { return 20, 3, nil }
func (frames) Capabilities() terminal.Capabilities  { return terminal.Capabilities{} }
func (frames) Exit() error                          { return nil }

type wallClock struct{}

func (wallClock) Now() time.Time                         { return time.Now() }
func (wallClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func twind(t *testing.T, scheme theme.Scheme) theme.Theme {
	t.Helper()
	for _, th := range theme.Builtin() {
		if th.Name == "twind" && th.Scheme == scheme {
			return th
		}
	}
	t.Fatal("no twind theme")
	return theme.Theme{}
}

func TestSetThemeBeforeRunDrawsOnce(t *testing.T) {
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	out := make(frames, 4)
	rt := twi.New(twi.Styles(s), twi.Theme(twind(t, theme.Light)), twi.Backend(out, wallClock{}), twi.ColorProfile(color.TrueColor))
	rt.SetTheme(twind(t, theme.Dark))
	done := make(chan error, 1)
	go func() { done <- rt.Run(hello.Card) }()
	const white, darkCard = "48;2;255;255;255", "48;2;9;8;13"
	select {
	case first := <-out:
		if !strings.Contains(first, darkCard) || strings.Contains(first, white) {
			t.Errorf("first frame after SetTheme(twind dark) before Run: want %q and no %q:\n%q", darkCard, white, first)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no frame within 2s")
	}
	select {
	case f := <-out:
		t.Errorf("a second frame with no input: %q", f)
	case <-time.After(200 * time.Millisecond):
	}
	rt.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSetThemeFromAnotherGoroutine(t *testing.T) {
	s, err := hello.Styles()
	if err != nil {
		t.Fatal(err)
	}
	out := make(frames, 64)
	rt := twi.New(twi.Styles(s), twi.Theme(twind(t, theme.Light)), twi.Backend(out, wallClock{}), twi.ColorProfile(color.TrueColor))
	done := make(chan error, 1)
	go func() { done <- rt.Run(hello.Card) }()
	next := func() string {
		t.Helper()
		select {
		case f := <-out:
			return f
		case <-time.After(2 * time.Second):
			t.Fatal("no frame within 2s")
			return ""
		}
	}
	const white, darkCard = "48;2;255;255;255", "48;2;9;8;13"
	if first := next(); !strings.Contains(first, white) {
		t.Errorf("first frame under twind light has no white card background %q:\n%q", white, first)
	}
	go rt.SetTheme(twind(t, theme.Dark))
	second := next()
	t.Logf("frame after SetTheme: %q", second)
	if !strings.Contains(second, darkCard) || strings.Contains(second, white) {
		t.Errorf("frame after SetTheme(twind dark): want card background %q and no %q", darkCard, white)
	}
	select {
	case f := <-out:
		t.Errorf("a second frame after one SetTheme: %q", f)
	case <-time.After(200 * time.Millisecond):
	}
	schemes := [...]theme.Theme{twind(t, theme.Light), twind(t, theme.Dark)}
	var wg sync.WaitGroup
	for i := range cap(out) {
		wg.Go(func() { rt.SetTheme(schemes[i%2]) })
	}
	wg.Wait()
	rt.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
