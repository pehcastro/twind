//go:build !windows

package runtime_test

import (
	"testing"
	"time"
)

func stopwatch(*testing.B) func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}
