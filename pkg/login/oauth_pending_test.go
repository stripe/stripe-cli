package login

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestOauthContinuationDeadline(t *testing.T) {
	issuedAt := time.Now().Add(-1 * time.Minute)
	cont := &oauthContinuation{IssuedAt: issuedAt, ExpiresIn: 1800}
	assert.WithinDuration(t, issuedAt.Add(1800*time.Second), cont.deadline(), time.Second)
}

func TestOauthContinuationDeadline_MinimumTenMinutes(t *testing.T) {
	issuedAt := time.Now()
	cont := &oauthContinuation{IssuedAt: issuedAt, ExpiresIn: 30}
	assert.WithinDuration(t, issuedAt.Add(10*time.Minute), cont.deadline(), time.Second)
}

func TestSaveAndLoadPendingDeviceAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	issuedAt := time.Now().Truncate(time.Second)
	cont := &oauthContinuation{
		DeviceCode:      "device-code",
		Interval:        5,
		ExpiresIn:       300,
		AccessBaseURL:   QAAccessBaseURL,
		VerificationURI: "https://qa-access.stripe.com/verify",
		UserCode:        "ABCD-EFGH",
		IssuedAt:        issuedAt,
	}
	require.NoError(t, savePendingDeviceAuth(cont))

	loaded, err := loadPendingDeviceAuth()
	require.NoError(t, err)
	assert.Equal(t, cont.DeviceCode, loaded.DeviceCode)
	assert.Equal(t, cont.VerificationURI, loaded.VerificationURI)
	assert.Equal(t, cont.UserCode, loaded.UserCode)
	assert.True(t, cont.IssuedAt.Equal(loaded.IssuedAt))

	clearPendingDeviceAuth()
	_, err = loadPendingDeviceAuth()
	require.Error(t, err)
}

func TestSavePendingReauthAccounts_RepairsPreexistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// A file seeded before the CLI runs keeps the umask's 0644 under a bare
	// os.WriteFile; the save must leave it owner-only.
	path := pendingReauthAccountsPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0644))

	require.NoError(t, savePendingReauthAccounts([]config.AuthorizedAccount{}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSavePendingDeviceAuth_RepairsPreexistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := pendingDeviceAuthPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0644))

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{DeviceCode: "code", AccessBaseURL: QAAccessBaseURL, IssuedAt: time.Now(), ExpiresIn: 300}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSavePendingDeviceAuth_OverwritesPrevious(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{DeviceCode: "first", AccessBaseURL: QAAccessBaseURL, IssuedAt: time.Now(), ExpiresIn: 300}))
	require.NoError(t, savePendingDeviceAuth(&oauthContinuation{DeviceCode: "second", AccessBaseURL: QAAccessBaseURL, IssuedAt: time.Now(), ExpiresIn: 300}))

	loaded, err := loadPendingDeviceAuth()
	require.NoError(t, err)
	assert.Equal(t, "second", loaded.DeviceCode)
}
