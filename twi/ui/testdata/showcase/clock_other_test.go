//go:build !windows

package main

import (
	"testing"
	"time"
)

func precise(*testing.B) func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}
