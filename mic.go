// mic.go provides microphone-based impact detection as an alternative to
// the accelerometer. This enables spank to work on M1 Macs (and any Mac)
// by detecting slap sounds through the built-in microphone.
//
// Uses macOS Core Audio (Audio Queue Services) via CGo.
package main

/*
#cgo LDFLAGS: -framework AudioToolbox -framework CoreFoundation
#include <AudioToolbox/AudioToolbox.h>
#include <stdlib.h>
#include <string.h>

// Ring buffer for audio samples shared between callback and Go.
// Lock-free single-producer (callback) single-consumer (Go) ring.
#define MIC_RING_SIZE 32768

static float micRing[MIC_RING_SIZE];
static volatile int64_t micWriteIdx = 0;

static void micCallback(
    void *inUserData,
    AudioQueueRef inAQ,
    AudioQueueBufferRef inBuffer,
    const AudioTimeStamp *inStartTime,
    UInt32 inNumPackets,
    const AudioStreamPacketDescription *inPacketDesc)
{
    int16_t *samples = (int16_t *)inBuffer->mAudioData;
    UInt32 count = inBuffer->mAudioDataByteSize / sizeof(int16_t);

    for (UInt32 i = 0; i < count; i++) {
        int64_t idx = __sync_fetch_and_add(&micWriteIdx, 1);
        micRing[idx % MIC_RING_SIZE] = (float)samples[i] / 32768.0f;
    }

    AudioQueueEnqueueBuffer(inAQ, inBuffer, 0, NULL);
}

// startMicCapture opens the default input device and begins recording.
// Returns 0 on success, non-zero on failure.
static int startMicCapture(AudioQueueRef *outQueue) {
    AudioStreamBasicDescription fmt;
    memset(&fmt, 0, sizeof(fmt));
    fmt.mSampleRate       = 44100.0;
    fmt.mFormatID         = kAudioFormatLinearPCM;
    fmt.mFormatFlags      = kLinearPCMFormatFlagIsSignedInteger | kLinearPCMFormatFlagIsPacked;
    fmt.mBitsPerChannel   = 16;
    fmt.mChannelsPerFrame = 1;
    fmt.mFramesPerPacket  = 1;
    fmt.mBytesPerFrame    = 2;
    fmt.mBytesPerPacket   = 2;

    OSStatus status = AudioQueueNewInput(
        &fmt,
        micCallback,
        NULL,
        NULL,           // run loop (NULL = internal)
        NULL,           // run loop mode
        0,              // flags
        outQueue
    );
    if (status != 0) return (int)status;

    // Allocate 3 buffers of 1024 samples each (~23ms at 44.1kHz)
    for (int i = 0; i < 3; i++) {
        AudioQueueBufferRef buf;
        status = AudioQueueAllocateBuffer(*outQueue, 1024 * 2, &buf);
        if (status != 0) return (int)status;
        status = AudioQueueEnqueueBuffer(*outQueue, buf, 0, NULL);
        if (status != 0) return (int)status;
    }

    status = AudioQueueStart(*outQueue, NULL);
    return (int)status;
}

static void stopMicCapture(AudioQueueRef queue) {
    AudioQueueStop(queue, true);
    AudioQueueDispose(queue, true);
}

static int64_t getMicWriteIdx() {
    return micWriteIdx;
}

static float getMicSample(int64_t idx) {
    return micRing[idx % MIC_RING_SIZE];
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

const micRingSize = C.MIC_RING_SIZE

// MicCapture manages microphone recording via Core Audio.
type MicCapture struct {
	queue    C.AudioQueueRef
	lastRead int64
}

// NewMicCapture creates and starts microphone capture.
func NewMicCapture() (*MicCapture, error) {
	var queue C.AudioQueueRef
	ret := C.startMicCapture(&queue)
	if ret != 0 {
		return nil, fmt.Errorf("failed to start mic capture (Core Audio error %d); check microphone permissions in System Settings > Privacy > Microphone", ret)
	}
	mc := &MicCapture{
		queue:    queue,
		lastRead: int64(C.getMicWriteIdx()),
	}
	return mc, nil
}

// ReadNew returns new audio samples since the last call.
// Samples are float32 in [-1.0, 1.0].
func (mc *MicCapture) ReadNew() []float32 {
	writeIdx := int64(C.getMicWriteIdx())
	avail := writeIdx - mc.lastRead
	if avail <= 0 {
		return nil
	}
	if avail > micRingSize {
		// Overrun — skip to latest data
		mc.lastRead = writeIdx - micRingSize/2
		avail = writeIdx - mc.lastRead
	}

	samples := make([]float32, avail)
	for i := int64(0); i < avail; i++ {
		samples[i] = float32(C.getMicSample(C.int64_t(mc.lastRead + i)))
	}
	mc.lastRead = writeIdx
	return samples
}

// Close stops and releases the mic capture resources.
func (mc *MicCapture) Close() {
	C.stopMicCapture(mc.queue)
	mc.queue = (C.AudioQueueRef)(unsafe.Pointer(nil))
}
