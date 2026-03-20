package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

func showHistory() error {
	running, pid := checkDaemonRunning()
	if !running {
		fmt.Fprintln(os.Stderr, "Note: monitor is not running (new items not captured).")
		fmt.Fprintln(os.Stderr, "Start it with:  toclippy --monitor --daemon")
		fmt.Fprintln(os.Stderr, "")
	}
	_ = pid

	entries, err := loadHistory()
	if err != nil {
		return err
	}
	defer func() {
		for i := range entries {
			entries[i].zero()
		}
	}()

	if len(entries) == 0 {
		fmt.Println("No clipboard history.")
		return nil
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, oldState)

	selected := len(entries) - 1 // most recent first in display
	for {
		renderMenu(entries, selected)
		key, err := readKey(os.Stdin)
		if err != nil {
			break
		}
		switch key {
		case "up":
			if selected > 0 {
				selected--
			}
		case "down":
			if selected < len(entries)-1 {
				selected++
			}
		case "1", "2", "3", "4", "5":
			n := int(key[0]-'1')
			if n < len(entries) {
				selected = n
			}
		case "\r", "\n":
			term.Restore(fd, oldState)
			clearScreen()
			if err := clipboardWrite(entries[selected].Content); err != nil {
				return err
			}
			fmt.Println("Clipboard restored.")
			return nil
		case "c", "C":
			term.Restore(fd, oldState)
			clearScreen()
			fmt.Print("Clear all clipboard history? [y/N]: ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := scanner.Text()
			if strings.EqualFold(strings.TrimSpace(answer), "y") {
				if err := clearHistorySecure(); err != nil {
					return err
				}
				fmt.Println("Clipboard history cleared.")
				return nil
			}
			oldState, _ = term.MakeRaw(fd)
		case "q", "Q", "\x1b":
			return nil
		}
	}
	return nil
}

func clearAllHistory() error {
	fmt.Print("Clear all clipboard history? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	answer := scanner.Text()
	if !strings.EqualFold(strings.TrimSpace(answer), "y") {
		fmt.Println("Aborted.")
		return nil
	}
	if err := clearHistorySecure(); err != nil {
		return err
	}
	fmt.Println("Clipboard history cleared.")
	return nil
}

func renderMenu(entries []historyEntry, selected int) {
	clearScreen()
	fmt.Print("Clipboard History   [arrows/numbers=select   Enter=restore   C=clear   Q=quit]\r\n\r\n")
	for i, e := range entries {
		ts := time.Unix(e.Timestamp, 0).Format("2006-01-02 15:04:05")
		preview := entryPreview(e)
		line := fmt.Sprintf("[%d] %s  %-60s", i+1, ts, preview)
		if i == selected {
			fmt.Printf(" \033[7m > %s\033[0m\r\n", line)
		} else {
			fmt.Printf("   %s\r\n", line)
		}
	}
	fmt.Print("\r\n")
}

func entryPreview(e historyEntry) string {
	if e.Kind == kindBinary {
		return "[BINARY DATA]"
	}
	s := string(e.Content)
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > 60 {
		return string(runes[:57]) + "..."
	}
	if utf8.RuneCountInString(s) > 60 {
		return string([]rune(s)[:57]) + "..."
	}
	return s
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func readKey(r io.Reader) (string, error) {
	buf := make([]byte, 1)
	if _, err := r.Read(buf); err != nil {
		return "", err
	}
	if buf[0] != 0x1b {
		return string(buf[:1]), nil
	}
	seq := make([]byte, 3)
	n, _ := r.Read(seq)
	if n == 0 {
		return "\x1b", nil
	}
	if n >= 2 && seq[0] == '[' {
		switch seq[1] {
		case 'A':
			return "up", nil
		case 'B':
			return "down", nil
		case 'C':
			return "right", nil
		case 'D':
			return "left", nil
		}
	}
	return "\x1b" + string(seq[:n]), nil
}
