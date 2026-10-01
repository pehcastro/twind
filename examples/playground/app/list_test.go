package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/twind-dev/twind/twi/drive"
)

const listCardEnd = 56

func listShown(frame string) []int {
	var rows []int
	for l := range strings.Lines(frame) {
		if _, after, ok := strings.Cut(l, "▐ row "); ok {
			n, _ := strconv.Atoi(strings.Fields(after)[0])
			rows = append(rows, n)
		}
	}
	return rows
}

func besideList(frame string) string {
	var beside []string
	for l := range strings.Lines(frame) {
		if r := []rune(strings.TrimRight(l, "\n")); len(r) > listCardEnd {
			beside = append(beside, string(r[listCardEnd:]))
		}
	}
	return strings.Join(beside, "\n")
}

func TestListScript(t *testing.T) {
	sheet, err := Styles()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("testdata", "list.twd"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := drive.RunScript(bytes.NewReader(src), App, out, drive.Styles(sheet)); err != nil {
		t.Fatal(err)
	}
	frame := func(name string) string {
		b, err := os.ReadFile(filepath.Join(out, name+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	first, side := frame("1-row-1"), besideList(frame("1-row-1"))
	for _, step := range []struct {
		name  string
		first int
	}{{"1-row-1", 1}, {"2-three-notches", 10}, {"3-wheel-beside", 10}, {"4-pagedown", 20}} {
		f := frame(step.name)
		if rows := listShown(f); len(rows) == 0 || rows[0] != step.first {
			t.Errorf("%s: rows %v shown, want %d first:\n%s", step.name, rows, step.first, f)
		}
		if besideList(f) != side {
			t.Errorf("%s: the text beside the list moved:\n%s\nwas:\n%s", step.name, f, first)
		}
	}
	t.Logf("three notches and a PageDown:\n%s", frame("4-pagedown"))
}

func TestListPageTabReveals(t *testing.T) {
	d := open(t)
	run(d, "list")
	for range 10 {
		if strings.HasSuffix(status(d), "focus row 1 list") {
			break
		}
		press(d, "tab")
	}
	if !strings.HasSuffix(status(d), "focus row 1 list") {
		t.Fatalf("tab never reached the first row: status %q", status(d))
	}
	for row := 2; row <= listRows; row++ {
		press(d, "tab")
		shown := listShown(d.Frame().Text())
		if !strings.HasSuffix(status(d), "focus row "+strconv.Itoa(row)+" list") || len(shown) == 0 || row < shown[0] || row > shown[len(shown)-1] {
			t.Fatalf("tab to row %d: rows %v shown, status %q:\n%s", row, shown, status(d), d.Frame().Text())
		}
		if row == 14 {
			t.Logf("tabbed to row 14, past the fold:\n%s", d.Frame().Text())
		}
	}
	t.Logf("tabbed to the last row:\n%s", d.Frame().Text())
}
