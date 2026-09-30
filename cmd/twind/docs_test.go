package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDocsArgs(t *testing.T) {
	var usage usageError
	if err := docs([]string{"button"}, io.Discard); !errors.As(err, &usage) {
		t.Errorf("docs button: got %v, want a usage error", err)
	}
	if err := docs([]string{"-page", "slider"}, io.Discard); err == nil || !strings.Contains(err.Error(), `page "slider"`) {
		t.Errorf("docs -page slider: got %v, want an unknown page", err)
	}
	if err := docs([]string{"-theme", "zinc"}, io.Discard); err == nil || !strings.Contains(err.Error(), `theme "zinc"`) {
		t.Errorf("docs -theme zinc: got %v, want an unknown theme", err)
	}
}
