package main

import (
	"testing"

	"github.com/gopxl/beep/v2"
)

// stubStreamer is a no-op beep.Streamer used to exercise resampling wiring.
type stubStreamer struct{}

func (stubStreamer) Stream(samples [][2]float64) (int, bool) { return 0, false }
func (stubStreamer) Err() error                              { return nil }

func TestResampleForPlayback(t *testing.T) {
	src := stubStreamer{}

	tests := []struct {
		name         string
		fileRate     beep.SampleRate
		speakerRate  beep.SampleRate
		speed        float64
		wantResample bool
		wantRatio    float64
	}{
		{
			name:         "matching rate normal speed returns source unchanged",
			fileRate:     44100,
			speakerRate:  44100,
			speed:        1.0,
			wantResample: false,
		},
		{
			name:         "48k file on 44.1k speaker resamples down",
			fileRate:     48000,
			speakerRate:  44100,
			speed:        1.0,
			wantResample: true,
			wantRatio:    48000.0 / 44100.0,
		},
		{
			name:         "44.1k file on 48k speaker resamples up",
			fileRate:     44100,
			speakerRate:  48000,
			speed:        1.0,
			wantResample: true,
			wantRatio:    44100.0 / 48000.0,
		},
		{
			name:         "speed change folds into resample even when rates match",
			fileRate:     44100,
			speakerRate:  44100,
			speed:        2.0,
			wantResample: true,
			wantRatio:    88200.0 / 44100.0,
		},
		{
			name:         "speed and rate mismatch combine",
			fileRate:     48000,
			speakerRate:  44100,
			speed:        1.5,
			wantResample: true,
			wantRatio:    72000.0 / 44100.0,
		},
		{
			name:         "zero speed is ignored",
			fileRate:     44100,
			speakerRate:  44100,
			speed:        0,
			wantResample: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resampleForPlayback(src, tt.fileRate, tt.speakerRate, tt.speed)
			r, isResampler := got.(*beep.Resampler)
			if tt.wantResample != isResampler {
				t.Fatalf("wantResample=%v, got *beep.Resampler=%v", tt.wantResample, isResampler)
			}
			if !tt.wantResample {
				if got != beep.Streamer(src) {
					t.Fatalf("expected original source to be returned unchanged")
				}
				return
			}
			const eps = 1e-9
			if diff := r.Ratio() - tt.wantRatio; diff > eps || diff < -eps {
				t.Fatalf("ratio = %v, want %v", r.Ratio(), tt.wantRatio)
			}
		})
	}
}
