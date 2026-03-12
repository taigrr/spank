//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/taigrr/apple-silicon-accelerometer/detector"
	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

// sensorReady is closed once shared memory is created and the sensor
// worker is about to enter the CFRunLoop.
var sensorReady = make(chan struct{})

// sensorErr receives any error from the sensor worker.
var sensorErr = make(chan error, 1)

// sensorStartupDelay gives the sensor time to start producing data.
const sensorStartupDelay = 100 * time.Millisecond

// runAccel runs the Apple Silicon accelerometer detection path.
// It requires root (for IOKit HID access).
func runAccel(ctx context.Context, pack *soundPack, tuning runtimeTuning) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("spank requires root privileges for accelerometer access, run with: sudo spank")
	}

	// Create shared memory for accelerometer data.
	accelRing, err := shm.CreateRing(shm.NameAccel)
	if err != nil {
		return fmt.Errorf("creating accel shm: %w", err)
	}
	defer accelRing.Close()
	defer accelRing.Unlink()

	// Start the sensor worker in a background goroutine.
	// sensor.Run() needs runtime.LockOSThread for CFRunLoop, which it
	// handles internally.
	go func() {
		close(sensorReady)
		if err := sensor.Run(sensor.Config{
			AccelRing: accelRing,
			Restarts:  0,
		}); err != nil {
			sensorErr <- err
		}
	}()

	// Wait for sensor to be ready.
	select {
	case <-sensorReady:
	case err := <-sensorErr:
		return fmt.Errorf("sensor worker failed: %w", err)
	case <-ctx.Done():
		return nil
	}

	// Give the sensor a moment to start producing data.
	time.Sleep(sensorStartupDelay)

	return listenForSlaps(ctx, pack, accelRing, tuning)
}

func listenForSlaps(ctx context.Context, pack *soundPack, accelRing *shm.RingBuffer, tuning runtimeTuning) error {
	tracker := newSlapTracker(pack, tuning.cooldown)
	speakerInit := false
	det := detector.New()
	var lastAccelTotal uint64
	var lastEventTime time.Time
	var lastYell time.Time

	// Start stdin command reader if in JSON mode.
	if stdioMode {
		go readStdinCommands()
	}

	presetLabel := "default"
	if fastMode {
		presetLabel = "fast"
	}
	fmt.Printf("spank: listening for slaps in %s mode with %s tuning... (ctrl+c to quit)\n", pack.name, presetLabel)
	if stdioMode {
		fmt.Println(`{"status":"ready"}`)
	}

	ticker := time.NewTicker(tuning.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nbye!")
			return nil
		case err := <-sensorErr:
			return fmt.Errorf("sensor worker failed: %w", err)
		case <-ticker.C:
		}

		// Check if paused.
		pausedMu.RLock()
		isPaused := paused
		pausedMu.RUnlock()
		if isPaused {
			continue
		}

		now := time.Now()
		tNow := float64(now.UnixNano()) / 1e9

		samples, newTotal := accelRing.ReadNew(lastAccelTotal, shm.AccelScale)
		lastAccelTotal = newTotal
		if len(samples) > tuning.maxBatch {
			samples = samples[len(samples)-tuning.maxBatch:]
		}

		nSamples := len(samples)
		for idx, sample := range samples {
			tSample := tNow - float64(nSamples-idx-1)/float64(det.FS)
			det.Process(sample.X, sample.Y, sample.Z, tSample)
		}

		if len(det.Events) == 0 {
			continue
		}

		ev := det.Events[len(det.Events)-1]
		if ev.Time.Equal(lastEventTime) {
			continue
		}
		lastEventTime = ev.Time

		// Use globals so live-tuning via stdin takes effect immediately.
		if time.Since(lastYell) <= time.Duration(cooldownMs)*time.Millisecond {
			continue
		}
		if ev.Amplitude < minAmplitude {
			continue
		}

		lastYell = now
		num, score := tracker.record(now)
		file := tracker.getFile(score)
		if stdioMode {
			event := map[string]interface{}{
				"timestamp":  now.Format(time.RFC3339Nano),
				"slapNumber": num,
				"amplitude":  ev.Amplitude,
				"severity":   string(ev.Severity),
				"file":       file,
			}
			if data, err := json.Marshal(event); err == nil {
				fmt.Println(string(data))
			}
		} else {
			fmt.Printf("slap #%d [%s amp=%.5fg] -> %s\n", num, ev.Severity, ev.Amplitude, file)
		}
		go playAudio(pack, file, ev.Amplitude, &speakerInit)
	}
}
