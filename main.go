// spank detects slaps/hits on the laptop and plays audio responses.
// It reads the Apple Silicon accelerometer directly via IOKit HID —
// no separate sensor daemon required. Needs sudo.
package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/spf13/cobra"
	"github.com/taigrr/apple-silicon-accelerometer/detector"
	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

var version = "dev"

//go:embed audio/pain/*.mp3
var painAudio embed.FS

//go:embed audio/sexy/*.mp3
var sexyAudio embed.FS

//go:embed audio/halo/*.mp3
var haloAudio embed.FS

var (
	sexyMode     bool
	haloMode     bool
	minAmplitude float64
)

// sensorReady is closed once shared memory is created and the sensor
// worker is about to enter the CFRunLoop.
var sensorReady = make(chan struct{})

// sensorErr receives any error from the sensor worker.
var sensorErr = make(chan error, 1)

type playMode int

const (
	modeRandom playMode = iota
	modeEscalation
)

type soundPack struct {
	name  string
	fs    embed.FS
	dir   string
	mode  playMode
	files []string
}

func (sp *soundPack) loadFiles() error {
	entries, err := sp.fs.ReadDir(sp.dir)
	if err != nil {
		return err
	}
	sp.files = make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			sp.files = append(sp.files, sp.dir+"/"+e.Name())
		}
	}
	sort.Strings(sp.files)
	return nil
}

type slapTracker struct {
	mu       sync.Mutex
	score    float64
	lastTime time.Time
	total    int
	halfLife float64 // seconds
	scale    float64 // controls the escalation curve shape
	pack     *soundPack
}

func newSlapTracker(pack *soundPack) *slapTracker {
	halfLife := 30.0 // seconds — intensity halves every 30s of inactivity
	// scale is derived so that the theoretical max steady-state score
	// (at peak slap rate with 500ms cooldown) maps to approximately the
	// second-to-last file, making the last file asymptotically unreachable.
	ssMax := 1.0 / (1.0 - math.Pow(0.5, 0.5/halfLife))
	scale := (ssMax - 1) / math.Log(float64(len(pack.files)))
	return &slapTracker{
		halfLife: halfLife,
		scale:   scale,
		pack:    pack,
	}
}

func (st *slapTracker) record(t time.Time) (int, float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.lastTime.IsZero() {
		elapsed := t.Sub(st.lastTime).Seconds()
		st.score *= math.Pow(0.5, elapsed/st.halfLife)
	}
	st.score += 1.0
	st.lastTime = t
	st.total++
	return st.total, st.score
}

func (st *slapTracker) getFile(score float64) string {
	if len(st.pack.files) == 0 {
		return ""
	}

	if st.pack.mode == modeRandom {
		return st.pack.files[rand.Intn(len(st.pack.files))]
	}

	// Escalation: 1-exp(-x) curve asymptotically approaches the top
	// without ever reaching it. Slap faster to climb; slow down to decay.
	maxIdx := len(st.pack.files) - 1
	idx := int(float64(maxIdx) * (1.0 - math.Exp(-(score-1)/st.scale)))
	if idx > maxIdx {
		idx = maxIdx
	}
	return st.pack.files[idx]
}

