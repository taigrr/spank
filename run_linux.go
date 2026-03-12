//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mic "github.com/taigrr/spank/sensor/linux"
)

// runAccel is a stub on Linux: the Apple Silicon accelerometer is not available.
func runAccel(_ context.Context, _ *soundPack, _ runtimeTuning) error {
	return fmt.Errorf("accelerometer detection requires macOS on Apple Silicon; use --mic for microphone-based detection on Linux")
}

// runMic runs the microphone-based knock detection path using arecord (ALSA).
// It does not require root privileges.
func runMic(ctx context.Context, pack *soundPack, tuning runtimeTuning) error {
	cfg := mic.Config{
		Device:     micDevice,
		Channels:   4,
		SampleRate: 48000,
		Threshold:  micThreshold,
		Cooldown:   tuning.cooldown,
	}

	eventCh := make(chan mic.Event, 16)

	go func() {
		if err := mic.Run(ctx, cfg, eventCh); err != nil {
			// Only log errors that aren't due to context cancellation.
			if ctx.Err() == nil {
				fmt.Printf("spank: mic error: %v\n", err)
			}
		}
	}()

	return listenForMicSlaps(ctx, pack, eventCh, tuning)
}

// listenForMicSlaps mirrors listenForSlaps but reads from a mic.Event channel
// instead of polling the accelerometer ring buffer.
func listenForMicSlaps(ctx context.Context, pack *soundPack, eventCh <-chan mic.Event, tuning runtimeTuning) error {
	tracker := newSlapTracker(pack, tuning.cooldown)
	speakerInit := false
	var lastYell time.Time

	// Start stdin command reader if in JSON mode.
	if stdioMode {
		go readStdinCommands()
	}

	presetLabel := "default"
	if fastMode {
		presetLabel = "fast"
	}
	fmt.Printf("spank: listening for knocks via mic in %s mode with %s tuning... (ctrl+c to quit)\n", pack.name, presetLabel)
	if stdioMode {
		fmt.Println(`{"status":"ready"}`)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nbye!")
			return nil
		case ev, ok := <-eventCh:
			if !ok {
				return nil
			}

			// Check if paused.
			pausedMu.RLock()
			isPaused := paused
			pausedMu.RUnlock()
			if isPaused {
				continue
			}

			now := ev.Time

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
					"file":       file,
				}
				if data, err := json.Marshal(event); err == nil {
					fmt.Println(string(data))
				}
			} else {
				fmt.Printf("slap #%d [amp=%.5f] -> %s\n", num, ev.Amplitude, file)
			}
			go playAudio(pack, file, ev.Amplitude, &speakerInit)
		}
	}
}
