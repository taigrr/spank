package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"bytes"
)

const agentLabel = "com.taigrr.spank-ui"

var plistTemplate = template.Must(template.New("agent").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.taigrr.spank-ui</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.ExecPath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardErrorPath</key>
    <string>/tmp/com.taigrr.spank-ui.log</string>
</dict>
</plist>
`))

func agentPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", agentLabel+".plist"), nil
}

// Enable installs a user LaunchAgent so SpankUI starts at login.
func Enable(execPath string) error {
	plistPath, err := agentPlistPath()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := plistTemplate.Execute(&buf, struct{ ExecPath string }{execPath}); err != nil {
		return fmt.Errorf("rendering plist template: %w", err)
	}

	dir := filepath.Dir(plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}

	if err := os.WriteFile(plistPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	return nil
}

// Disable removes the LaunchAgent, preventing SpankUI from starting at login.
func Disable() error {
	plistPath, err := agentPlistPath()
	if err != nil {
		return err
	}

	// Unload if loaded (ignore errors — may not be loaded)
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}
	return nil
}

// IsEnabled reports whether the LaunchAgent plist exists.
func IsEnabled() bool {
	path, err := agentPlistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
