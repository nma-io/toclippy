package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"unicode/utf8"

	"golang.design/x/clipboard"
)

const clipHoldEnv = "TOCLIPPY_CLIP_HOLDER"

func init() {
	if os.Getenv(clipHoldEnv) == "1" {
		return
	}
	clipboard.Init()
}

type entryKind byte

const (
	kindText   entryKind = 0
	kindBinary entryKind = 1
)

func clipboardWrite(data []byte) error {
	if runtime.GOOS == "linux" {
		return clipboardWriteLinux(data)
	}
	done := clipboard.Write(clipboard.FmtText, data)
	if done == nil {
		return fmt.Errorf("clipboard write failed")
	}
	return nil
}

func clipboardWriteLinux(data []byte) error {
	if err := clipboard.Init(); err != nil {
		// No DISPLAY - try /dev/clipboard (iSH on iPad)??
		return writeDevClipboard(data)
	}

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), clipHoldEnv+"=1")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		// Fallback: write directly and block briefly
		done := clipboard.Write(clipboard.FmtText, data)
		if done == nil {
			return fmt.Errorf("clipboard write failed")
		}
		return nil
	}
	detachProc(cmd)
	if err := cmd.Start(); err != nil {
		stdin.Close()
		// Fallback: direct write
		done := clipboard.Write(clipboard.FmtText, data)
		if done == nil {
			return fmt.Errorf("clipboard write failed")
		}
		return nil
	}
	_, err = stdin.Write(data)
	stdin.Close()
	return err
}

// runClipHolder is invoked when we are the background clipboard holder process on Linux.
func runClipHolder() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		os.Exit(1)
	}
	if err := clipboard.Init(); err != nil {
		zeroBytes(data)
		os.Exit(1)
	}
	done := clipboard.Write(clipboard.FmtText, data)
	if done == nil {
		zeroBytes(data)
		os.Exit(1)
	}
	<-done
	zeroBytes(data)
	os.Exit(0)
}

func clipboardRead() ([]byte, error) {
	if err := clipboard.Init(); err != nil {
		return readDevClipboard()
	}
	data := clipboard.Read(clipboard.FmtText)
	if len(data) > 0 {
		return data, nil
	}
	return nil, nil
}

func clipboardReadForMonitor() ([]byte, entryKind, error) {
	if err := clipboard.Init(); err != nil {
		data, err := readDevClipboard()
		if err != nil {
			return nil, kindText, err
		}
		return classifyContent(data)
	}

	text := clipboard.Read(clipboard.FmtText)
	if len(text) > 0 {
		return classifyContent(text)
	}

	img := clipboard.Read(clipboard.FmtImage)
	if len(img) > 0 {
		return img, kindBinary, nil
	}

	return nil, kindText, nil
}

func classifyContent(data []byte) ([]byte, entryKind, error) {
	if !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return data, kindBinary, nil
	}
	return data, kindText, nil
}

func clipboardToFile(path string) error {
	data, err := clipboardRead()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("clipboard is empty")
	}
	if path == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func readDevClipboard() ([]byte, error) {
	f, err := os.Open("/dev/clipboard")
	if err != nil {
		return nil, fmt.Errorf("clipboard unavailable (no display and no /dev/clipboard)")
	}
	defer f.Close()
	buf := make([]byte, 65536)
	n, _ := f.Read(buf)
	return buf[:n], nil
}

func writeDevClipboard(data []byte) error {
	f, err := os.OpenFile("/dev/clipboard", os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("clipboard unavailable (no display and no /dev/clipboard)")
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
