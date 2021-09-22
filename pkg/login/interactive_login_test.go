package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testAccountName = "test-account-name"

func TestDisplayName(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}
	account.Settings.Dashboard.DisplayName = testAccountName

	displayName, err := getDisplayName(context.Background(), account, "", "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		testAccountName,
		displayName,
	)
}

func TestDisplayNameNoName(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}

	displayName, err := getDisplayName(context.Background(), account, "", "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"",
		displayName,
	)
}

func TestDisplayNameGetAccount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}
		account.Settings.Dashboard.DisplayName = testAccountName

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	displayName, err := getDisplayName(context.Background(), nil, ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		testAccountName,
		displayName,
	)
}

func TestDisplayNameGetAccountNoName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	displayName, err := getDisplayName(context.Background(), nil, ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"",
		displayName,
	)
}

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
