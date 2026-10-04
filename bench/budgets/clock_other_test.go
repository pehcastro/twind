//go:build !windows

package budgets_test

import (
	"testing"
	"time"
)

func cycles() uint64 { return 0 }

func clock(testing.TB) func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}