func main() {
	cmd := &cobra.Command{
		Use:   "spank",
		Short: "Yells 'ow!' when you slap the laptop",
		Long: `spank reads the Apple Silicon accelerometer directly via IOKit HID
and plays audio responses when a slap or hit is detected.

Requires sudo (for IOKit HID access to the accelerometer).

Use --sexy for a different experience. In sexy mode, the more you slap
within a minute, the more intense the sounds become.

Use --halo to play random audio clips from Halo soundtracks on each slap.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context())
		},
		SilenceUsage: true,
	}

	cmd.Flags().BoolVarP(&sexyMode, "sexy", "s", false, "Enable sexy mode")
	cmd.Flags().BoolVarP(&haloMode, "halo", "H", false, "Enable halo mode")
	cmd.Flags().Float64Var(&minAmplitude, "min-amplitude", 0.3, "Minimum amplitude threshold (0.0-1.0, lower = more sensitive)")

	if err := fang.Execute(context.Background(), cmd); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("spank requires root privileges for accelerometer access, run with: sudo spank")
	}

	if sexyMode && haloMode {
		return fmt.Errorf("--sexy and --halo are mutually exclusive; pick one")
	}

	if minAmplitude < 0 || minAmplitude > 1 {
		return fmt.Errorf("--min-amplitude must be between 0.0 and 1.0")
	}

	var pack *soundPack
	switch {
	case sexyMode:
		pack = &soundPack{name: "sexy", fs: sexyAudio, dir: "audio/sexy", mode: modeEscalation}
	case haloMode:
		pack = &soundPack{name: "halo", fs: haloAudio, dir: "audio/halo", mode: modeRandom}
	default:
		pack = &soundPack{name: "pain", fs: painAudio, dir: "audio/pain", mode: modeRandom}
	}

	if err := pack.loadFiles(); err != nil {
		return fmt.Errorf("loading %s audio: %w", pack.name, err)
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create shared memory for accelerometer data.
	accelRing, err := shm.CreateRing(shm.NameAccel)
	if err != nil {
		return fmt.Errorf("creating accel shm: %w", err)
	}
	defer accelRing.Close()
	defer accelRing.Unlink()

	// Start the sensor worker in a background goroutine.
	// sensor.Run() needs runtime.LockOSThread for CFRunLoop, which it
	// handles internally. We launch detection on the current goroutine.
	go func() {
		close(sensorReady)
		err := sensor.Run(sensor.Config{
			AccelRing: accelRing,
			Restarts:  0,
		})
		if err != nil {
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
	time.Sleep(100 * time.Millisecond)

	tracker := newSlapTracker(pack)
	speakerInit := false
	det := detector.New()
	var lastAccelTotal uint64
	var lastEventTime time.Time
	lastYell := time.Time{}
	cooldown := 500 * time.Millisecond
	maxBatch := 200

	fmt.Printf("spank: listening for slaps in %s mode... (ctrl+c to quit)\n", pack.name)

	ticker := time.NewTicker(10 * time.Millisecond)
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

		now := time.Now()
		tNow := float64(now.UnixNano()) / 1e9

		samples, newTotal := accelRing.ReadNew(lastAccelTotal, shm.AccelScale)
		lastAccelTotal = newTotal
		if len(samples) > maxBatch {
			samples = samples[len(samples)-maxBatch:]
		}

		nSamples := len(samples)
		for idx, s := range samples {
			tSample := tNow - float64(nSamples-idx-1)/float64(det.FS)
			det.Process(s.X, s.Y, s.Z, tSample)
		}

		newEventIdx := len(det.Events)
		if newEventIdx > 0 {
			ev := det.Events[newEventIdx-1]
			if ev.Time != lastEventTime {
				lastEventTime = ev.Time
				if time.Since(lastYell) > cooldown {
					if ev.Amplitude >= minAmplitude {
						lastYell = now
						num, score := tracker.record(now)
						file := tracker.getFile(score)
						fmt.Printf("slap #%d [%s amp=%.5fg] -> %s\n", num, ev.Severity, ev.Amplitude, file)
						go playEmbedded(pack.fs, file, &speakerInit)
					}
				}
			}
		}
	}
}

var speakerMu sync.Mutex

func playEmbedded(fs embed.FS, path string, speakerInit *bool) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return
	}

	streamer, format, err := mp3.Decode(io.NopCloser(bytes.NewReader(data)))
	if err != nil {
		return
	}
	defer streamer.Close()

	speakerMu.Lock()
	if !*speakerInit {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		*speakerInit = true
	}
	speakerMu.Unlock()

	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))
	<-done
}
