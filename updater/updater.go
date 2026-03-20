package updater

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	versionURL = "https://disog-files.s3.amazonaws.com/toclippy/toclippy.version"
	binaryBase = "https://disog-files.s3.amazonaws.com/toclippy/"
)

var osMap = map[string]string{
	"darwin":  "osx",
	"linux":   "elf",
	"windows": "windows",
}

var archMap = map[string]string{
	"amd64": "x86",
	"arm64": "arm",
	"386":   "ish",
}

func platformBinaryURL() string {
	osName := osMap[runtime.GOOS]
	archName := archMap[runtime.GOARCH]
	if osName == "" || archName == "" {
		return ""
	}
	return fmt.Sprintf("%stoclippy.%s.%s", binaryBase, osName, archName)
}

func Update(currentVersion string) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
		Timeout: 20 * time.Second,
	}

	fmt.Println("[!] Checking for updates...")

	resp, err := client.Get(versionURL)
	if err != nil {
		return fmt.Errorf("version check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("version check returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read version: %w", err)
	}

	latest := strings.TrimSpace(string(body))
	fmt.Printf("[+] Current: %s\n", currentVersion)
	fmt.Printf("[+] Latest:  %s\n", latest)

	curParts := strings.Split(currentVersion, ".")
	latParts := strings.Split(latest, ".")
	if len(curParts) != 3 || len(latParts) != 3 {
		return fmt.Errorf("unexpected version format: %s / %s", currentVersion, latest)
	}

	for i := 0; i < 3; i++ {
		cur, e1 := strconv.Atoi(curParts[i])
		lat, e2 := strconv.Atoi(latParts[i])
		if e1 != nil || e2 != nil {
			return fmt.Errorf("non-numeric version component")
		}
		if lat > cur {
			return downloadAndReplace(client, latest)
		} else if lat < cur {
			fmt.Println("[+] Running newer than available release.")
			return nil
		}
	}

	fmt.Println("[+] Already up to date.")
	return nil
}

func downloadAndReplace(client *http.Client, latest string) error {
	binaryURL := platformBinaryURL()
	if binaryURL == "" {
		return fmt.Errorf("no release available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("[!] Downloading update from: %s\n", binaryURL)

	resp, err := client.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlink: %w", err)
	}

	tmp := execPath + ".new"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}

	if err := os.Rename(tmp, execPath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	fmt.Printf("[+] Updated to %s. Restart toclippy to use the new version.\n", latest)
	return nil
}
