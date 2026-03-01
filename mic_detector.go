// mic_detector.go implements audio-based impact detection for microphone mode.
// Uses STA/LTA ratio on RMS amplitude to detect sudden sounds (slaps,
// impacts, taps) while adapting to background noise.
//
// A laptop slap produces a low-level thud that is much quieter than a hand
// clap. To handle this, we use the STA/LTA *ratio* as the primary signal
// (detecting sudden changes relative to background) rather than requiring
// high absolute loudness.
package main

import (
	"math"
	"time"

	"github.com/taigrr/apple-silicon-accelerometer/detector"
)

const (
	// micSampleRate matches the Core Audio capture rate.
	micSampleRate = 44100

	// micFrameSize is the number of samples per analysis frame (~23ms).
	micFrameSize = 1024

	// micSTAFrames is the short-term average window in frames (~46ms).
	micSTAFrames = 2

	// micLTAFrames is the long-term average window in frames (~1.6s).
	micLTAFrames = 70

	// micSTALTAThreshold is the STA/LTA ratio that triggers an event.
	// Lower = more sensitive. A laptop slap typically produces a ratio
	// of 2-4, while a hand clap produces 10+.
	micSTALTAThreshold = 1.8

	// micMinEventGap prevents duplicate detections within this duration.
	micMinEventGap = 50 * time.Millisecond

	// micWarmupFrames is the number of frames before detection starts.
	// Shorter warmup means faster startup but slightly noisier initial LTA.
	micWarmupFrames = 30
)

// MicDetector analyzes audio frames to detect impact events.
type MicDetector struct {
	sta       float64
	lta       float64
	staAlpha  float64
	ltaAlpha  float64
	peakRMS   float64 // tracks recent peak for amplitude normalization
	ready     bool
	warmup    int
	lastEvent time.Time
}

// NewMicDetector creates a new audio impact detector.
func NewMicDetector() *MicDetector {
	return &MicDetector{
		staAlpha: 1.0 / float64(micSTAFrames),
		ltaAlpha: 1.0 / float64(micLTAFrames),
	}
}

// ProcessFrame analyzes a frame of audio samples and returns a
// detector.Event if an impact is detected, or nil otherwise.
// Samples should be float32 in [-1.0, 1.0].
func (md *MicDetector) ProcessFrame(samples []float32) *detector.Event {
	if len(samples) == 0 {
		return nil
	}

	// Calculate RMS energy for this frame
	var sumSq float64
	var peak float64
	for _, s := range samples {
		v := float64(s)
		sumSq += v * v
		if abs := math.Abs(v); abs > peak {
			peak = abs
		}
	}
	rms := math.Sqrt(sumSq / float64(len(samples)))

	// Track peak RMS with slow decay for amplitude normalization
	if rms > md.peakRMS {
		md.peakRMS = rms
	} else {
		md.peakRMS *= 0.999 // slow decay
	}

	// Exponential moving average for STA and LTA
	energy := rms * rms
	if !md.ready {
		md.sta = energy
		md.lta = energy
		md.ready = true
		md.warmup = 0
		return nil
	}

	md.sta += md.staAlpha * (energy - md.sta)
	md.lta += md.ltaAlpha * (energy - md.lta)

	// Need warmup period for LTA to stabilize
	md.warmup++
	if md.warmup < micWarmupFrames {
		return nil
	}

	// STA/LTA ratio — this is the key metric.
	// It measures how much the current energy exceeds the background,
	// so even a quiet laptop thud will spike the ratio if the room is quiet.
	ratio := md.sta / (md.lta + 1e-30)

	now := time.Now()

	// Check for impact
	if ratio > micSTALTAThreshold && now.Sub(md.lastEvent) > micMinEventGap {
		md.lastEvent = now

		// Use the ratio itself to derive amplitude, not just raw RMS.
		// This way a laptop slap in a quiet room gives a similar amplitude
		// to a clap in a noisy room — both represent "sudden change".
		// Map ratio to 0-1: ratio of 1.8 → ~0.05, ratio of 20 → ~1.0
		ratioAmp := math.Min(1.0, (ratio-1.0)/15.0)

		// Also factor in absolute RMS scaled up aggressively
		rmsAmp := math.Min(1.0, rms*20.0)

		// Combined amplitude: primarily ratio-driven, boosted by RMS
		amplitude := math.Max(ratioAmp, rmsAmp)

		// Classify severity based on ratio strength
		var severity string
		switch {
		case ratio > 10.0:
			severity = "CHOC_MAJEUR"
		case ratio > 5.0:
			severity = "CHOC_MOYEN"
		case ratio > 2.5:
			severity = "MICRO_CHOC"
		default:
			severity = "VIBRATION"
		}

		return &detector.Event{
			Time:      now,
			Severity:  severity,
			Symbol:    "★",
			Label:     "mic-impact",
			Amplitude: amplitude,
			Sources:   []string{"MIC_STA_LTA"},
		}
	}

	return nil
}
