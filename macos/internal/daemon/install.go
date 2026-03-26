package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const sudoersFile = "/etc/sudoers.d/com.taigrr.spankd"

// SpankdPath returns the absolute path to the spankd binary,
// located alongside the running SpankUI executable.
func SpankdPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}
	// Follow symlinks (important when launched via LaunchAgent)
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("evaluating symlinks: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), "spankd"), nil
}

// IsSudoersInstalled reports whether the sudoers entry exists.
func IsSudoersInstalled() bool {
	_, err := os.Stat(sudoersFile)
	return err == nil
}

// InstallSudoers installs a passwordless sudo entry for spankd.
// It writes the sudoers fragment to a temp file using Go (safe, no quoting
// issues), then uses a single osascript elevation to validate and install it.
func InstallSudoers() error {
	spankd, err := SpankdPath()
	if err != nil {
		return err
	}

	content := fmt.Sprintf("ALL ALL=(root) NOPASSWD: %s\n", spankd)
	tmpFile := "/tmp/com.taigrr.spankd.sudoers"

	// Write using Go — avoids all shell quoting/newline problems.
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing temp sudoers: %w", err)
	}

	// Single privileged operation: validate syntax, install, fix ownership.
	shellScript := strings.Join([]string{
		"visudo -cf " + tmpFile,
		"mv " + tmpFile + " " + sudoersFile,
		"chown root:wheel " + sudoersFile,
		"chmod 440 " + sudoersFile,
	}, " && ")

	appleScript := fmt.Sprintf(
		`do shell script "%s" with administrator privileges`,
		strings.ReplaceAll(shellScript, `"`, `\"`),
	)

	out, err := exec.Command("osascript", "-e", appleScript).CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("installing sudoers (osascript): %w\noutput: %s", err, out)
	}
	return nil
}

// SudoersMatchesPath reports whether the installed sudoers entry
// references the current spankd binary path. Returns false if the
// entry is stale (e.g. app was moved) and needs reinstalling.
func SudoersMatchesPath() bool {
	spankd, err := SpankdPath()
	if err != nil {
		return false
	}
	// We can't read the file (root-only), so test directly with sudo -n.
	err = exec.Command("sudo", "-n", spankd, "--version").Run()
	return err == nil
}

// UninstallSudoers removes the sudoers entry via an elevated shell.
func UninstallSudoers() error {
	appleScript := fmt.Sprintf(
		`do shell script "rm -f %s" with administrator privileges`,
		sudoersFile,
	)
	out, err := exec.Command("osascript", "-e", appleScript).CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing sudoers: %w\noutput: %s", err, out)
	}
	return nil
}
