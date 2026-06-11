//go:build !darwin

package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func isWSLFromVersion(procVersion string) bool {
	lower := strings.ToLower(procVersion)
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return isWSLFromVersion(string(data))
}

func wslFilePasswordFromPaths(machineIDPath, bootIDPath string) (string, error) {
	machineID, err := os.ReadFile(machineIDPath)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", machineIDPath, err)
	}
	bootID, err := os.ReadFile(bootIDPath)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", bootIDPath, err)
	}
	const appKey = "stripe-cli-keyring-v1"
	mac := hmac.New(sha256.New, []byte(appKey))
	mac.Write([]byte(strings.TrimSpace(string(machineID))))
	mac.Write([]byte(strings.TrimSpace(string(bootID))))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func wslFilePassword(_ string) (string, error) {
	return wslFilePasswordFromPaths("/etc/machine-id", "/proc/sys/kernel/random/boot_id")
}
