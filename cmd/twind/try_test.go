package main

import (
	"errors"
	"strings"
	"testing"
)

func TestTryLists(t *testing.T) {
	var out strings.Builder
	if err := try(nil, &out); err != nil {
		t.Fatalf("try with no name: %v", err)
	}
	for _, name := range []string{"docs", "landing", "portfolio", "playground", "gallery"} {
		if !strings.Contains(out.String(), "\n  "+name+" ") {
			t.Errorf("the list has no line for %s:\n%s", name, out.String())
		}
	}
}

func TestTryUnknown(t *testing.T) {
	var usage usageError
	if err := try([]string{"nope"}, &strings.Builder{}); !errors.As(err, &usage) {
		t.Errorf("try nope: got %v, want a usage error", err)
	}
}
