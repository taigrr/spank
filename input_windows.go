//go:build windows

package main

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	setWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	getMessage          = user32.NewProc("GetMessageW")
)

const (
	WH_MOUSE_LL    = 14
	WM_LBUTTONDOWN = 0x0201
)

// HookProc is the signature of the callback function
type HookProc func(int, uintptr, uintptr) uintptr

// startMouseTrigger installs a global low-level mouse hook to trigger slaps.
func startMouseTrigger(ctx context.Context, trigger func()) {
	var hHook uintptr

	// Define the callback
	// Note: We use a variable to keep the callback reference alive
	var callback HookProc
	callback = func(code int, wParam uintptr, lParam uintptr) uintptr {
		if code >= 0 && wParam == WM_LBUTTONDOWN {
			trigger()
		}
		ret, _, _ := callNextHookEx.Call(hHook, uintptr(code), wParam, lParam)
		return ret
	}

	// Register the hook
	hHook, _, _ = setWindowsHookEx.Call(
		WH_MOUSE_LL,
		syscall.NewCallback(callback),
		0,
		0,
	)

	if hHook == 0 {
		fmt.Println("spank: failed to register mouse hook")
		return
	}
	defer unhookWindowsHookEx.Call(hHook)

	fmt.Println("spank: Mouse/Touchpad trigger active. Click or tap to slap!")

	// Simple message loop for the hook
	type MSG struct {
		Hwnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
	}

	msg := MSG{}

	// Run message loop in a loop that checks for context cancellation
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// GetMessage is blocking, but we only need it to process the hook
			// In a console app, this might block forever if no messages come.
			// However, WH_MOUSE_LL will wake it up on mouse events.
			ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(ret) <= 0 {
				return
			}
		}
	}
}
