package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/taigrr/spank/macos/internal/config"
)

// Manager controls the spankd subprocess and provides JSON IPC.
type Manager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	running bool

	// SlapEvents receives slap events from spankd (non-blocking send).
	SlapEvents chan SlapEvent
	// StatusEvents receives status/reply lines from spankd.
	StatusEvents chan StatusResponse
	// Errors receives fatal errors from the read loop.
	Errors chan error
}

// New creates a Manager with buffered event channels.
func New() *Manager {
	return &Manager{
		SlapEvents:   make(chan SlapEvent, 16),
		StatusEvents: make(chan StatusResponse, 8),
		Errors:       make(chan error, 1),
	}
}

// IsRunning reports whether spankd is currently running.
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// Start launches spankd with the given config as a sudo subprocess.
// Returns immediately; communication happens asynchronously via channels.
func (m *Manager) Start(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	spankd, err := SpankdPath()
	if err != nil {
		return err
	}

	args := buildArgs(cfg)
	m.cmd = exec.Command("sudo", append([]string{"-n", spankd}, args...)...)

	stdin, err := m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}
	m.stdin = stdin

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("starting spankd: %w", err)
	}

	m.running = true
	go m.readLoop(stdout)
	go m.waitLoop()
	return nil
}

// Stop sends SIGTERM to spankd and waits for it to exit.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	// Close stdin to signal EOF to spankd
	_ = m.stdin.Close()
	_ = m.cmd.Process.Kill()
	return nil
}

// Pause sends a pause command to spankd.
func (m *Manager) Pause() error {
	return m.sendCmd(Command{Cmd: "pause"})
}

// Resume sends a resume command to spankd.
func (m *Manager) Resume() error {
	return m.sendCmd(Command{Cmd: "resume"})
}

// UpdateSettings sends updated amplitude/cooldown/speed without restarting.
func (m *Manager) UpdateSettings(cfg *config.Config) error {
	return m.sendCmd(Command{
		Cmd:       "set",
		Amplitude: cfg.MinAmplitude,
		Cooldown:  cfg.Cooldown,
		Speed:     cfg.Speed,
	})
}

// RequestStatus asks spankd for its current status.
func (m *Manager) RequestStatus() error {
	return m.sendCmd(Command{Cmd: "status"})
}

func (m *Manager) sendCmd(cmd Command) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.stdin == nil {
		return fmt.Errorf("spankd is not running")
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(m.stdin, "%s\n", data)
	return err
}

// buildArgs constructs the spankd CLI arguments from config.
func buildArgs(cfg *config.Config) []string {
	args := []string{"--stdio"}
	switch cfg.Mode {
	case "sexy":
		args = append(args, "--sexy")
	case "halo":
		args = append(args, "--halo")
	// "pain" is the default, no flag needed
	}
	args = append(args,
		"--min-amplitude", fmt.Sprintf("%.4f", cfg.MinAmplitude),
		"--cooldown", fmt.Sprintf("%d", cfg.Cooldown),
		"--speed", fmt.Sprintf("%.2f", cfg.Speed),
	)
	if cfg.VolumeScaling {
		args = append(args, "--volume-scaling")
	}
	return args
}

// readLoop reads JSON lines from spankd's stdout and dispatches to channels.
func (m *Manager) readLoop(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			// Skip non-JSON startup messages like "spank: listening for slaps..."
			continue
		}

		// Try to unmarshal as a slap event first (has "slapNumber" field).
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		if _, ok := raw["slapNumber"]; ok {
			var ev SlapEvent
			if err := json.Unmarshal([]byte(line), &ev); err == nil {
				select {
				case m.SlapEvents <- ev:
				default:
				}
			}
			continue
		}

		// Otherwise it's a status/error response.
		var sr StatusResponse
		if err := json.Unmarshal([]byte(line), &sr); err == nil {
			select {
			case m.StatusEvents <- sr:
			default:
			}
		}
	}
}

// waitLoop waits for the subprocess to exit and marks it as stopped.
func (m *Manager) waitLoop() {
	if m.cmd != nil {
		_ = m.cmd.Wait()
	}
	m.mu.Lock()
	m.running = false
	m.cmd = nil
	m.stdin = nil
	m.mu.Unlock()
}
