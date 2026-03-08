//go:build !darwin && !linux

package main

import (
	"context"
	"fmt"
)

func runAccel(_ context.Context, _ *soundPack, _ runtimeTuning) error {
	return fmt.Errorf("accelerometer detection is only supported on macOS Apple Silicon")
}

func runMic(_ context.Context, _ *soundPack, _ runtimeTuning) error {
	return fmt.Errorf("--mic is only supported on Linux; on macOS use the default accelerometer mode")
}
