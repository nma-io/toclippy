//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func detachProc(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | 0x08000000, // DETACHED_PROCESS | CREATE_NO_WINDOW
	}
}
