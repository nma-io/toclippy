package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func runMonitor(maxBuff int64) {
	if err := writePID(); err != nil {
		fmt.Fprintln(os.Stderr, "could not write PID file:", err)
	}
	defer removePID()

	fmt.Fprintln(os.Stderr, "toclippy monitor running (PID", os.Getpid(), ")")

	var last []byte

	for {
		time.Sleep(5 * time.Second)

		data, kind, err := clipboardReadForMonitor()
		if err != nil || len(data) == 0 {
			continue
		}

		if bytes.Equal(data, last) {
			continue
		}

		if isForegroundPasswordManager() {
			last = data
			continue
		}

		if err := appendToHistory(data, kind, maxBuff); err != nil {
			fmt.Fprintln(os.Stderr, "history write error:", err)
		}

		lastCopy := make([]byte, len(data))
		copy(lastCopy, data)
		zeroBytes(last)
		last = lastCopy
	}
}

func writePID() error {
	path, err := pidPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0600)
}

func removePID() {
	path, err := pidPath()
	if err != nil {
		return
	}
	os.Remove(path)
}

func checkDaemonRunning() (bool, int) {
	path, err := pidPath()
	if err != nil {
		return false, 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}
	if err := processExists(proc); err != nil {
		return false, 0
	}
	return true, pid
}

func startDaemon(maxBuff int64) error {
	args := []string{
		"--monitor",
		fmt.Sprintf("--maxbuff=%d", maxBuff),
	}
	cmd := exec.Command(os.Args[0], args...)
	detachProc(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	fmt.Printf("toclippy monitor started (PID %d)\n", cmd.Process.Pid)
	return nil
}
