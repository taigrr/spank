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
	sexyMode   bool
	haloMode   bool
	customPath string
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
	name   string
	fs     embed.FS
	dir    string
	mode   playMode
	files  []string
	custom bool
}

func (sp *soundPack) loadFiles() error {
	if sp.custom {
		entries, err := os.ReadDir(sp.dir)
		if err != nil {
			return err
		}
		sp.files = make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				sp.files = append(sp.files, sp.dir+"/"+e.Name())
			}
		}
	} else {
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
	}
	sort.Strings(sp.files)
	return nil
}

type slapTracker struct {
	mu     sync.Mutex
	times  []time.Time
	window time.Duration
	pack   *soundPack
	altIdx int
}

func newSlapTracker(pack *soundPack) *slapTracker {
	return &slapTracker{
		window: 5 * time.Minute,
		pack:   pack,
	}
}

func (st *slapTracker) record(t time.Time) int {
	st.mu.Lock()
	defer st.mu.Unlock()

	cutoff := t.Add(-st.window)
	newTimes := make([]time.Time, 0, len(st.times)+1)
	for _, tt := range st.times {
		if tt.After(cutoff) {
			newTimes = append(newTimes, tt)
		}
	}
	newTimes = append(newTimes, t)
	st.times = newTimes
	return len(st.times)
}

func (st *slapTracker) getFile(count int) string {
	st.mu.Lock()
	defer st.mu.Unlock()

	if len(st.pack.files) == 0 {
		return ""
	}

	if st.pack.mode == modeRandom {
		return st.pack.files[rand.Intn(len(st.pack.files))]
	}

	// Escalation mode
	maxIdx := len(st.pack.files) - 1
	topTwo := maxIdx - 1
	if topTwo < 0 {
		topTwo = 0
	}

	var idx int
	if count >= 20 {
		st.altIdx = 1 - st.altIdx
		idx = topTwo + st.altIdx
	} else {
		ratio := float64(count) / 20.0
		if ratio > 1 {
			ratio = 1
		}
		idx = int(ratio * float64(topTwo))
	}

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
	cmd.Flags().StringVarP(&customPath, "custom", "c", "", "Path to custom MP3 audio directory")

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
	if customPath != "" && (sexyMode || haloMode) {
		return fmt.Errorf("--custom cannot be used with --sexy or --halo")
	}

	var pack *soundPack
	switch {
	case customPath != "":
		pack = &soundPack{name: "custom", dir: customPath, mode: modeRandom, custom: true}
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
					if ev.Severity == "CHOC_MAJEUR" || ev.Severity == "CHOC_MOYEN" || ev.Severity == "MICRO_CHOC" {
						lastYell = now
						count := tracker.record(now)
						file := tracker.getFile(count)
						fmt.Printf("slap #%d [%s amp=%.5fg] -> %s\n", count, ev.Severity, ev.Amplitude, file)
						go playAudio(pack, file, &speakerInit)
					}
				}
			}
		}
	}
}

var speakerMu sync.Mutex

func playAudio(pack *soundPack, path string, speakerInit *bool) {
	var streamer beep.StreamSeekCloser
	var format beep.Format

	if pack.custom {
		file, err := os.Open(path)
		if err != nil {
			return
		}
		defer file.Close()
		streamer, format, err = mp3.Decode(file)
		if err != nil {
			return
		}
	} else {
		data, err := pack.fs.ReadFile(path)
		if err != nil {
			return
		}
		streamer, format, err = mp3.Decode(io.NopCloser(bytes.NewReader(data)))
		if err != nil {
			return
		}
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
