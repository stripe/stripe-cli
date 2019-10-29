package login

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyInput(t *testing.T) {
	expectedKey := "sk_test_foo1234"

	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := getConfigureAPIKey(keyInput)

	require.Equal(t, expectedKey, actualKey)
	require.NoError(t, err)
}

func TestAPIKeyInputEmpty(t *testing.T) {
	expectedKey := ""
	expectedErrorString := "API key is required, please provide your API key"

	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := getConfigureAPIKey(keyInput)

	require.Equal(t, expectedKey, actualKey)
	require.NotNil(t, err)
	require.EqualError(t, err, expectedErrorString)
}

func TestDeviceNameInput(t *testing.T) {
	expectedDeviceName := "Bender's Laptop"
	deviceNameInput := strings.NewReader(expectedDeviceName)

	actualDeviceName := getConfigureDeviceName(deviceNameInput)

	require.Equal(t, expectedDeviceName, actualDeviceName)
}

func TestDeviceNameAutoDetect(t *testing.T) {
	hostName, _ := os.Hostname()
	deviceNameInput := strings.NewReader("")

	actualDeviceName := getConfigureDeviceName(deviceNameInput)

	require.Equal(t, hostName, actualDeviceName)
}
