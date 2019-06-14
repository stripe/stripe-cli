package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKeyInput(t *testing.T) {
	expectedKey := "sk_test_foo1234"

	cc := newConfigureCmd()
	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := cc.getConfigureAPIKey(keyInput)

	assert.Equal(t, expectedKey, actualKey)
	assert.Nil(t, err)
}

func TestAPIKeyInputEmpty(t *testing.T) {
	expectedKey := ""
	expectedErrorString := "API key is required, please provide your test mode secret API key"

	cc := newConfigureCmd()
	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := cc.getConfigureAPIKey(keyInput)

	assert.Equal(t, expectedKey, actualKey)
	assert.NotNil(t, err)
	assert.EqualError(t, err, expectedErrorString)
}

func TestAPIKeyInputLivemode(t *testing.T) {
	expectedKey := ""
	livemodeKey := "sk_live_foo123"
	expectedErrorString := "the CLI only supports using a test mode secret key"

	cc := newConfigureCmd()
	keyInput := strings.NewReader(livemodeKey + "\n")
	actualKey, err := cc.getConfigureAPIKey(keyInput)

	assert.Equal(t, expectedKey, actualKey)
	assert.NotNil(t, err)
	assert.EqualError(t, err, expectedErrorString)
}

func TestDeviceNameInput(t *testing.T) {
	expectedDeviceName := "Bender's Laptop"
	deviceNameInput := strings.NewReader(expectedDeviceName)

	cc := newConfigureCmd()
	actualDeviceName := cc.getConfigureDeviceName(deviceNameInput)

	assert.Equal(t, expectedDeviceName, actualDeviceName)
}

func TestDeviceNameAutoDetect(t *testing.T) {
	hostName, _ := os.Hostname()
	deviceNameInput := strings.NewReader("")

	cc := newConfigureCmd()
	actualDeviceName := cc.getConfigureDeviceName(deviceNameInput)

	assert.Equal(t, hostName, actualDeviceName)
}
