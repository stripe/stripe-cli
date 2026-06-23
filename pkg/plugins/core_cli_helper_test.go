package plugins

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestEcho(t *testing.T) {
	ctx := context.Background()
	coreCLIHelper := NewCoreCLIHelper(ctx, nil, afero.NewMemMapFs())
	output, err := coreCLIHelper.Echo("test")
	require.NoError(t, err)
	require.Equal(t, "test", output)
}

func TestSendAnalytics(t *testing.T) {
	// Test with no telemetry client in context (should not error)
	ctx := context.Background()
	coreCLIHelper := NewCoreCLIHelper(ctx, nil, afero.NewMemMapFs())
	err := coreCLIHelper.SendAnalytics("test_event", "test_value")
	require.NoError(t, err)
}

type fakeKeyringGetResult struct {
	data []byte
	err  error
}

type fakeKeyring struct {
	getResults  []fakeKeyringGetResult
	getCalls    int
	setErr      error
	setCalls    int
	lastSetKey  string
	lastSetData []byte
	keys        []string
	removeErr   error
	removeCalls int
	removedKey  string
}

func (f *fakeKeyring) Get(key string) ([]byte, error) {
	f.getCalls++
	if len(f.getResults) == 0 {
		return nil, keyring.ErrKeyNotFound
	}

	index := f.getCalls - 1
	if index >= len(f.getResults) {
		index = len(f.getResults) - 1
	}

	result := f.getResults[index]
	return result.data, result.err
}

func (f *fakeKeyring) Set(key string, data []byte, description string) error {
	f.setCalls++
	f.lastSetKey = key
	f.lastSetData = data
	return f.setErr
}

func (f *fakeKeyring) Remove(key string) error {
	f.removeCalls++
	f.removedKey = key
	return f.removeErr
}

func (f *fakeKeyring) Keys() ([]string, error) {
	return f.keys, nil
}

func stubKeychainVisibilityClock(t *testing.T) func(time.Duration) {
	t.Helper()

	originalNow := keychainVisibilityNow
	currentTime := time.Unix(0, 0)

	keychainVisibilityNow = func() time.Time {
		return currentTime
	}

	t.Cleanup(func() {
		keychainVisibilityNow = originalNow
	})

	return func(d time.Duration) {
		currentTime = currentTime.Add(d)
	}
}

func resetPendingKeychainValues(t *testing.T) {
	t.Helper()

	keychainVisibilityPendingMu.Lock()
	originalPendingValues := keychainVisibilityPendingValues
	keychainVisibilityPendingValues = map[string]pendingKeychainValue{}
	keychainVisibilityPendingMu.Unlock()

	t.Cleanup(func() {
		keychainVisibilityPendingMu.Lock()
		keychainVisibilityPendingValues = originalPendingValues
		keychainVisibilityPendingMu.Unlock()
	})
}

func TestKeychainGetPasswordReturnsNotFoundWithoutRetry(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
			{data: []byte("sk_test_123")},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("missing.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainGetPasswordReturnsUnexpectedError(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	expectedErr := errors.New("boom")
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: expectedErr},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("broken.key")
	require.ErrorIs(t, err, expectedErr)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainSetPasswordMakesRecentWriteVisibleWhenKeychainHasNotCaughtUp(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 0, ring.getCalls)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "sk_test_123", value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainGetPasswordPrefersRecentWriteOverStaleVisibleValue(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{data: []byte("sk_test_old")},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "sk_test_123", value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainGetPasswordClearsRecentWriteOnceKeychainMatches(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{data: []byte("sk_test_123")},
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "sk_test_123", value)

	value, found, err = coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 2, ring.getCalls)
}

func TestKeychainGetPasswordDoesNotReturnRecentWriteAfterExpiry(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	resetPendingKeychainValues(t)
	advanceClock := stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)

	advanceClock(200 * time.Millisecond)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainSetPasswordDoesNotRememberRecentWriteWhenDisabled(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = false
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 0, ring.getCalls)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainSetPasswordReturnsSetError(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	expectedErr := errors.New("boom")
	ring := &fakeKeyring{
		setErr: expectedErr,
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 0, ring.getCalls)
}

func TestKeychainDeletePasswordClearsRecentWrite(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		keys: []string{"test.key"},
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	resetPendingKeychainValues(t)
	stubKeychainVisibilityClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)

	deleted, err := coreCLIHelper.KeychainDeletePassword("test.key")
	require.NoError(t, err)
	require.True(t, deleted)
	require.Equal(t, 1, ring.removeCalls)
	require.Equal(t, "test.key", ring.removedKey)

	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestSendAnalyticsWithTelemetryClient(t *testing.T) {
	// Test with a NoOp telemetry client
	ctx := context.Background()
	telemetryClient := &stripe.NoOpTelemetryClient{}
	ctx = stripe.WithTelemetryClient(ctx, telemetryClient)

	coreCLIHelper := NewCoreCLIHelper(ctx, nil, afero.NewMemMapFs())
	err := coreCLIHelper.SendAnalytics("test_event", "test_value")
	require.NoError(t, err)
}
