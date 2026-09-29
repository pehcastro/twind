//go:build !windows

package budgets_test

import (
	"testing"
	"time"
)

func clock(*testing.B) func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}
