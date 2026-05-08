package plugins

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/99designs/keyring"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
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
	item keyring.Item
	err  error
}

type fakeKeyring struct {
	getResults  []fakeKeyringGetResult
	getCalls    int
	setErr      error
	setCalls    int
	lastSetItem keyring.Item
}

func (f *fakeKeyring) Get(key string) (keyring.Item, error) {
	f.getCalls++
	if len(f.getResults) == 0 {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}

	index := f.getCalls - 1
	if index >= len(f.getResults) {
		index = len(f.getResults) - 1
	}

	result := f.getResults[index]
	return result.item, result.err
}

func (f *fakeKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (f *fakeKeyring) Set(item keyring.Item) error {
	f.setCalls++
	f.lastSetItem = item
	return f.setErr
}

func (f *fakeKeyring) Remove(key string) error {
	return nil
}

func (f *fakeKeyring) Keys() ([]string, error) {
	return nil, nil
}

func stubKeychainRetryClock(t *testing.T) {
	t.Helper()

	originalNow := keychainVisibilityNow
	originalSleep := keychainVisibilitySleep
	currentTime := time.Unix(0, 0)

	keychainVisibilityNow = func() time.Time {
		return currentTime
	}
	keychainVisibilitySleep = func(d time.Duration) {
		currentTime = currentTime.Add(d)
	}

	t.Cleanup(func() {
		keychainVisibilityNow = originalNow
		keychainVisibilitySleep = originalSleep
	})
}

func TestKeychainGetPasswordRetriesTransientNotFound(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
			{err: keyring.ErrKeyNotFound},
			{item: keyring.Item{Data: []byte("sk_test_123")}},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("test.key")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "sk_test_123", value)
	require.Equal(t, 3, ring.getCalls)
}

func TestKeychainGetPasswordReturnsNotFoundAfterRetryWindow(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("missing.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 3, ring.getCalls)
}

func TestKeychainGetPasswordReturnsUnexpectedError(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	expectedErr := errors.New("boom")
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: expectedErr},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("broken.key")
	require.ErrorIs(t, err, expectedErr)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainGetPasswordReturnsNotFoundWithoutRetryWhenDisabled(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = false
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 300 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	value, found, err := coreCLIHelper.KeychainGetPassword("missing.key")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, value)
	require.Equal(t, 1, ring.getCalls)
}

func TestKeychainSetPasswordVerifiesVisibleValueAfterWrite(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
			{item: keyring.Item{Data: []byte("sk_test_123")}},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, keyring.Item{
		Key:   "test.key",
		Data:  []byte("sk_test_123"),
		Label: "test.key",
	}, ring.lastSetItem)
	require.Equal(t, 2, ring.getCalls)
}

func TestKeychainSetPasswordReturnsErrorWhenWrittenValueStaysInvisible(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.EqualError(t, err, `keychain value for "test.key" not visible after write`)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 3, ring.getCalls)
}

func TestKeychainSetPasswordReturnsErrorWhenVisibleValueDoesNotMatch(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{item: keyring.Item{Data: []byte("sk_test_old")}},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = true
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.EqualError(t, err, `keychain value for "test.key" did not match after write`)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 3, ring.getCalls)
}

func TestKeychainSetPasswordSkipsVerificationWhenRetryDisabled(t *testing.T) {
	originalKeyRing := config.KeyRing
	originalEnabled := keychainVisibilityRetryEnabled
	originalInterval := keychainVisibilityRetryInterval
	originalTimeout := keychainVisibilityRetryTimeout
	ring := &fakeKeyring{
		getResults: []fakeKeyringGetResult{
			{err: keyring.ErrKeyNotFound},
		},
	}

	config.KeyRing = ring
	keychainVisibilityRetryEnabled = false
	keychainVisibilityRetryInterval = 100 * time.Millisecond
	keychainVisibilityRetryTimeout = 200 * time.Millisecond
	stubKeychainRetryClock(t)
	t.Cleanup(func() {
		config.KeyRing = originalKeyRing
		keychainVisibilityRetryEnabled = originalEnabled
		keychainVisibilityRetryInterval = originalInterval
		keychainVisibilityRetryTimeout = originalTimeout
	})

	coreCLIHelper := NewCoreCLIHelper(context.Background(), nil, afero.NewMemMapFs())
	err := coreCLIHelper.KeychainSetPassword("test.key", "sk_test_123")
	require.NoError(t, err)
	require.Equal(t, 1, ring.setCalls)
	require.Equal(t, 0, ring.getCalls)
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
