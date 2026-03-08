package main

// AccelSample matches the structure used in the detector
type AccelSample struct {
	X, Y, Z float64
}

// RingBuffer defines the interface for accelerometer data storage
type RingBuffer interface {
	ReadNew(lastTotal uint64, scale float64) ([]AccelSample, uint64)
	Close() error
	Unlink() error
}
