//go:build !darwin && !windows

package main

func isForegroundPasswordManager() bool {
	return false
}
