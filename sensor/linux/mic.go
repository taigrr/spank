//go:build linux

package mic

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"sort"
	"time"
)

// Config holds configuration for mic-based knock detection.
type Config struct {
	Device     string        // ALSA device, e.g. "hw:0,6" or "default"
	Channels   int           // number of channels (4 for sof-hda-dsp DMIC)
	SampleRate int           // sample rate in Hz
	Threshold  float64       // minimum RMS to trigger (0.0-1.0)
	Cooldown   time.Duration // minimum time between triggers
}

// Event represents a detected knock/slap event.
type Event struct {
	Amplitude float64
	Time      time.Time
}

// DefaultConfig returns a Config suitable for sof-hda-dsp DMIC hardware
// (e.g. HP ZenBook with Intel Sound Open Firmware).
func DefaultConfig() Config {
	return Config{
		Device:     "hw:0,6",
		Channels:   4,
		SampleRate: 48000,
		Threshold:  0.02,
		Cooldown:   750 * time.Millisecond,
	}
}

const (
	// noiseHistorySize is the number of chunks used for the rolling noise floor.
	// At 10ms per chunk this is ~1 second of history.
	noiseHistorySize = 100

	// chunkDuration is how much audio we process per tick.
	chunkDuration = 10 * time.Millisecond
)

// Run starts arecord and reads raw S16_LE PCM data, sending Event values to
// eventCh whenever a knock louder than the effective threshold is detected.
// It returns when ctx is cancelled or a fatal error occurs.
func Run(ctx context.Context, cfg Config, eventCh chan<- Event) error {
	const bytesPerSample = 2 // S16_LE

	samplesPerChunk := int(float64(cfg.SampleRate) * chunkDuration.Seconds())
	chunkBytes := samplesPerChunk * cfg.Channels * bytesPerSample

	args := []string{
		"-D", cfg.Device,
		"-r", fmt.Sprintf("%d", cfg.SampleRate),
		"-c", fmt.Sprintf("%d", cfg.Channels),
		"-f", "S16_LE",
		"-t", "raw",
	}

	cmd := exec.CommandContext(ctx, "arecord", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mic: creating stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mic: starting arecord: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	buf := make([]byte, chunkBytes)
	noiseHistory := make([]float64, 0, noiseHistorySize)
	var lastTrigger time.Time

	for {
		// Check for cancellation or arecord exit before blocking on read.
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				return fmt.Errorf("mic: arecord exited: %w", err)
			}
			return fmt.Errorf("mic: arecord exited unexpectedly")
		default:
		}

		if _, err := io.ReadFull(stdout, buf); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("mic: reading audio: %w", err)
		}

		rms := computeRMS(buf, cfg.Channels)

		// Update rolling noise-floor history.
		noiseHistory = append(noiseHistory, rms)
		if len(noiseHistory) > noiseHistorySize {
			noiseHistory = noiseHistory[1:]
		}

		// Effective threshold adapts to ambient noise level.
		noiseFloor := median(noiseHistory)
		effectiveThreshold := math.Max(cfg.Threshold, 4.0*noiseFloor)

		if rms >= effectiveThreshold {
			now := time.Now()
			if now.Sub(lastTrigger) >= cfg.Cooldown {
				lastTrigger = now
				select {
				case eventCh <- Event{Amplitude: rms, Time: now}:
				default:
				}
			}
		}
	}
}

// computeRMS computes the RMS amplitude of an S16_LE interleaved PCM chunk
// averaged across all channels. Returns a normalised value in [0.0, 1.0].
func computeRMS(data []byte, channels int) float64 {
	if len(data) == 0 || channels <= 0 {
		return 0
	}

	const bytesPerSample = 2
	frameSize := channels * bytesPerSample
	if len(data) < frameSize {
		return 0
	}
	numFrames := len(data) / frameSize

	var sumSq float64
	totalSamples := numFrames * channels

	for i := 0; i < numFrames; i++ {
		for c := 0; c < channels; c++ {
			offset := i*frameSize + c*bytesPerSample
			sample := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
			normalized := float64(sample) / 32768.0
			sumSq += normalized * normalized
		}
	}

	return math.Sqrt(sumSq / float64(totalSamples))
}

// median returns the median value of a float64 slice.
// Returns 0.0 for an empty slice. Does not modify the input slice.
func median(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
