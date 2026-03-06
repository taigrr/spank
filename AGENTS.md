# AGENTS.md

> Guidelines for AI agents working in this repository.

## Project Overview

**spank** is a macOS CLI tool that detects physical hits/slaps on Apple Silicon MacBooks via the accelerometer and plays audio responses. Single-file Go application with embedded MP3 assets.

- **Platform**: macOS on Apple Silicon (M2+) only
- **Runtime requirement**: `sudo` (for IOKit HID accelerometer access)
- **Architecture**: Single `main.go` file with embedded audio assets

## Commands

### Build & Run

```bash
# Build
go build -o spank .

# Run (requires sudo)
sudo ./spank
sudo ./spank --sexy      # escalating responses mode
sudo ./spank --halo      # Halo death sounds mode
sudo ./spank --custom /path/to/mp3s  # custom audio directory
```

### Install

```bash
go install github.com/taigrr/spank@latest
```

### Release

Releases are automated via GitHub Actions + GoReleaser Pro when a `v*` tag is pushed:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Code Organization

```
spank/
├── main.go              # All application code (single file)
├── audio/
│   ├── pain/            # Default "ow!" responses (10 MP3s)
│   ├── sexy/            # Escalating responses (60 MP3s)
│   ├── halo/            # Halo death sounds (9 MP3s)
│   └── laugh/           # Laugh audio clips (10 MP3s)
├── go.mod
├── .goreleaser.yaml     # Release configuration
└── .github/workflows/   # CI/CD
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/taigrr/apple-silicon-accelerometer` | Reads accelerometer via IOKit HID |
| `github.com/gopxl/beep/v2` | Audio playback (MP3 decoding, speaker output) |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/charmbracelet/fang` | CLI config/execution wrapper |

## Code Patterns

### Embedded Assets

Audio files are embedded at compile time using `//go:embed`:

```go
//go:embed audio/pain/*.mp3
var painAudio embed.FS
```

### Play Modes

Two playback strategies in `playMode`:
- `modeRandom`: Random file selection (pain, halo, custom modes)
- `modeEscalation`: Intensity increases with slap frequency (sexy mode)

### Slap Detection Flow

1. `sensor.Run()` reads accelerometer in background goroutine
2. Data shared via `shm.RingBuffer` (POSIX shared memory)
3. `detector.New()` processes samples with vibration detection algorithms
4. Events trigger audio playback with 750ms cooldown

### Concurrency

- `speakerMu sync.Mutex` protects speaker initialization
- `slapTracker.mu sync.Mutex` protects slap scoring state
- Audio playback runs in goroutines (`go playAudio(...)`)

## Constants

Key tuning parameters in `main.go`:

| Constant | Value | Purpose |
|----------|-------|---------|
| `decayHalfLife` | 30s | How fast escalation fades |
| `slapCooldown` | 750ms | Minimum time between audio plays |
| `sensorPollInterval` | 10ms | Accelerometer polling rate |
| `maxSampleBatch` | 200 | Max samples processed per tick |

## Gotchas

1. **Root required**: The app must run with `sudo` for IOKit HID access. The `run()` function checks `os.Geteuid() != 0`.

2. **Apple Silicon only**: Only builds for `darwin/arm64`. Intel Macs are not supported.

3. **Private dependency**: `github.com/taigrr/apple-silicon-accelerometer` requires `GOPRIVATE` setting and GitHub PAT for CI.

4. **Single file**: All code is in `main.go`. When adding features, follow the existing pattern of types and functions in the same file.

5. **Mutually exclusive modes**: `--sexy`, `--halo`, `--laugh`, and `--custom` flags cannot be combined.

6. **CGO disabled**: Builds use `CGO_ENABLED=0` despite targeting macOS.

## Adding Audio

To add a new sound pack:

1. Create directory under `audio/`
2. Add MP3 files (numbered for escalation mode, any names for random)
3. Add `//go:embed audio/newpack/*.mp3` variable
4. Add flag and case in `run()` switch statement
5. Create `soundPack` with appropriate `playMode`

## Version

Version is injected via ldflags at build time:

```go
var version = "dev"
```

GoReleaser sets `-X main.version={{.Version}}` during release builds.
