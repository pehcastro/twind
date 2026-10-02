package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRejectsWhatTheChildMustNotGet(t *testing.T) {
	for name, body := range map[string]string{
		"half written":  `{"version":1,"focus":"inp`,
		"unknown field": `{"version":1,"route":"input"}`,
		"old version":   `{"version":0,"focus":"input"}`,
		"empty":         ``,
	} {
		path := filepath.Join(t.TempDir(), "state.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil {
			t.Errorf("%s: Read accepted %q", name, body)
		}
	}
	path := filepath.Join(t.TempDir(), "state.json")
	if err := Write(path, Snapshot{Focus: "email", Scroll: map[string]Offset{"main": {Y: 12}}, Values: map[string]string{"page": "input"}}); err != nil {
		t.Fatal(err)
	}
	s, err := Read(path)
	if err != nil || s.Focus != "email" || s.Scroll["main"].Y != 12 || s.Values["page"] != "input" {
		t.Errorf("Read gave %+v, %v", s, err)
	}
	if _, err := os.Stat(path + ".partial"); !os.IsNotExist(err) {
		t.Errorf("the partial file is left behind: %v", err)
	}
}
