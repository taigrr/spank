//go:build darwin

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

// shmRingWrapper wraps the original shm.RingBuffer to implement our RingBuffer interface
type shmRingWrapper struct {
	*shm.RingBuffer
}

func (w *shmRingWrapper) ReadNew(lastTotal uint64, scale float64) ([]AccelSample, uint64) {
	samples, newTotal := w.RingBuffer.ReadNew(lastTotal, scale)
	
	// Convert from shm.AccelSample to our local AccelSample
	// Assuming the structures are compatible or manually converting
	result := make([]AccelSample, len(samples))
	for i, s := range samples {
		result[i] = AccelSample{X: s.X, Y: s.Y, Z: s.Z}
	}
	return result, newTotal
}

func initSensor(ctx context.Context, simulate bool) (RingBuffer, error) {
	accelRing, err := shm.CreateRing(shm.NameAccel)
	if err != nil {
		return nil, fmt.Errorf("creating accel shm: %w", err)
	}

	go func() {
		if err := sensor.Run(sensor.Config{
			AccelRing: accelRing,
			Restarts:  0,
		}); err != nil {
			// In the original main.go, this was piped to sensorErr
			// For simplicity in this bridge, we'll log it
			fmt.Printf("sensor worker failed: %v\n", err)
		}
	}()

	// Give the sensor a moment to start producing data.
	time.Sleep(100 * time.Millisecond)

	return &shmRingWrapper{accelRing}, nil
}
