package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// resetGlobals resets global state before each test
func resetGlobals() {
	pausedMu.Lock()
	paused = false
	pausedMu.Unlock()
	minAmplitude = 0.05
	cooldownMs = 750
	stdioMode = true
}

func TestPauseCommand(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"pause"}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	// Check state changed
	pausedMu.RLock()
	if !paused {
		t.Error("expected paused to be true after pause command")
	}
	pausedMu.RUnlock()

	// Check output
	var resp map[string]string
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "paused" {
		t.Errorf("expected status 'paused', got %q", resp["status"])
	}
}

func TestResumeCommand(t *testing.T) {
	resetGlobals()

	// First pause
	pausedMu.Lock()
	paused = true
	pausedMu.Unlock()

	input := `{"cmd":"resume"}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	// Check state changed
	pausedMu.RLock()
	if paused {
		t.Error("expected paused to be false after resume command")
	}
	pausedMu.RUnlock()

	// Check output
	var resp map[string]string
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "resumed" {
		t.Errorf("expected status 'resumed', got %q", resp["status"])
	}
}

func TestSetAmplitudeCommand(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"set","amplitude":0.15}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	// Check state changed
	if minAmplitude != 0.15 {
		t.Errorf("expected minAmplitude 0.15, got %f", minAmplitude)
	}

	// Check output
	var resp struct {
		Status    string  `json:"status"`
		Amplitude float64 `json:"amplitude"`
		Cooldown  int     `json:"cooldown"`
	}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != "settings_updated" {
		t.Errorf("expected status 'settings_updated', got %q", resp.Status)
	}
	if resp.Amplitude != 0.15 {
		t.Errorf("expected amplitude 0.15 in response, got %f", resp.Amplitude)
	}
}

func TestSetCooldownCommand(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"set","cooldown":500}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	// Check state changed
	if cooldownMs != 500 {
		t.Errorf("expected cooldownMs 500, got %d", cooldownMs)
	}

	// Check output
	var resp struct {
		Status   string `json:"status"`
		Cooldown int    `json:"cooldown"`
	}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Cooldown != 500 {
		t.Errorf("expected cooldown 500 in response, got %d", resp.Cooldown)
	}
}

func TestSetBothCommand(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"set","amplitude":0.2,"cooldown":1000}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	if minAmplitude != 0.2 {
		t.Errorf("expected minAmplitude 0.2, got %f", minAmplitude)
	}
	if cooldownMs != 1000 {
		t.Errorf("expected cooldownMs 1000, got %d", cooldownMs)
	}
}

func TestSetAmplitudeOutOfRange(t *testing.T) {
	resetGlobals()
	originalAmplitude := minAmplitude

	// Test amplitude > 1 (should be ignored)
	input := `{"cmd":"set","amplitude":1.5}` + "\n"
	var output bytes.Buffer
	processCommands(strings.NewReader(input), &output)

	if minAmplitude != originalAmplitude {
		t.Errorf("amplitude should not change for value > 1, got %f", minAmplitude)
	}

	// Test amplitude <= 0 (should be ignored)
	resetGlobals()
	input = `{"cmd":"set","amplitude":0}` + "\n"
	output.Reset()
	processCommands(strings.NewReader(input), &output)

	if minAmplitude != originalAmplitude {
		t.Errorf("amplitude should not change for value <= 0, got %f", minAmplitude)
	}

	// Test negative amplitude
	resetGlobals()
	input = `{"cmd":"set","amplitude":-0.5}` + "\n"
	output.Reset()
	processCommands(strings.NewReader(input), &output)

	if minAmplitude != originalAmplitude {
		t.Errorf("amplitude should not change for negative value, got %f", minAmplitude)
	}
}

func TestStatusCommand(t *testing.T) {
	resetGlobals()
	minAmplitude = 0.1
	cooldownMs = 600

	input := `{"cmd":"status"}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	var resp struct {
		Status    string  `json:"status"`
		Paused    bool    `json:"paused"`
		Amplitude float64 `json:"amplitude"`
		Cooldown  int     `json:"cooldown"`
	}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Paused != false {
		t.Errorf("expected paused false, got %t", resp.Paused)
	}
	if resp.Amplitude != 0.1 {
		t.Errorf("expected amplitude 0.1, got %f", resp.Amplitude)
	}
	if resp.Cooldown != 600 {
		t.Errorf("expected cooldown 600, got %d", resp.Cooldown)
	}
}

func TestStatusCommandWhenPaused(t *testing.T) {
	resetGlobals()
	pausedMu.Lock()
	paused = true
	pausedMu.Unlock()

	input := `{"cmd":"status"}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	var resp struct {
		Paused bool `json:"paused"`
	}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Paused != true {
		t.Errorf("expected paused true, got %t", resp.Paused)
	}
}

func TestUnknownCommand(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"invalid"}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	var resp map[string]string
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Error("expected error field in response for unknown command")
	}
	if !strings.Contains(resp["error"], "unknown command") {
		t.Errorf("expected 'unknown command' error, got %q", resp["error"])
	}
}

func TestInvalidJSON(t *testing.T) {
	resetGlobals()

	input := `{not valid json}` + "\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	var resp map[string]string
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Error("expected error field in response for invalid JSON")
	}
	if !strings.Contains(resp["error"], "invalid command") {
		t.Errorf("expected 'invalid command' error, got %q", resp["error"])
	}
}

func TestEmptyLines(t *testing.T) {
	resetGlobals()

	// Empty lines should be skipped, only the status command should produce output
	input := "\n\n" + `{"cmd":"status"}` + "\n\n"
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 output line, got %d: %v", len(lines), lines)
	}
}

func TestMultipleCommands(t *testing.T) {
	resetGlobals()

	input := `{"cmd":"pause"}
{"cmd":"status"}
{"cmd":"resume"}
{"cmd":"status"}
`
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 output lines, got %d", len(lines))
	}

	// First: paused
	var resp1 map[string]string
	json.Unmarshal([]byte(lines[0]), &resp1)
	if resp1["status"] != "paused" {
		t.Errorf("line 1: expected 'paused', got %q", resp1["status"])
	}

	// Second: status shows paused=true
	var resp2 struct {
		Paused bool `json:"paused"`
	}
	json.Unmarshal([]byte(lines[1]), &resp2)
	if !resp2.Paused {
		t.Error("line 2: expected paused=true")
	}

	// Third: resumed
	var resp3 map[string]string
	json.Unmarshal([]byte(lines[2]), &resp3)
	if resp3["status"] != "resumed" {
		t.Errorf("line 3: expected 'resumed', got %q", resp3["status"])
	}

	// Fourth: status shows paused=false
	var resp4 struct {
		Paused bool `json:"paused"`
	}
	json.Unmarshal([]byte(lines[3]), &resp4)
	if resp4.Paused {
		t.Error("line 4: expected paused=false")
	}
}

func TestNoOutputWhenStdioModeDisabled(t *testing.T) {
	resetGlobals()
	stdioMode = false

	input := `{"cmd":"pause"}
{"cmd":"status"}
{"cmd":"set","amplitude":0.5}
`
	var output bytes.Buffer

	processCommands(strings.NewReader(input), &output)

	// No output should be produced when stdioMode is false
	if output.Len() != 0 {
		t.Errorf("expected no output when stdioMode=false, got %q", output.String())
	}

	// But state should still change
	pausedMu.RLock()
	if !paused {
		t.Error("expected paused to be true even with stdioMode=false")
	}
	pausedMu.RUnlock()

	if minAmplitude != 0.5 {
		t.Errorf("expected minAmplitude 0.5, got %f", minAmplitude)
	}
}
