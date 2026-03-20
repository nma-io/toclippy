//go:build !windows

package main

import (
	"os"
	"syscall"
)

func processExists(proc *os.Process) error {
	return proc.Signal(syscall.Signal(0))
}
