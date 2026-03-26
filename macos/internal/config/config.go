package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds persisted user preferences.
type Config struct {
	Mode             string  `json:"mode"`             // "pain", "sexy", "halo"
	MinAmplitude     float64 `json:"minAmplitude"`     // detection threshold
	Cooldown         int     `json:"cooldown"`         // ms between responses
	Speed            float64 `json:"speed"`            // playback speed multiplier
	VolumeScaling    bool    `json:"volumeScaling"`    // scale volume by amplitude
	LaunchAtLogin    bool    `json:"launchAtLogin"`
	SudoersInstalled bool    `json:"sudoersInstalled"` // one-time setup done
}

func Default() *Config {
	return &Config{
		Mode:         "pain",
		MinAmplitude: 0.05,
		Cooldown:     750,
		Speed:        1.0,
	}
}

func prefPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Preferences", "com.taigrr.spank.json"), nil
}

func Load() (*Config, error) {
	path, err := prefPath()
	if err != nil {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Default(), nil
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return Default(), nil
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := prefPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
