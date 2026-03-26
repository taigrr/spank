package tray

import (
	"fmt"
	"os"

	"github.com/getlantern/systray"
	"github.com/taigrr/spank/macos/internal/config"
	"github.com/taigrr/spank/macos/internal/daemon"
	"github.com/taigrr/spank/macos/internal/launcher"
)

var (
	mgr *daemon.Manager
	cfg *config.Config

	mStatus      *systray.MenuItem
	mToggle      *systray.MenuItem
	mModePain    *systray.MenuItem
	mModeSexy    *systray.MenuItem
	mModeHalo    *systray.MenuItem
	mSensLow     *systray.MenuItem
	mSensMed     *systray.MenuItem
	mSensHigh    *systray.MenuItem
	mSpeedHalf   *systray.MenuItem
	mSpeedNorm   *systray.MenuItem
	mSpeedFast   *systray.MenuItem
	mVolScale    *systray.MenuItem
	mLoginItem   *systray.MenuItem
	mQuit        *systray.MenuItem
)

// Build creates all menu items.
func Build(m *daemon.Manager, c *config.Config) {
	mgr = m
	cfg = c

	systray.SetTemplateIcon(iconTemplate, iconRegular)
	systray.SetTooltip("Spank — slap your Mac, it yells back")

	mStatus = systray.AddMenuItem("Status: Stopped", "")
	mStatus.Disable()

	systray.AddSeparator()

	mToggle = systray.AddMenuItem("▶  Start", "Start or stop detection")

	systray.AddSeparator()

	// Mode submenu
	modeMenu := systray.AddMenuItem("Mode", "Sound mode")
	mModePain = modeMenu.AddSubMenuItem("Pain (default)", "Ow sounds")
	mModeSexy = modeMenu.AddSubMenuItem("Sexy", "Escalating intensity")
	mModeHalo = modeMenu.AddSubMenuItem("Halo", "Video game sounds")

	// Sensitivity submenu
	sensMenu := systray.AddMenuItem("Sensitivity", "Detection threshold")
	mSensLow = sensMenu.AddSubMenuItem("Low  (0.10)", "Less sensitive")
	mSensMed = sensMenu.AddSubMenuItem("Medium  (0.05)", "Default")
	mSensHigh = sensMenu.AddSubMenuItem("High  (0.02)", "More sensitive")

	// Speed submenu
	speedMenu := systray.AddMenuItem("Playback Speed", "Audio playback speed")
	mSpeedHalf = speedMenu.AddSubMenuItem("0.5×  Slow", "Half speed")
	mSpeedNorm = speedMenu.AddSubMenuItem("1.0×  Normal", "Normal speed")
	mSpeedFast = speedMenu.AddSubMenuItem("1.5×  Fast", "Fast speed")

	// Volume scaling toggle
	mVolScale = systray.AddMenuItem("Volume Scaling: Off", "Scale volume by slap force")

	systray.AddSeparator()

	mLoginItem = systray.AddMenuItem("Launch at Login", "Start Spank when you log in")

	systray.AddSeparator()

	mQuit = systray.AddMenuItem("Quit Spank", "Exit the application")

	// Apply initial checkbox states from loaded config
	refreshChecks()
}

// refreshChecks updates all checkmark states to match cfg.
func refreshChecks() {
	// Mode
	uncheckAll(mModePain, mModeSexy, mModeHalo)
	switch cfg.Mode {
	case "sexy":
		mModeSexy.Check()
	case "halo":
		mModeHalo.Check()
	default:
		mModePain.Check()
	}

	// Sensitivity
	uncheckAll(mSensLow, mSensMed, mSensHigh)
	switch {
	case cfg.MinAmplitude <= 0.03:
		mSensHigh.Check()
	case cfg.MinAmplitude <= 0.07:
		mSensMed.Check()
	default:
		mSensLow.Check()
	}

	// Speed
	uncheckAll(mSpeedHalf, mSpeedNorm, mSpeedFast)
	switch {
	case cfg.Speed < 0.75:
		mSpeedHalf.Check()
	case cfg.Speed < 1.25:
		mSpeedNorm.Check()
	default:
		mSpeedFast.Check()
	}

	// Volume scaling
	if cfg.VolumeScaling {
		mVolScale.SetTitle("Volume Scaling: On")
		mVolScale.Check()
	} else {
		mVolScale.SetTitle("Volume Scaling: Off")
		mVolScale.Uncheck()
	}

	// Login item
	if launcher.IsEnabled() {
		mLoginItem.Check()
		cfg.LaunchAtLogin = true
	} else {
		mLoginItem.Uncheck()
		cfg.LaunchAtLogin = false
	}
}

