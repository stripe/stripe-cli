//go:build arm64
// +build arm64

package config

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/stripe/stripe-cli/pkg/validators"
	exec "golang.org/x/sys/execabs"
)

const (
	// execPathKeychain is the path to the keychain binary
	execPathKeychain = "/usr/bin/security"

	// encodingPrefix is a well-known prefix added to strings encoded by Set.
	encodingPrefix = "go-keyring-encoded:"
)

// saveLivemodeValue saves livemode value of given key in keyring
func (p *Profile) saveLivemodeValue(key, value, description string) {
	fieldID := p.GetConfigField(key)
	value = encodingPrefix + hex.EncodeToString([]byte(value))

	cmd := exec.Command(execPathKeychain, "-i")
	stdIn, _ := cmd.StdinPipe()
	cmd.Start()

	command := fmt.Sprintf(
		"add-generic-password -U -s %s -a %s -w %s\n",
		shellescape.Quote(fieldID),
		shellescape.Quote(KeyManagementService),
		shellescape.Quote(value),
	)

	io.WriteString(stdIn, command)
	stdIn.Close()
	cmd.Wait()
}

// retrieveLivemodeValue retrieves livemode value of given key in keyring
func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
	fieldID := p.GetConfigField(key)

	out, err := exec.Command(
		execPathKeychain,
		"find-generic-password",
		"-s", fieldID,
		"-wa", KeyManagementService).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "could not be found") {
			return "", validators.ErrAPIKeyNotConfigured
		}
	}

	value := strings.TrimSpace(string(out[:]))
	// if the string has the well-known prefix, assume it's encoded
	if strings.HasPrefix(value, encodingPrefix) {
		dec, err := hex.DecodeString(value[len(encodingPrefix):])
		return string(dec), err
	}

	if value != "" {
		return value, nil
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// deleteLivemodeValue deletes livemode value of given key in keyring
func (p *Profile) deleteLivemodeValue(key string) error {
	fieldID := p.GetConfigField(key)
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
