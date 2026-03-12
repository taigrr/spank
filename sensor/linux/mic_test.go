//go:build linux

package mic

import (
	"encoding/binary"
	"math"
	"testing"
)

// makeS16LEBytes encodes a slice of int16 samples as S16_LE bytes.
func makeS16LEBytes(samples []int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	return buf
}

func TestComputeRMSSilence(t *testing.T) {
	// All-zero samples → RMS should be exactly 0.
	samples := make([]int16, 480*4) // 480 frames, 4 channels
	data := makeS16LEBytes(samples)
	got := computeRMS(data, 4)
	if got != 0.0 {
		t.Errorf("expected 0.0 for silence, got %f", got)
	}
}

func TestComputeRMSMaxAmplitude(t *testing.T) {
	// All samples at maximum positive int16 value.
	// Normalised sample = 32767/32768 ≈ 1.0.
	// RMS ≈ 1.0 (within rounding of int16 max vs 32768 divisor).
	channels := 4
	frames := 480
	samples := make([]int16, frames*channels)
	for i := range samples {
		samples[i] = 32767
	}
	data := makeS16LEBytes(samples)
	got := computeRMS(data, channels)
	want := 32767.0 / 32768.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected ~%f for max amplitude, got %f", want, got)
	}
}

func TestComputeRMSKnownValue(t *testing.T) {
	// 4-channel frame with known values: [16384, -16384, 16384, -16384]
	// normalised: [0.5, -0.5, 0.5, -0.5]
	// RMS = sqrt(mean(0.25, 0.25, 0.25, 0.25)) = sqrt(0.25) = 0.5
	channels := 4
	samples := []int16{16384, -16384, 16384, -16384}
	data := makeS16LEBytes(samples)
	got := computeRMS(data, channels)
	want := 0.5
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("expected %f, got %f", want, got)
	}
}

func TestComputeRMSSingleChannel(t *testing.T) {
	// Single channel, single sample at half amplitude.
	// normalised = 16384 / 32768 = 0.5, RMS = 0.5
	samples := []int16{16384}
	data := makeS16LEBytes(samples)
	got := computeRMS(data, 1)
	want := 0.5
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("expected %f, got %f", want, got)
	}
}

func TestComputeRMSEmpty(t *testing.T) {
	got := computeRMS([]byte{}, 4)
	if got != 0.0 {
		t.Errorf("expected 0.0 for empty input, got %f", got)
	}
}

func TestMedianOdd(t *testing.T) {
	// Odd length: median is the middle element after sorting.
	values := []float64{3.0, 1.0, 2.0}
	got := median(values)
	want := 2.0
	if got != want {
		t.Errorf("expected %f, got %f", want, got)
	}
}

func TestMedianEven(t *testing.T) {
	// Even length: median is average of two middle elements.
	values := []float64{4.0, 1.0, 3.0, 2.0}
	got := median(values)
	want := 2.5
	if got != want {
		t.Errorf("expected %f, got %f", want, got)
	}
}

func TestMedianSingle(t *testing.T) {
	values := []float64{7.0}
	got := median(values)
	if got != 7.0 {
		t.Errorf("expected 7.0, got %f", got)
	}
}

func TestMedianEmpty(t *testing.T) {
	got := median([]float64{})
	if got != 0.0 {
		t.Errorf("expected 0.0 for empty slice, got %f", got)
	}
}

func TestMedianDoesNotMutateInput(t *testing.T) {
	values := []float64{3.0, 1.0, 2.0}
	original := make([]float64, len(values))
	copy(original, values)
	median(values)
	for i, v := range values {
		if v != original[i] {
			t.Errorf("median mutated input at index %d: got %f, want %f", i, v, original[i])
		}
	}
}