func uncheckAll(items ...*systray.MenuItem) {
	for _, item := range items {
		item.Uncheck()
	}
}

// setStatus updates the status label.
func setStatus(running bool) {
	if running {
		mStatus.SetTitle("Status: Running")
		mToggle.SetTitle("■  Stop")
	} else {
		mStatus.SetTitle("Status: Stopped")
		mToggle.SetTitle("▶  Start")
	}
}

// EventLoop blocks and handles all menu item clicks.
// Call this in a goroutine.
func EventLoop() {
	// Watch for slap events and daemon errors to keep status accurate
	go func() {
		for {
			select {
			case <-mgr.SlapEvents:
				// Could update slap counter here in future
			case err := <-mgr.Errors:
				showError("Daemon stopped", err.Error())
				setStatus(false)
			}
		}
	}()

	for {
		select {
		// Start/Stop toggle
		case <-mToggle.ClickedCh:
			if mgr.IsRunning() {
				stopDaemon()
			} else {
				startDaemon()
			}

		// Mode changes
		case <-mModePain.ClickedCh:
			changeMode("pain")
		case <-mModeSexy.ClickedCh:
			changeMode("sexy")
		case <-mModeHalo.ClickedCh:
			changeMode("halo")

		// Sensitivity
		case <-mSensLow.ClickedCh:
			cfg.MinAmplitude = 0.10
			updateSettings()
		case <-mSensMed.ClickedCh:
			cfg.MinAmplitude = 0.05
			updateSettings()
		case <-mSensHigh.ClickedCh:
			cfg.MinAmplitude = 0.02
			updateSettings()

		// Speed
		case <-mSpeedHalf.ClickedCh:
			cfg.Speed = 0.5
			updateSettings()
		case <-mSpeedNorm.ClickedCh:
			cfg.Speed = 1.0
			updateSettings()
		case <-mSpeedFast.ClickedCh:
			cfg.Speed = 1.5
			updateSettings()

		// Volume scaling
		case <-mVolScale.ClickedCh:
			cfg.VolumeScaling = !cfg.VolumeScaling
			if mgr.IsRunning() {
				_ = mgr.UpdateSettings(cfg)
			}
			_ = cfg.Save()
			refreshChecks()

		// Login item
		case <-mLoginItem.ClickedCh:
			toggleLoginItem()

		// Quit
		case <-mQuit.ClickedCh:
			_ = mgr.Stop()
			_ = cfg.Save()
			systray.Quit()
			return
		}
	}
}

func startDaemon() {
	// Ensure sudoers is installed first
	if !daemon.IsSudoersInstalled() || !daemon.SudoersMatchesPath() {
		if err := daemon.InstallSudoers(); err != nil {
			showError("Setup failed", err.Error())
			return
		}
		cfg.SudoersInstalled = true
		_ = cfg.Save()
	}

	if err := mgr.Start(cfg); err != nil {
		showError("Failed to start", err.Error())
		return
	}
	setStatus(true)
}

func stopDaemon() {
	_ = mgr.Stop()
	setStatus(false)
}

func changeMode(mode string) {
	wasRunning := mgr.IsRunning()
	if wasRunning {
		_ = mgr.Stop()
	}
	cfg.Mode = mode
	_ = cfg.Save()
	refreshChecks()
	if wasRunning {
		startDaemon()
	}
}

func updateSettings() {
	_ = cfg.Save()
	if mgr.IsRunning() {
		// Send live update (amplitude, cooldown, speed can be changed without restart)
		_ = mgr.UpdateSettings(cfg)
	}
	refreshChecks()
}

func toggleLoginItem() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if launcher.IsEnabled() {
		_ = launcher.Disable()
		cfg.LaunchAtLogin = false
	} else {
		_ = launcher.Enable(exe)
		cfg.LaunchAtLogin = true
	}
	_ = cfg.Save()
	refreshChecks()
}

func showError(title, msg string) {
	// Update status bar to show error briefly
	mStatus.SetTitle(fmt.Sprintf("Error: %s", title))
	// Log to stderr as well
	_, _ = fmt.Fprintf(os.Stderr, "spank: %s: %s\n", title, msg)
}
