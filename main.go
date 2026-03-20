package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"unicode/utf16"
	"unicode/utf8"

	"toclippy/updater"
)

const (
	companyName     = "DISOG LLC"
	copyright       = "(c) Nicholas Albright (@nma-io)"
	fileDescription = "Send screen data to the clipboard"
	version         = "2026.1.0"
	internalName    = "toClippy"
	productName     = "toClippy"
)

func main() {
	// Hidden mode: clipboard holder for Linux X11 persistence
	if os.Getenv(clipHoldEnv) == "1" {
		runClipHolder()
		return
	}

	var (
		maxBuff  int64
		inFile   string
		outFile  string
		utf8Opt  bool
		fromCB   bool
		shortF   bool
		monitor  bool
		daemonF  bool
		restore  bool
		historyF bool
		clearAll  bool
		updateF   bool
	)

	flag.BoolVar(&updateF, "update", false, "check for and install updates")
	flag.Int64Var(&maxBuff, "maxbuff", 100*1024*1024, "max buffer size in bytes (default 100MB)")
	flag.StringVar(&inFile, "i", "", "input file")
	flag.StringVar(&outFile, "o", "", "output file (use with -fromcb)")
	flag.BoolVar(&utf8Opt, "utf8", false, "convert UTF-16 input to UTF-8")
	flag.BoolVar(&fromCB, "fromcb", false, "read clipboard and write to -o file")
	flag.BoolVar(&shortF, "f", false, "read clipboard and write to -o file")
	flag.BoolVar(&monitor, "monitor", false, "monitor clipboard every 5s")
	flag.BoolVar(&daemonF, "daemon", false, "run monitor in background (use with --monitor)")
	flag.BoolVar(&restore, "restore", false, "browse clipboard history and restore selection")
	flag.BoolVar(&historyF, "history", false, "browse clipboard history and restore selection")
	flag.BoolVar(&clearAll, "clear-all", false, "securely wipe all clipboard history")
	flag.Parse()

	if updateF {
		fmt.Printf("toclippy v%s\n", version)
		if err := updater.Update(version); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if fromCB || shortF {
		if err := clipboardToFile(outFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if monitor {
		if daemonF {
			if err := startDaemon(maxBuff); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		runMonitor(maxBuff)
		return
	}

	if restore || historyF {
		if err := showHistory(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if clearAll {
		if err := clearAllHistory(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	var data []byte
	var err error

	if inFile != "" {
		data, err = readFile(inFile, maxBuff)
	} else {
		data, err = readStdin(maxBuff)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(data) == 0 {
		fmt.Fprintln(os.Stderr, "no input: pipe data or use -i <file>")
		flag.Usage()
		os.Exit(1)
	}

	if utf8Opt {
		data, err = convertUTF16(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "utf-16 conversion:", err)
			os.Exit(1)
		}
	}

	if err := clipboardWrite(data); err != nil {
		fmt.Fprintln(os.Stderr, "clipboard write:", err)
		os.Exit(1)
	}
}

func readFile(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, max))
}

func readStdin(max int64) ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, nil
	}
	return io.ReadAll(io.LimitReader(os.Stdin, max))
}

func convertUTF16(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return data, nil
	}
	var u16 []uint16
	switch {
	case data[0] == 0xFF && data[1] == 0xFE:
		for i := 2; i+1 < len(data); i += 2 {
			u16 = append(u16, uint16(data[i])|uint16(data[i+1])<<8)
		}
	case data[0] == 0xFE && data[1] == 0xFF:
		for i := 2; i+1 < len(data); i += 2 {
			u16 = append(u16, uint16(data[i])<<8|uint16(data[i+1]))
		}
	default:
		return data, nil
	}
	runes := utf16.Decode(u16)
	out := make([]byte, 0, len(runes)*2)
	enc := make([]byte, utf8.UTFMax)
	for _, r := range runes {
		n := utf8.EncodeRune(enc, r)
		out = append(out, enc[:n]...)
	}
	return out, nil
}
