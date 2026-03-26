package daemon

// Command is sent to spankd via stdin (JSON, one per line).
type Command struct {
	Cmd       string  `json:"cmd"`
	Amplitude float64 `json:"amplitude,omitempty"`
	Cooldown  int     `json:"cooldown,omitempty"`
	Speed     float64 `json:"speed,omitempty"`
}

// StatusResponse is received from spankd stdout for status/control replies.
type StatusResponse struct {
	Status        string  `json:"status"`
	Paused        bool    `json:"paused"`
	Amplitude     float64 `json:"amplitude"`
	Cooldown      int     `json:"cooldown"`
	VolumeScaling bool    `json:"volume_scaling"`
	Speed         float64 `json:"speed"`
	Error         string  `json:"error,omitempty"`
}

// SlapEvent is emitted by spankd on each detected slap.
type SlapEvent struct {
	Timestamp  string  `json:"timestamp"`
	SlapNumber int     `json:"slapNumber"`
	Amplitude  float64 `json:"amplitude"`
	Severity   string  `json:"severity"`
	File       string  `json:"file"`
}
