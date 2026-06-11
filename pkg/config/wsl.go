//go:build !darwin

package config

import (
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
