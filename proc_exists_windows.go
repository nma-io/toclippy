//go:build windows

package main

import "os"

func processExists(proc *os.Process) error {
	// On Windows, FindProcess always succeeds; check via OpenProcess
	_, err := os.FindProcess(proc.Pid)
	return err
}
