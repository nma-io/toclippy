package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	historyMagic    = "TOCLIPPY"
	historyVersion  = byte(0x01)
	maxHistoryItems = 5
	keySize         = 32
	nonceSize       = 12
)

type historyEntry struct {
	Timestamp int64
	Kind      entryKind
	Content   []byte
}

func (e *historyEntry) zero() {
	zeroBytes(e.Content)
	e.Content = nil
}

func configDir() (string, error) {
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return "", fmt.Errorf("APPDATA not set")
		}
		return filepath.Join(appdata, "toclippy"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "toclippy"), nil
}

func ensureConfigDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func loadKey() ([]byte, error) {
	dir, err := ensureConfigDir()
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(dir, ".key")
	data, err := os.ReadFile(keyPath)
	if err == nil && len(data) == keySize {
		return data, nil
	}
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("key write failed: %w", err)
	}
	return key, nil
}

func historyPath() (string, error) {
	dir, err := ensureConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history"), nil
}

func pidPath() (string, error) {
	dir, err := ensureConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "toclippy.pid"), nil
}

func encryptEntry(key, plaintext []byte, ts int64, kind entryKind) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// Additional data: timestamp + kind
	ad := make([]byte, 9)
	binary.LittleEndian.PutUint64(ad[:8], uint64(ts))
	ad[8] = byte(kind)

	ciphertext := gcm.Seal(nonce, nonce, plaintext, ad)
	return ciphertext, nil
}

func decryptEntry(key, ciphertext []byte, ts int64, kind entryKind) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := ciphertext[:nonceSize]
	ad := make([]byte, 9)
	binary.LittleEndian.PutUint64(ad[:8], uint64(ts))
	ad[8] = byte(kind)

	return gcm.Open(nil, nonce, ciphertext[nonceSize:], ad)
}

// History file binary format:
//   [8 bytes: magic "TOCLIPPY"][1 byte: version][1 byte: entry count]
//   For each entry:
//     [4 bytes LE: entry data length][8 bytes LE: timestamp]
//     [1 byte: kind][N bytes: encrypted data (nonce+ciphertext)]

func loadHistory() ([]historyEntry, error) {
	key, err := loadKey()
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)

	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	magic := make([]byte, 8)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, nil
	}
	if string(magic) != historyMagic {
		return nil, nil
	}

	var ver, count byte
	if err := binary.Read(f, binary.LittleEndian, &ver); err != nil {
		return nil, nil
	}
	if ver != historyVersion {
		return nil, nil
	}
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, nil
	}

	entries := make([]historyEntry, 0, count)
	for i := 0; i < int(count); i++ {
		var dataLen uint32
		if err := binary.Read(f, binary.LittleEndian, &dataLen); err != nil {
			break
		}
		var ts int64
		if err := binary.Read(f, binary.LittleEndian, &ts); err != nil {
			break
		}
		var kind byte
		if err := binary.Read(f, binary.LittleEndian, &kind); err != nil {
			break
		}
		enc := make([]byte, dataLen)
		if _, err := io.ReadFull(f, enc); err != nil {
			break
		}
		plain, err := decryptEntry(key, enc, ts, entryKind(kind))
		if err != nil {
			continue
		}
		entries = append(entries, historyEntry{
			Timestamp: ts,
			Kind:      entryKind(kind),
			Content:   plain,
		})
	}
	return entries, nil
}

func saveHistory(entries []historyEntry) error {
	key, err := loadKey()
	if err != nil {
		return err
	}
	defer zeroBytes(key)

	path, err := historyPath()
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	f.WriteString(historyMagic)
	f.Write([]byte{historyVersion, byte(len(entries))})

	for _, e := range entries {
		enc, err := encryptEntry(key, e.Content, e.Timestamp, e.Kind)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		binary.Write(f, binary.LittleEndian, uint32(len(enc)))
		binary.Write(f, binary.LittleEndian, e.Timestamp)
		f.Write([]byte{byte(e.Kind)})
		f.Write(enc)
		zeroBytes(enc)
	}
	f.Sync()
	f.Close()
	return os.Rename(tmp, path)
}

func appendToHistory(data []byte, kind entryKind, maxBuff int64) error {
	entries, err := loadHistory()
	if err != nil {
		entries = nil
	}
	defer func() {
		for i := range entries {
			entries[i].zero()
		}
	}()

	entry := historyEntry{
		Timestamp: time.Now().Unix(),
		Kind:      kind,
		Content:   make([]byte, len(data)),
	}
	copy(entry.Content, data)

	entries = append(entries, entry)

	// Enforce max items and maxBuff total size
	for len(entries) > maxHistoryItems {
		entries[0].zero()
		entries = entries[1:]
	}

	var totalBytes int64
	for i := len(entries) - 1; i >= 0; i-- {
		totalBytes += int64(len(entries[i].Content))
		if totalBytes > maxBuff {
			for j := 0; j < i; j++ {
				entries[j].zero()
			}
			entries = entries[i:]
			break
		}
	}

	return saveHistory(entries)
}

func clearHistorySecure() error {
	path, err := historyPath()
	if err != nil {
		return err
	}

	entries, _ := loadHistory()
	for i := range entries {
		entries[i].zero()
	}

	if err := secureDeleteFile(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func secureDeleteFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	size := info.Size()

	buf := make([]byte, 4096)
	for pass := 0; pass < 3; pass++ {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			break
		}
		var fill byte
		switch pass {
		case 0:
			fill = 0x00
		case 1:
			fill = 0xFF
		case 2:
			rand.Read(buf)
		}
		if pass != 2 {
			for i := range buf {
				buf[i] = fill
			}
		}
		for written := int64(0); written < size; {
			chunk := buf
			if int64(len(chunk)) > size-written {
				chunk = chunk[:size-written]
			}
			n, err := f.Write(chunk)
			if err != nil {
				break
			}
			written += int64(n)
		}
		f.Sync()
	}
	f.Close()
	return os.Remove(path)
}
