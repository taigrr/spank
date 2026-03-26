package main

import "github.com/taigrr/spank/macos/internal/tray"

func main() {
	// systray.Run must be called on the main goroutine.
	tray.Run()
}
