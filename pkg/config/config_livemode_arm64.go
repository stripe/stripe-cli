//go:build arm64
// +build arm64

package config

import exec "golang.org/x/sys/execabs"

func deleteLivemodeKey(key string, profile string) error {
	fieldID := profile + "." + key
	_, err := exec.Command(
		execPathKeychain,
		"delete-generic-password",
		"-s", fieldID,
		"-a", KeyManagementService).CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
