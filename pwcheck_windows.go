//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
)

func isForegroundPasswordManager() bool {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return false
	}
	buf := make([]uint16, 256)
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	title := syscall.UTF16ToString(buf)
	return isPasswordManagerName(title)
}
