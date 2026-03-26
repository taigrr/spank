package tray

import (
	"github.com/getlantern/systray"
	"github.com/taigrr/spank/macos/internal/config"
	"github.com/taigrr/spank/macos/internal/daemon"
)

// Run starts the system tray application. Blocks until Quit is called.
func Run() {
	systray.Run(onReady, onExit)
}

func onReady() {
	cfg, _ := config.Load()
	mgr := daemon.New()

	Build(mgr, cfg)

	// Auto-start if the daemon was running when we last quit.
	// (For now we don't persist running state, so just update status.)
	setStatus(false)

	go EventLoop()
}

func onExit() {
	// Nothing needed — EventLoop handles cleanup on Quit click.
}
