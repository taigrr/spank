//go:build !windows

package main

import "context"

// startMouseTrigger is a no-op on non-Windows platforms.
func startMouseTrigger(ctx context.Context, trigger func()) {
}
